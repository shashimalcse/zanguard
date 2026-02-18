package tenant

import (
	"context"
	"errors"
	"net/http"
)

// TenantResolver resolves a tenant ID from a request.
type TenantResolver interface {
	Resolve(ctx context.Context, r *http.Request) (string, error)
}

// ResolverChain tries each resolver in order until one succeeds.
type ResolverChain struct {
	resolvers []TenantResolver
}

// NewResolverChain creates a resolver chain from the given resolvers.
func NewResolverChain(resolvers ...TenantResolver) *ResolverChain {
	return &ResolverChain{resolvers: resolvers}
}

// Resolve tries each resolver in order, returning the first successful result.
func (c *ResolverChain) Resolve(ctx context.Context, r *http.Request) (string, error) {
	var errs []error
	for _, resolver := range c.resolvers {
		id, err := resolver.Resolve(ctx, r)
		if err == nil && id != "" {
			return id, nil
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return "", errors.Join(errs...)
	}
	return "", errors.New("no tenant resolver matched")
}

// StaticResolver always returns the configured tenant ID.
// Useful for single-tenant embedded mode (Phase 1).
type StaticResolver struct {
	TenantID string
}

// Resolve returns the static tenant ID.
func (s *StaticResolver) Resolve(ctx context.Context, r *http.Request) (string, error) {
	if s.TenantID == "" {
		return "", errors.New("static resolver: no tenant ID configured")
	}
	return s.TenantID, nil
}

// HeaderResolver extracts the tenant ID from an HTTP header.
type HeaderResolver struct {
	Header string
}

// Resolve extracts the tenant from the configured header.
func (h *HeaderResolver) Resolve(ctx context.Context, r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("header resolver: no request")
	}
	v := r.Header.Get(h.Header)
	if v == "" {
		return "", nil
	}
	return v, nil
}
