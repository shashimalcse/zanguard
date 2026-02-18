package storage

import (
	"context"
	"errors"
	"io"

	"zanguard/pkg/model"
)

// Sentinel errors for storage operations.
var (
	ErrTenantNotFound  = errors.New("tenant not found")
	ErrTenantDeleted   = errors.New("tenant has been deleted")
	ErrTenantSuspended = errors.New("tenant is suspended")
	ErrQuotaExceeded   = errors.New("tenant quota exceeded")
	ErrDuplicateTuple  = errors.New("tuple already exists")
	ErrNotFound        = errors.New("not found")
)

// TupleStore is the primary storage interface. All data operations are
// automatically scoped to the tenant extracted from ctx.
// Tenant management methods take an explicit tenantID parameter.
type TupleStore interface {
	// Tenant management (explicit tenantID, no ctx tenant required)
	CreateTenant(ctx context.Context, tenant *model.Tenant) error
	GetTenant(ctx context.Context, tenantID string) (*model.Tenant, error)
	UpdateTenant(ctx context.Context, tenant *model.Tenant) error
	ListTenants(ctx context.Context, filter *model.TenantFilter) ([]*model.Tenant, error)

	// Core CRUD (tenant derived from ctx via model.TenantFromContext)
	WriteTuple(ctx context.Context, tuple *model.RelationTuple) error
	WriteTuples(ctx context.Context, tuples []*model.RelationTuple) error
	DeleteTuple(ctx context.Context, tuple *model.RelationTuple) error
	ReadTuples(ctx context.Context, filter *model.TupleFilter) ([]*model.RelationTuple, error)

	// Zanzibar lookups (all tenant-scoped via ctx)
	// CheckDirect checks if a direct relation tuple exists.
	CheckDirect(ctx context.Context, objectType, objectID, relation, subjectType, subjectID string) (bool, error)
	ListRelatedObjects(ctx context.Context, objectType, objectID, relation string) ([]*model.ObjectRef, error)
	ListSubjects(ctx context.Context, objectType, objectID, relation string) ([]*model.SubjectRef, error)
	Expand(ctx context.Context, objectType, objectID, relation string) (*model.SubjectTree, error)

	// Cross-tenant subject lookup (explicit tenant parameter)
	CheckDirectCrossTenant(ctx context.Context, targetTenantID, objectType, objectID, relation, subjectType, subjectID string) (bool, error)

	// Attributes (tenant-scoped)
	GetObjectAttributes(ctx context.Context, objectType, objectID string) (map[string]any, error)
	SetObjectAttributes(ctx context.Context, objectType, objectID string, attrs map[string]any) error
	GetSubjectAttributes(ctx context.Context, subjectType, subjectID string) (map[string]any, error)
	SetSubjectAttributes(ctx context.Context, subjectType, subjectID string, attrs map[string]any) error

	// Changelog (tenant-scoped)
	AppendChangelog(ctx context.Context, entry *model.ChangelogEntry) error
	ReadChangelog(ctx context.Context, sinceSeq uint64, limit int) ([]*model.ChangelogEntry, error)
	LatestSequence(ctx context.Context) (uint64, error)

	// Tenant data operations
	CountTuples(ctx context.Context) (int64, error)
	PurgeTenantData(ctx context.Context) error
	ExportTenantSnapshot(ctx context.Context, w io.Writer) error
}
