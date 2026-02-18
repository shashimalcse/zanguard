package model

import (
	"context"
	"errors"
	"time"
)

// TenantStatus represents the lifecycle state of a tenant.
type TenantStatus string

const (
	TenantPending   TenantStatus = "pending"
	TenantActive    TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
	TenantDeleted   TenantStatus = "deleted"
)

// SchemaMode determines how a tenant's schema is sourced.
type SchemaMode string

const (
	SchemaOwn       SchemaMode = "own"
	SchemaShared    SchemaMode = "shared"
	SchemaInherited SchemaMode = "inherited"
)

// Tenant represents a single tenant in the system.
type Tenant struct {
	ID              string            `json:"id"`
	DisplayName     string            `json:"display_name"`
	ParentTenantID  string            `json:"parent_tenant_id,omitempty"`
	Status          TenantStatus      `json:"status"`
	SchemaMode      SchemaMode        `json:"schema_mode"`
	SharedSchemaRef string            `json:"shared_schema_ref,omitempty"`
	Config          TenantConfig      `json:"config"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// IsWritable returns true if the tenant can accept write operations.
func (t *Tenant) IsWritable() bool {
	return t.Status == TenantActive
}

// IsReadable returns true if the tenant can serve read operations.
func (t *Tenant) IsReadable() bool {
	return t.Status == TenantActive || t.Status == TenantSuspended
}

// TenantConfig holds per-tenant configuration and quotas.
type TenantConfig struct {
	MaxTuples         int64          `json:"max_tuples" yaml:"max_tuples"`
	MaxRequestsPerSec int            `json:"max_requests_per_sec" yaml:"max_requests_per_sec"`
	CacheTTLOverride  *time.Duration `json:"cache_ttl_override,omitempty" yaml:"cache_ttl_override,omitempty"`
	AllowedObjectTypes []string      `json:"allowed_object_types,omitempty" yaml:"allowed_object_types,omitempty"`
	RetentionDays     int            `json:"retention_days" yaml:"retention_days"`
	SyncEnabled       bool           `json:"sync_enabled" yaml:"sync_enabled"`
	WebhookURL        string         `json:"webhook_url,omitempty" yaml:"webhook_url,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// TenantContext is injected into every request context for downstream access.
type TenantContext struct {
	TenantID   string
	Tenant     *Tenant
	SchemaHash string
	Config     *TenantConfig
}

// unexported context key type to avoid collisions.
type tenantContextKeyType struct{}

var tenantContextKey = tenantContextKeyType{}

// WithTenantContext stores a TenantContext in the given context.
func WithTenantContext(ctx context.Context, tc *TenantContext) context.Context {
	return context.WithValue(ctx, tenantContextKey, tc)
}

// TenantFromContext retrieves the TenantContext from a context.
// Returns nil if not present.
func TenantFromContext(ctx context.Context) *TenantContext {
	tc, _ := ctx.Value(tenantContextKey).(*TenantContext)
	return tc
}

// MustTenantFromContext retrieves the TenantContext or panics.
func MustTenantFromContext(ctx context.Context) *TenantContext {
	tc := TenantFromContext(ctx)
	if tc == nil {
		panic("tenant context not found in context — missing tenant middleware?")
	}
	return tc
}

// ErrNoTenantContext is returned when a tenant context is required but not present.
var ErrNoTenantContext = errors.New("no tenant context in request")
