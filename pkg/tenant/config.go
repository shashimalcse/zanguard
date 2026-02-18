package tenant

import "zanguard/pkg/model"

// DefaultTenantConfig returns the default configuration for a new tenant.
func DefaultTenantConfig() model.TenantConfig {
	return model.TenantConfig{
		MaxTuples:         1_000_000,
		MaxRequestsPerSec: 1000,
		RetentionDays:     30,
		SyncEnabled:       true,
	}
}

// MergeConfig merges per-tenant config with defaults, returning resolved config.
// If tenant has zero values, defaults are used.
func MergeConfig(t *model.Tenant, defaults *model.TenantConfig) *model.TenantConfig {
	base := DefaultTenantConfig()
	if defaults != nil {
		base = *defaults
	}

	cfg := t.Config

	if cfg.MaxTuples == 0 {
		cfg.MaxTuples = base.MaxTuples
	}
	if cfg.MaxRequestsPerSec == 0 {
		cfg.MaxRequestsPerSec = base.MaxRequestsPerSec
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = base.RetentionDays
	}
	if cfg.CacheTTLOverride == nil {
		cfg.CacheTTLOverride = base.CacheTTLOverride
	}

	return &cfg
}
