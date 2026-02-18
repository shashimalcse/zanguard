package tenant

import (
	"context"
	"fmt"

	"zanguard/pkg/model"
	"zanguard/pkg/storage"
)

// BuildContext loads the tenant from the store and returns a context
// enriched with the TenantContext.
func BuildContext(ctx context.Context, store storage.TupleStore, tenantID string) (context.Context, error) {
	t, err := store.GetTenant(ctx, tenantID)
	if err != nil {
		return ctx, fmt.Errorf("build tenant context for %q: %w", tenantID, err)
	}

	cfg := MergeConfig(t, nil)
	tc := &model.TenantContext{
		TenantID: tenantID,
		Tenant:   t,
		Config:   cfg,
	}
	return model.WithTenantContext(ctx, tc), nil
}
