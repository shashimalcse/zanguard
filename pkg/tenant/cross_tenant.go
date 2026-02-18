package tenant

import (
	"context"
	"errors"
)

// CrossTenantValidator validates cross-tenant references.
// Phase 1: cross-tenant references are disabled.
type CrossTenantValidator struct {
	enabled bool
}

// NewCrossTenantValidator creates a validator. In Phase 1, enabled=false.
func NewCrossTenantValidator(enabled bool) *CrossTenantValidator {
	return &CrossTenantValidator{enabled: enabled}
}

// ErrCrossTenantDisabled is returned when a cross-tenant reference is attempted
// but cross-tenant access is disabled.
var ErrCrossTenantDisabled = errors.New("cross-tenant references are disabled")

// Validate checks whether a cross-tenant reference is permitted.
func (v *CrossTenantValidator) Validate(ctx context.Context, fromTenantID, toTenantID string) error {
	if !v.enabled {
		return ErrCrossTenantDisabled
	}
	// Phase 2+: check cross_tenant_grants table
	return nil
}
