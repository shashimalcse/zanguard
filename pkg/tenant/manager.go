package tenant

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"zanguard/pkg/model"
	"zanguard/pkg/storage"
)

// tenantIDRegexp validates tenant IDs: lowercase alphanumeric + hyphens, 3-128 chars.
var tenantIDRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,126}[a-z0-9]$`)

// Manager provides tenant CRUD and lifecycle operations.
type Manager struct {
	store storage.TupleStore
}

// NewManager creates a new Manager with the given store.
func NewManager(store storage.TupleStore) *Manager {
	return &Manager{store: store}
}

// ValidateTenantID returns an error if the tenant ID is invalid.
func ValidateTenantID(id string) error {
	if !tenantIDRegexp.MatchString(id) {
		return fmt.Errorf("invalid tenant ID %q: must match ^[a-z0-9][a-z0-9-]{1,126}[a-z0-9]$", id)
	}
	return nil
}

// Create creates a new tenant in pending state.
func (m *Manager) Create(ctx context.Context, id, displayName string, mode model.SchemaMode) (*model.Tenant, error) {
	if err := ValidateTenantID(id); err != nil {
		return nil, err
	}
	if displayName == "" {
		displayName = id
	}
	t := &model.Tenant{
		ID:          id,
		DisplayName: displayName,
		Status:      model.TenantPending,
		SchemaMode:  mode,
		Config:      DefaultTenantConfig(),
		CreatedAt:   time.Now().UTC(),
	}
	if err := m.store.CreateTenant(ctx, t); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return t, nil
}

// Get retrieves a tenant by ID.
func (m *Manager) Get(ctx context.Context, tenantID string) (*model.Tenant, error) {
	t, err := m.store.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Activate transitions a pending or suspended tenant to active.
func (m *Manager) Activate(ctx context.Context, tenantID string) error {
	t, err := m.store.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if t.Status == model.TenantDeleted {
		return fmt.Errorf("cannot activate deleted tenant %q", tenantID)
	}
	t.Status = model.TenantActive
	return m.store.UpdateTenant(ctx, t)
}

// Suspend transitions an active tenant to suspended (read-only).
func (m *Manager) Suspend(ctx context.Context, tenantID string) error {
	t, err := m.store.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if t.Status != model.TenantActive {
		return fmt.Errorf("can only suspend active tenants, current status: %s", t.Status)
	}
	t.Status = model.TenantSuspended
	return m.store.UpdateTenant(ctx, t)
}

// Delete soft-deletes a tenant. Data is retained per retention policy.
func (m *Manager) Delete(ctx context.Context, tenantID string) error {
	t, err := m.store.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if t.Status == model.TenantDeleted {
		return fmt.Errorf("tenant %q is already deleted", tenantID)
	}
	t.Status = model.TenantDeleted
	return m.store.UpdateTenant(ctx, t)
}

// List returns tenants matching the given filter.
func (m *Manager) List(ctx context.Context, filter *model.TenantFilter) ([]*model.Tenant, error) {
	return m.store.ListTenants(ctx, filter)
}
