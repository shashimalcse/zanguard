package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	tuples    map[string][]*model.RelationTuple         // tenantID → tuples
	objAttrs  map[string]map[string]map[string]any      // tenantID → "type:id" → attrs
	subAttrs  map[string]map[string]map[string]any      // tenantID → "type:id" → attrs
	changelog map[string][]*model.ChangelogEntry        // tenantID → entries
	seqCounter map[string]uint64                        // tenantID → next sequence number
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
	return result, nil
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

	// Uniqueness check
	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == tuple.ObjectType &&
			t.ObjectID == tuple.ObjectID &&
			t.Relation == tuple.Relation &&
			t.SubjectType == tuple.SubjectType &&
			t.SubjectID == tuple.SubjectID &&
			t.SubjectRelation == tuple.SubjectRelation {
			return storage.ErrDuplicateTuple
		}
	}

	now := time.Now().UTC()
	t := *tuple
	t.TenantID = tenantID
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	s.tuples[tenantID] = append(s.tuples[tenantID], &t)
	return nil
}

func (s *Store) WriteTuples(ctx context.Context, tuples []*model.RelationTuple) error {
	for _, t := range tuples {
		if err := s.WriteTuple(ctx, t); err != nil {
			return err
		}
	}
	return nil
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
	for _, t := range s.tuples[tenantID] {
		if matchesTupleFilter(t, filter) {
			cp := *t
			result = append(result, &cp)
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

	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == objectType &&
			t.ObjectID == objectID &&
			t.Relation == relation &&
			t.SubjectType == subjectType &&
			t.SubjectID == subjectID &&
			t.SubjectRelation == "" {
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
	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == objectType && t.ObjectID == objectID && t.Relation == relation {
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
	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == objectType && t.ObjectID == objectID && t.Relation == relation {
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
	for _, t := range s.tuples[tenantID] {
		if t.ObjectType == objectType && t.ObjectID == objectID && t.Relation == relation {
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

	for _, t := range s.tuples[targetTenantID] {
		if t.ObjectType == objectType &&
			t.ObjectID == objectID &&
			t.Relation == relation &&
			t.SubjectType == subjectType &&
			t.SubjectID == subjectID {
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

	key := objectType + ":" + objectID
	if attrs, ok := s.objAttrs[tenantID][key]; ok {
		return copyMap(attrs), nil
	}
	return map[string]any{}, nil
}

func (s *Store) SetObjectAttributes(ctx context.Context, objectType, objectID string, attrs map[string]any) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := objectType + ":" + objectID
	s.objAttrs[tenantID][key] = copyMap(attrs)
	return nil
}

func (s *Store) GetSubjectAttributes(ctx context.Context, subjectType, subjectID string) (map[string]any, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := subjectType + ":" + subjectID
	if attrs, ok := s.subAttrs[tenantID][key]; ok {
		return copyMap(attrs), nil
	}
	return map[string]any{}, nil
}

func (s *Store) SetSubjectAttributes(ctx context.Context, subjectType, subjectID string, attrs map[string]any) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := subjectType + ":" + subjectID
	s.subAttrs[tenantID][key] = copyMap(attrs)
	return nil
}

// -- Changelog --

func (s *Store) AppendChangelog(ctx context.Context, entry *model.ChangelogEntry) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

	return int64(len(s.tuples[tenantID])), nil
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

	enc := json.NewEncoder(w)
	for _, t := range s.tuples[tenantID] {
		if err := enc.Encode(t); err != nil {
			return err
		}
	}
	return nil
}

// -- helpers --

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
