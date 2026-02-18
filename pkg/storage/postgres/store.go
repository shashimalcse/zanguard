package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"zanguard/pkg/model"
	"zanguard/pkg/storage"
)

// Option configures the postgres Store.
type Option func(*Store)

// WithMaxConns sets the maximum number of connections in the pool.
func WithMaxConns(n int32) Option {
	return func(s *Store) {
		s.maxConns = n
	}
}

// Store is a PostgreSQL-backed TupleStore.
type Store struct {
	pool     *pgxpool.Pool
	maxConns int32
}

// New creates a new postgres Store connected to the given DSN.
func New(ctx context.Context, dsn string, opts ...Option) (*Store, error) {
	s := &Store{maxConns: 10}
	for _, opt := range opts {
		opt(s)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	cfg.MaxConns = s.maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	s.pool = pool
	return s, nil
}

// Close releases the connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// tenantIDFromCtx extracts the tenant ID from context.
func tenantIDFromCtx(ctx context.Context) (string, error) {
	tc := model.TenantFromContext(ctx)
	if tc == nil {
		return "", model.ErrNoTenantContext
	}
	return tc.TenantID, nil
}

// -- Tenant management --

func (s *Store) CreateTenant(ctx context.Context, tenant *model.Tenant) error {
	configJSON, err := json.Marshal(tenant.Config)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO tenants (id, display_name, parent_tenant_id, status, schema_mode, shared_schema_ref, config, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''), $7, $8, $9)`,
		tenant.ID, tenant.DisplayName, tenant.ParentTenantID,
		string(tenant.Status), string(tenant.SchemaMode), tenant.SharedSchemaRef,
		configJSON, now, now,
	)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

func (s *Store) GetTenant(ctx context.Context, tenantID string) (*model.Tenant, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, display_name, COALESCE(parent_tenant_id, ''), status, schema_mode,
		       COALESCE(shared_schema_ref, ''), config, created_at, updated_at
		FROM tenants WHERE id = $1 AND deleted_at IS NULL`, tenantID)

	var t model.Tenant
	var configJSON []byte
	err := row.Scan(&t.ID, &t.DisplayName, &t.ParentTenantID, &t.Status, &t.SchemaMode,
		&t.SharedSchemaRef, &configJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	if err := json.Unmarshal(configJSON, &t.Config); err != nil {
		return nil, fmt.Errorf("unmarshal tenant config: %w", err)
	}
	return &t, nil
}

func (s *Store) UpdateTenant(ctx context.Context, tenant *model.Tenant) error {
	configJSON, err := json.Marshal(tenant.Config)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE tenants SET display_name=$2, status=$3, schema_mode=$4,
		shared_schema_ref=NULLIF($5,''), config=$6, updated_at=NOW()
		WHERE id=$1`,
		tenant.ID, tenant.DisplayName, string(tenant.Status), string(tenant.SchemaMode),
		tenant.SharedSchemaRef, configJSON,
	)
	return err
}

func (s *Store) ListTenants(ctx context.Context, filter *model.TenantFilter) ([]*model.Tenant, error) {
	query := `SELECT id, display_name, COALESCE(parent_tenant_id,''), status, schema_mode,
		COALESCE(shared_schema_ref,''), config, created_at, updated_at
		FROM tenants WHERE deleted_at IS NULL`
	args := []any{}
	argN := 1

	if filter != nil && filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, string(filter.Status))
		argN++
	}
	if filter != nil && filter.ParentID != "" {
		query += fmt.Sprintf(" AND parent_tenant_id = $%d", argN)
		args = append(args, filter.ParentID)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []*model.Tenant
	for rows.Next() {
		var t model.Tenant
		var configJSON []byte
		if err := rows.Scan(&t.ID, &t.DisplayName, &t.ParentTenantID, &t.Status, &t.SchemaMode,
			&t.SharedSchemaRef, &configJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(configJSON, &t.Config); err != nil {
			return nil, err
		}
		tenants = append(tenants, &t)
	}
	return tenants, rows.Err()
}

// -- Core CRUD --

func (s *Store) WriteTuple(ctx context.Context, tuple *model.RelationTuple) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO relation_tuples
		  (tenant_id, object_type, object_id, relation, subject_type, subject_id, subject_relation,
		   subject_tenant_id, source_system, external_id)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),NULLIF($10,''))
		ON CONFLICT (tenant_id, object_type, object_id, relation, subject_type, subject_id, subject_relation)
		DO UPDATE SET updated_at = NOW()`,
		tenantID, tuple.ObjectType, tuple.ObjectID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
		tuple.SubjectTenantID, tuple.SourceSystem, tuple.ExternalID,
	)
	return err
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
	_, err = s.pool.Exec(ctx, `
		UPDATE relation_tuples SET deleted_at = NOW()
		WHERE tenant_id=$1 AND object_type=$2 AND object_id=$3 AND relation=$4
		  AND subject_type=$5 AND subject_id=$6 AND COALESCE(subject_relation,'')=$7
		  AND deleted_at IS NULL`,
		tenantID, tuple.ObjectType, tuple.ObjectID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
	)
	return err
}

func (s *Store) ReadTuples(ctx context.Context, filter *model.TupleFilter) ([]*model.RelationTuple, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	query := `SELECT tenant_id, object_type, object_id, relation, subject_type, subject_id,
		COALESCE(subject_relation,''), COALESCE(subject_tenant_id,''),
		COALESCE(source_system,''), COALESCE(external_id,''), created_at, updated_at
		FROM relation_tuples WHERE tenant_id=$1 AND deleted_at IS NULL`
	args := []any{tenantID}
	argN := 2

	if filter != nil {
		if filter.ObjectType != "" {
			query += fmt.Sprintf(" AND object_type=$%d", argN)
			args = append(args, filter.ObjectType)
			argN++
		}
		if filter.ObjectID != "" {
			query += fmt.Sprintf(" AND object_id=$%d", argN)
			args = append(args, filter.ObjectID)
			argN++
		}
		if filter.Relation != "" {
			query += fmt.Sprintf(" AND relation=$%d", argN)
			args = append(args, filter.Relation)
			argN++
		}
		if filter.SubjectType != "" {
			query += fmt.Sprintf(" AND subject_type=$%d", argN)
			args = append(args, filter.SubjectType)
			argN++
		}
		if filter.SubjectID != "" {
			query += fmt.Sprintf(" AND subject_id=$%d", argN)
			args = append(args, filter.SubjectID)
			argN++
		}
		if filter.SubjectRelation != "" {
			query += fmt.Sprintf(" AND subject_relation=$%d", argN)
			args = append(args, filter.SubjectRelation)
		}
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tuples []*model.RelationTuple
	for rows.Next() {
		var t model.RelationTuple
		if err := rows.Scan(&t.TenantID, &t.ObjectType, &t.ObjectID, &t.Relation,
			&t.SubjectType, &t.SubjectID, &t.SubjectRelation, &t.SubjectTenantID,
			&t.SourceSystem, &t.ExternalID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tuples = append(tuples, &t)
	}
	return tuples, rows.Err()
}

// -- Zanzibar lookups --

func (s *Store) CheckDirect(ctx context.Context, objectType, objectID, relation, subjectType, subjectID string) (bool, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM relation_tuples
		WHERE tenant_id=$1 AND object_type=$2 AND object_id=$3 AND relation=$4
		  AND subject_type=$5 AND subject_id=$6 AND (subject_relation IS NULL OR subject_relation='')
		  AND deleted_at IS NULL)`,
		tenantID, objectType, objectID, relation, subjectType, subjectID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) ListRelatedObjects(ctx context.Context, objectType, objectID, relation string) ([]*model.ObjectRef, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT subject_type, subject_id FROM relation_tuples
		WHERE tenant_id=$1 AND object_type=$2 AND object_id=$3 AND relation=$4 AND deleted_at IS NULL`,
		tenantID, objectType, objectID, relation,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []*model.ObjectRef
	for rows.Next() {
		var r model.ObjectRef
		if err := rows.Scan(&r.Type, &r.ID); err != nil {
			return nil, err
		}
		refs = append(refs, &r)
	}
	return refs, rows.Err()
}

func (s *Store) ListSubjects(ctx context.Context, objectType, objectID, relation string) ([]*model.SubjectRef, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT subject_type, subject_id, COALESCE(subject_relation,'') FROM relation_tuples
		WHERE tenant_id=$1 AND object_type=$2 AND object_id=$3 AND relation=$4 AND deleted_at IS NULL`,
		tenantID, objectType, objectID, relation,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []*model.SubjectRef
	for rows.Next() {
		var r model.SubjectRef
		if err := rows.Scan(&r.Type, &r.ID, &r.Relation); err != nil {
			return nil, err
		}
		refs = append(refs, &r)
	}
	return refs, rows.Err()
}

func (s *Store) Expand(ctx context.Context, objectType, objectID, relation string) (*model.SubjectTree, error) {
	subjects, err := s.ListSubjects(ctx, objectType, objectID, relation)
	if err != nil {
		return nil, err
	}
	root := &model.SubjectTree{
		Subject: &model.SubjectRef{Type: objectType, ID: objectID, Relation: relation},
	}
	for _, sub := range subjects {
		root.Children = append(root.Children, &model.SubjectTree{Subject: sub})
	}
	return root, nil
}

func (s *Store) CheckDirectCrossTenant(ctx context.Context, targetTenantID, objectType, objectID, relation, subjectType, subjectID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM relation_tuples
		WHERE tenant_id=$1 AND object_type=$2 AND object_id=$3 AND relation=$4
		  AND subject_type=$5 AND subject_id=$6 AND deleted_at IS NULL)`,
		targetTenantID, objectType, objectID, relation, subjectType, subjectID,
	).Scan(&exists)
	return exists, err
}

// -- Attributes --

func (s *Store) GetObjectAttributes(ctx context.Context, objectType, objectID string) (map[string]any, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	var attrsJSON []byte
	err = s.pool.QueryRow(ctx, `
		SELECT attributes FROM object_attributes WHERE tenant_id=$1 AND object_type=$2 AND object_id=$3`,
		tenantID, objectType, objectID,
	).Scan(&attrsJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var attrs map[string]any
	if err := json.Unmarshal(attrsJSON, &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

func (s *Store) SetObjectAttributes(ctx context.Context, objectType, objectID string, attrs map[string]any) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}
	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO object_attributes (tenant_id, object_type, object_id, attributes)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (tenant_id, object_type, object_id)
		DO UPDATE SET attributes=$4, updated_at=NOW()`,
		tenantID, objectType, objectID, attrsJSON,
	)
	return err
}

func (s *Store) GetSubjectAttributes(ctx context.Context, subjectType, subjectID string) (map[string]any, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	var attrsJSON []byte
	err = s.pool.QueryRow(ctx, `
		SELECT attributes FROM subject_attributes WHERE tenant_id=$1 AND subject_type=$2 AND subject_id=$3`,
		tenantID, subjectType, subjectID,
	).Scan(&attrsJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var attrs map[string]any
	if err := json.Unmarshal(attrsJSON, &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

func (s *Store) SetSubjectAttributes(ctx context.Context, subjectType, subjectID string, attrs map[string]any) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}
	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO subject_attributes (tenant_id, subject_type, subject_id, attributes)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (tenant_id, subject_type, subject_id)
		DO UPDATE SET attributes=$4, updated_at=NOW()`,
		tenantID, subjectType, subjectID, attrsJSON,
	)
	return err
}

// -- Changelog --

func (s *Store) AppendChangelog(ctx context.Context, entry *model.ChangelogEntry) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}
	metaJSON, _ := json.Marshal(entry.Metadata)
	_, err = s.pool.Exec(ctx, `
		INSERT INTO changelog (tenant_id, operation, object_type, object_id, relation,
		  subject_type, subject_id, subject_relation, actor, source, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,$10,$11)`,
		tenantID, string(entry.Operation),
		entry.Tuple.ObjectType, entry.Tuple.ObjectID, entry.Tuple.Relation,
		entry.Tuple.SubjectType, entry.Tuple.SubjectID, entry.Tuple.SubjectRelation,
		entry.Actor, entry.Source, metaJSON,
	)
	return err
}

func (s *Store) ReadChangelog(ctx context.Context, sinceSeq uint64, limit int) ([]*model.ChangelogEntry, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 1000
	}
	rows, err := s.pool.Query(ctx, `
		SELECT sequence, tenant_id, operation, object_type, object_id, relation,
		  subject_type, subject_id, COALESCE(subject_relation,''), actor, source, created_at
		FROM changelog WHERE tenant_id=$1 AND sequence > $2
		ORDER BY sequence LIMIT $3`,
		tenantID, sinceSeq, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*model.ChangelogEntry
	for rows.Next() {
		var e model.ChangelogEntry
		if err := rows.Scan(&e.Sequence, &e.TenantID, &e.Operation,
			&e.Tuple.ObjectType, &e.Tuple.ObjectID, &e.Tuple.Relation,
			&e.Tuple.SubjectType, &e.Tuple.SubjectID, &e.Tuple.SubjectRelation,
			&e.Actor, &e.Source, &e.Timestamp); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

func (s *Store) LatestSequence(ctx context.Context) (uint64, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return 0, err
	}
	var seq uint64
	err = s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(sequence),0) FROM changelog WHERE tenant_id=$1`, tenantID).Scan(&seq)
	return seq, err
}

// -- Tenant data operations --

func (s *Store) CountTuples(ctx context.Context) (int64, error) {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM relation_tuples WHERE tenant_id=$1 AND deleted_at IS NULL`, tenantID).Scan(&count)
	return count, err
}

func (s *Store) PurgeTenantData(ctx context.Context) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM relation_tuples WHERE tenant_id=$1`, tenantID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM object_attributes WHERE tenant_id=$1`, tenantID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM subject_attributes WHERE tenant_id=$1`, tenantID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM changelog WHERE tenant_id=$1`, tenantID)
	return err
}

func (s *Store) ExportTenantSnapshot(ctx context.Context, w io.Writer) error {
	tenantID, err := tenantIDFromCtx(ctx)
	if err != nil {
		return err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT tenant_id, object_type, object_id, relation, subject_type, subject_id,
		  COALESCE(subject_relation,''), created_at, updated_at
		FROM relation_tuples WHERE tenant_id=$1 AND deleted_at IS NULL`, tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()

	enc := json.NewEncoder(w)
	for rows.Next() {
		var t model.RelationTuple
		if err := rows.Scan(&t.TenantID, &t.ObjectType, &t.ObjectID, &t.Relation,
			&t.SubjectType, &t.SubjectID, &t.SubjectRelation, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return err
		}
		if err := enc.Encode(t); err != nil {
			return err
		}
	}
	return rows.Err()
}
