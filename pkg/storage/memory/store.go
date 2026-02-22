package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"zanguard/pkg/model"
	"zanguard/pkg/storage"
)

// Store is a thread-safe in-memory implementation of storage.TupleStore.
// Tenant data is stored in separate slices/maps for O(1) purge.
type Store struct {
	mu sync.RWMutex

	tenants map[string]*model.Tenant

	// Per-tenant data buckets
	tuples     map[string][]*model.RelationTuple    // tenantID → tuples
	objAttrs   map[string]map[string]map[string]any // tenantID → "type:id" → attrs
	subAttrs   map[string]map[string]map[string]any // tenantID → "type:id" → attrs
	changelog  map[string][]*model.ChangelogEntry   // tenantID → entries
	seqCounter map[string]uint64                    // tenantID → next sequence number
}

// New creates a new in-memory store.
func New() *Store {
	return &Store{
		tenants:    make(map[string]*model.Tenant),
		tuples:     make(map[string][]*model.RelationTuple),
		objAttrs:   make(map[string]map[string]map[string]any),
		subAttrs:   make(map[string]map[string]map[string]any),
		changelog:  make(map[string][]*model.ChangelogEntry),
		seqCounter: make(map[string]uint64),
	}
}

// tenantIDFromCtx extracts tenant ID from context (required for data ops).
func tenantIDFromCtx(ctx context.Context) (string, error) {
	tc := model.TenantFromContext(ctx)
	if tc == nil {
		return "", model.ErrNoTenantContext
	}
	return tc.TenantID, nil
}

// initTenant initializes per-tenant buckets (must hold write lock).
func (s *Store) initTenant(tenantID string) {
	if _, ok := s.tuples[tenantID]; !ok {
		s.tuples[tenantID] = nil
		s.objAttrs[tenantID] = make(map[string]map[string]any)
		s.subAttrs[tenantID] = make(map[string]map[string]any)
		s.changelog[tenantID] = nil
		s.seqCounter[tenantID] = 0
	}
}

// -- Tenant management --

func (s *Store) CreateTenant(ctx context.Context, tenant *model.Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tenants[tenant.ID]; exists {
		return fmt.Errorf("tenant %q already exists", tenant.ID)
	}
	now := time.Now().UTC()
	t := *tenant
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	s.tenants[tenant.ID] = &t
	s.initTenant(tenant.ID)
	return nil
}

func (s *Store) GetTenant(ctx context.Context, tenantID string) (*model.Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tenants[tenantID]
	if !ok {
		return nil, storage.ErrTenantNotFound
	}
	cp := *t
	return &cp, nil
}

func (s *Store) UpdateTenant(ctx context.Context, tenant *model.Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tenants[tenant.ID]; !ok {
		return storage.ErrTenantNotFound
	}
	t := *tenant
	t.UpdatedAt = time.Now().UTC()
	s.tenants[tenant.ID] = &t
	return nil
}

func (s *Store) ListTenants(ctx context.Context, filter *model.TenantFilter) ([]*model.Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Tenant
	for _, t := range s.tenants {
		if filter != nil {
			if filter.Status != "" && t.Status != filter.Status {
				continue
			}
			if filter.ParentID != "" && t.ParentTenantID != filter.ParentID {
				continue
			}
		}
		cp := *t
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return applyTenantFilter(result, filter), nil
}

// -- Core CRUD --

func (s *Store) WriteTuple(ctx context.Context, tuple *model.RelationTuple) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkTenantWritable(tenantID); err != nil {
		return err
	}
	return s.writeTuplesLocked(tenantID, []*model.RelationTuple{tuple})
}

func (s *Store) WriteTuples(ctx context.Context, tuples []*model.RelationTuple) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkTenantWritable(tenantID); err != nil {
		return err
	}
	return s.writeTuplesLocked(tenantID, tuples)
}

func (s *Store) DeleteTuple(ctx context.Context, tuple *model.RelationTuple) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkTenantWritable(tenantID); err != nil {
		return err
	}

	ts := s.tuples[tenantID]
	for i, t := range ts {
		if t.ObjectType == tuple.ObjectType &&
			t.ObjectID == tuple.ObjectID &&
			t.Relation == tuple.Relation &&
			t.SubjectType == tuple.SubjectType &&
			t.SubjectID == tuple.SubjectID &&
			t.SubjectRelation == tuple.SubjectRelation {
			s.tuples[tenantID] = append(ts[:i], ts[i+1:]...)
			s.appendTupleChangelogUnsafe(tenantID, model.ChangeOpDelete, t, "api")
			return nil
		}
	}
	return storage.ErrNotFound
}

func (s *Store) ReadTuples(ctx context.Context, filter *model.TupleFilter) ([]*model.RelationTuple, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	var result []*model.RelationTuple
	now := time.Now().UTC()
	includeExpired := filter != nil && filter.IncludeExpired
	for _, t := range s.tuples[tenantID] {
		if !includeExpired && tupleIsExpiredAt(t, now) {
			continue
		}
		if matchesTupleFilter(t, filter) {
			result = append(result, cloneTuple(t))
		}
	}
	return result, nil
}

// -- Zanzibar lookups --

func (s *Store) CheckDirect(ctx context.Context, objectType, objectID, relation, subjectType, subjectID string) (bool, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return false, err
	}

	now := time.Now().UTC()
	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == objectType &&
			t.ObjectID == objectID &&
			t.Relation == relation &&
			t.SubjectType == subjectType &&
			t.SubjectID == subjectID &&
			t.SubjectRelation == "" &&
			!tupleIsExpiredAt(t, now) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) ListRelatedObjects(ctx context.Context, objectType, objectID, relation string) ([]*model.ObjectRef, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	var result []*model.ObjectRef
	now := time.Now().UTC()
	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == objectType && t.ObjectID == objectID && t.Relation == relation &&
			!tupleIsExpiredAt(t, now) {
			result = append(result, &model.ObjectRef{
				Type: t.SubjectType,
				ID:   t.SubjectID,
			})
		}
	}
	return result, nil
}

func (s *Store) ListSubjects(ctx context.Context, objectType, objectID, relation string) ([]*model.SubjectRef, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	var result []*model.SubjectRef
	now := time.Now().UTC()
	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == objectType && t.ObjectID == objectID && t.Relation == relation &&
			!tupleIsExpiredAt(t, now) {
			result = append(result, &model.SubjectRef{
				Type:     t.SubjectType,
				ID:       t.SubjectID,
				Relation: t.SubjectRelation,
			})
		}
	}
	return result, nil
}

func (s *Store) Expand(ctx context.Context, objectType, objectID, relation string) (*model.SubjectTree, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	root := &model.SubjectTree{
		Subject: &model.SubjectRef{Type: objectType, ID: objectID, Relation: relation},
	}
	now := time.Now().UTC()
	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == objectType && t.ObjectID == objectID && t.Relation == relation &&
			!tupleIsExpiredAt(t, now) {
			root.Children = append(root.Children, &model.SubjectTree{
				Subject: &model.SubjectRef{Type: t.SubjectType, ID: t.SubjectID, Relation: t.SubjectRelation},
			})
		}
	}
	return root, nil
}

func (s *Store) CheckDirectCrossTenant(ctx context.Context, targetTenantID, objectType, objectID, relation, subjectType, subjectID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(targetTenantID); err != nil {
		return false, err
	}

	now := time.Now().UTC()
	for _, t := range s.tuples[targetTenantID] {
		if t.ObjectType == objectType &&
			t.ObjectID == objectID &&
			t.Relation == relation &&
			t.SubjectType == subjectType &&
			t.SubjectID == subjectID &&
			!tupleIsExpiredAt(t, now) {
			return true, nil
		}
	}
	return false, nil
}

// -- Attributes --

func (s *Store) GetObjectAttributes(ctx context.Context, objectType, objectID string) (map[string]any, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	key := objectType + ":" + objectID
	if attrs, ok := s.objAttrs[tenantID][key]; ok {
		return copyMap(attrs), nil
	}
	return nil, storage.ErrNotFound
}

func (s *Store) SetObjectAttributes(ctx context.Context, objectType, objectID string, attrs map[string]any) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkTenantWritable(tenantID); err != nil {
		return err
	}

	key := objectType + ":" + objectID
	s.objAttrs[tenantID][key] = copyMap(attrs)
	return nil
}

func (s *Store) ListObjectAttributes(ctx context.Context, objectType string) ([]*model.ObjectAttributes, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	var result []*model.ObjectAttributes
	for key, attrs := range s.objAttrs[tenantID] {
		idx := strings.Index(key, ":")
		if idx < 0 {
			continue
		}
		objType := key[:idx]
		objID := key[idx+1:]
		if objectType != "" && objType != objectType {
			continue
		}
		result = append(result, &model.ObjectAttributes{
			TenantID:   tenantID,
			ObjectType: objType,
			ObjectID:   objID,
			Attributes: copyMap(attrs),
		})
	}
	return result, nil
}

func (s *Store) GetSubjectAttributes(ctx context.Context, subjectType, subjectID string) (map[string]any, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	key := subjectType + ":" + subjectID
	if attrs, ok := s.subAttrs[tenantID][key]; ok {
		return copyMap(attrs), nil
	}
	return nil, storage.ErrNotFound
}

func (s *Store) SetSubjectAttributes(ctx context.Context, subjectType, subjectID string, attrs map[string]any) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkTenantWritable(tenantID); err != nil {
		return err
	}

	key := subjectType + ":" + subjectID
	s.subAttrs[tenantID][key] = copyMap(attrs)
	return nil
}

func (s *Store) ListSubjectAttributes(ctx context.Context, subjectType string) ([]*model.SubjectAttributes, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	var result []*model.SubjectAttributes
	for key, attrs := range s.subAttrs[tenantID] {
		idx := strings.Index(key, ":")
		if idx < 0 {
			continue
		}
		subType := key[:idx]
		subID := key[idx+1:]
		if subjectType != "" && subType != subjectType {
			continue
		}
		result = append(result, &model.SubjectAttributes{
			TenantID:    tenantID,
			SubjectType: subType,
			SubjectID:   subID,
			Attributes:  copyMap(attrs),
		})
	}
	return result, nil
}

// -- Changelog --

func (s *Store) AppendChangelog(ctx context.Context, entry *model.ChangelogEntry) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkTenantWritable(tenantID); err != nil {
		return err
	}

	s.seqCounter[tenantID]++
	e := *entry
	e.TenantID = tenantID
	e.Sequence = s.seqCounter[tenantID]
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	s.changelog[tenantID] = append(s.changelog[tenantID], &e)
	return nil
}

func (s *Store) ReadChangelog(ctx context.Context, sinceSeq uint64, limit int) ([]*model.ChangelogEntry, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return nil, err
	}

	var result []*model.ChangelogEntry
	for _, e := range s.changelog[tenantID] {
		if e.Sequence > sinceSeq {
			cp := *e
			result = append(result, &cp)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *Store) LatestSequence(ctx context.Context) (uint64, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return 0, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return 0, err
	}

	return s.seqCounter[tenantID], nil
}

// -- Tenant data operations --

func (s *Store) CountTuples(ctx context.Context) (int64, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return 0, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	var count int64
	for _, t := range s.tuples[tenantID] {
		if !tupleIsExpiredAt(t, now) {
			count++
		}
	}
	return count, nil
}

func (s *Store) PurgeTenantData(ctx context.Context) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// O(1) purge — just nil/reset the maps for this tenant
	s.tuples[tenantID] = nil
	s.objAttrs[tenantID] = make(map[string]map[string]any)
	s.subAttrs[tenantID] = make(map[string]map[string]any)
	s.changelog[tenantID] = nil
	s.seqCounter[tenantID] = 0
	return nil
}

func (s *Store) ExportTenantSnapshot(ctx context.Context, w io.Writer) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.checkTenantReadable(tenantID); err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	now := time.Now().UTC()
	for _, t := range s.tuples[tenantID] {
		if tupleIsExpiredAt(t, now) {
			continue
		}
		if err := enc.Encode(t); err != nil {
			return err
		}
	}
	return nil
}

// -- helpers --

func (s *Store) writeTuplesLocked(tenantID string, tuples []*model.RelationTuple) error {
	if len(tuples) == 0 {
		return nil
	}

	existingKeys := make(map[string]struct{}, len(s.tuples[tenantID]))
	expiredIndexes := make(map[string]int)
	now := time.Now().UTC()
	for i, t := range s.tuples[tenantID] {
		key := tupleIdentityKey(t)
		if tupleIsExpiredAt(t, now) {
			if _, ok := expiredIndexes[key]; !ok {
				expiredIndexes[key] = i
			}
			continue
		}
		existingKeys[key] = struct{}{}
	}
	batchKeys := make(map[string]struct{}, len(tuples))
	staged := make([]*model.RelationTuple, len(tuples))

	for i, tuple := range tuples {
		key := tupleIdentityKey(tuple)
		if _, ok := existingKeys[key]; ok {
			return storage.ErrDuplicateTuple
		}
		if _, ok := batchKeys[key]; ok {
			return storage.ErrDuplicateTuple
		}
		batchKeys[key] = struct{}{}

		cp := cloneTuple(tuple)
		cp.TenantID = tenantID
		if cp.CreatedAt.IsZero() {
			cp.CreatedAt = now
		}
		cp.UpdatedAt = now
		staged[i] = cp
	}

	appends := make([]*model.RelationTuple, 0, len(staged))
	for _, t := range staged {
		if idx, ok := expiredIndexes[tupleIdentityKey(t)]; ok {
			s.tuples[tenantID][idx] = t
			continue
		}
		appends = append(appends, t)
	}
	s.tuples[tenantID] = append(s.tuples[tenantID], appends...)
	for _, t := range staged {
		s.appendTupleChangelogUnsafe(tenantID, model.ChangeOpInsert, t, "api")
	}
	return nil
}

func (s *Store) appendTupleChangelogUnsafe(tenantID string, op model.ChangeOp, tuple *model.RelationTuple, source string) {
	s.seqCounter[tenantID]++
	entry := &model.ChangelogEntry{
		Sequence:  s.seqCounter[tenantID],
		TenantID:  tenantID,
		Operation: op,
		Tuple:     *tuple,
		Timestamp: time.Now().UTC(),
		Source:    source,
	}
	s.changelog[tenantID] = append(s.changelog[tenantID], entry)
}

func tupleIdentityKey(t *model.RelationTuple) string {
	return fmt.Sprintf("%s:%s#%s@%s:%s#%s", t.ObjectType, t.ObjectID, t.Relation, t.SubjectType, t.SubjectID, t.SubjectRelation)
}

func applyTenantFilter(tenants []*model.Tenant, filter *model.TenantFilter) []*model.Tenant {
	if filter == nil {
		return tenants
	}
	if filter.Offset >= len(tenants) {
		return []*model.Tenant{}
	}
	start := filter.Offset
	if start < 0 {
		start = 0
	}
	end := len(tenants)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}
	return tenants[start:end]
}

func (s *Store) checkTenantWritable(tenantID string) error {
	t, ok := s.tenants[tenantID]
	if !ok {
		return storage.ErrTenantNotFound
	}
	switch t.Status {
	case model.TenantDeleted:
		return storage.ErrTenantDeleted
	case model.TenantSuspended:
		return storage.ErrTenantSuspended
	case model.TenantPending:
		return storage.ErrTenantSuspended // pending is not writable yet
	}
	return nil
}

func (s *Store) checkTenantReadable(tenantID string) error {
	t, ok := s.tenants[tenantID]
	if !ok {
		return storage.ErrTenantNotFound
	}
	if t.Status == model.TenantDeleted {
		return storage.ErrTenantDeleted
	}
	return nil
}

func matchesTupleFilter(t *model.RelationTuple, f *model.TupleFilter) bool {
	if f == nil {
		return true
	}
	if f.ObjectType != "" && t.ObjectType != f.ObjectType {
		return false
	}
	if f.ObjectID != "" && t.ObjectID != f.ObjectID {
		return false
	}
	if f.Relation != "" && t.Relation != f.Relation {
		return false
	}
	if f.SubjectType != "" && t.SubjectType != f.SubjectType {
		return false
	}
	if f.SubjectID != "" && t.SubjectID != f.SubjectID {
		return false
	}
	if f.SubjectRelation != "" && t.SubjectRelation != f.SubjectRelation {
		return false
	}
	return true
}

func tupleIsExpiredAt(t *model.RelationTuple, now time.Time) bool {
	if t.ExpiresAt == nil {
		return false
	}
	return !t.ExpiresAt.After(now)
}

func copyTimePtr(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	cp := src.UTC()
	return &cp
}

func cloneTuple(src *model.RelationTuple) *model.RelationTuple {
	if src == nil {
		return nil
	}
	cp := *src
	cp.Attributes = copyMap(src.Attributes)
	cp.ExpiresAt = copyTimePtr(src.ExpiresAt)
	return &cp
}

func copyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
