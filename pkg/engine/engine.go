package engine

import (
	"context"
	"fmt"
	"sync"

	"zanguard/pkg/model"
	"zanguard/pkg/schema"
	"zanguard/pkg/storage"
)

// Config holds engine configuration.
type Config struct {
	MaxCheckDepth int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxCheckDepth: 25,
	}
}

// CheckRequest is the input to a permission check.
type CheckRequest struct {
	ObjectType  string
	ObjectID    string
	Permission  string
	SubjectType string
	SubjectID   string
	Context     map[string]any // request-time context (IP, time, etc.)
}

// CheckResult is the output of a permission check.
type CheckResult struct {
	Allowed        bool
	ResolutionPath []string
	Error          error
}

// allow / deny helpers.
func allow(path ...string) *CheckResult { return &CheckResult{Allowed: true, ResolutionPath: path} }
func deny() *CheckResult               { return &CheckResult{Allowed: false} }

// Engine is the core authorization check engine.
type Engine struct {
	store         storage.TupleStore
	schemasMu     sync.RWMutex
	schemas       map[string]*schema.CompiledSchema // tenantID → schema
	sharedSchemas map[string]*schema.CompiledSchema // sharedSchemaRef → schema
	cfg           Config
}

// New creates a new Engine.
func New(store storage.TupleStore, cfg Config) *Engine {
	return &Engine{
		store:         store,
		schemas:       make(map[string]*schema.CompiledSchema),
		sharedSchemas: make(map[string]*schema.CompiledSchema),
		cfg:           cfg,
	}
}

// LoadSchema registers a compiled schema for a tenant.
func (e *Engine) LoadSchema(tenantID string, cs *schema.CompiledSchema) {
	e.schemasMu.Lock()
	defer e.schemasMu.Unlock()
	e.schemas[tenantID] = cs
}

// LoadSharedSchema registers a shared schema by reference name.
func (e *Engine) LoadSharedSchema(ref string, cs *schema.CompiledSchema) {
	e.schemasMu.Lock()
	defer e.schemasMu.Unlock()
	e.sharedSchemas[ref] = cs
}

// schemaForTenant resolves the correct schema based on tenant schema mode.
func (e *Engine) schemaForTenant(tc *model.TenantContext) (*schema.CompiledSchema, error) {
	e.schemasMu.RLock()
	defer e.schemasMu.RUnlock()

	switch tc.Tenant.SchemaMode {
	case model.SchemaOwn:
		cs, ok := e.schemas[tc.TenantID]
		if !ok {
			return nil, fmt.Errorf("no schema loaded for tenant %q", tc.TenantID)
		}
		return cs, nil

	case model.SchemaShared:
		ref := tc.Tenant.SharedSchemaRef
		cs, ok := e.sharedSchemas[ref]
		if !ok {
			return nil, fmt.Errorf("shared schema %q not found", ref)
		}
		return cs, nil

	case model.SchemaInherited:
		// Phase 1: treat inherited same as own (no extension merging yet)
		cs, ok := e.schemas[tc.TenantID]
		if !ok {
			return nil, fmt.Errorf("no schema loaded for tenant %q", tc.TenantID)
		}
		return cs, nil

	default:
		return nil, fmt.Errorf("unknown schema mode: %s", tc.Tenant.SchemaMode)
	}
}

// Check performs a permission check for the given request.
func (e *Engine) Check(ctx context.Context, req *CheckRequest) (*CheckResult, error) {
	tc := model.TenantFromContext(ctx)
	if tc == nil {
		return deny(), model.ErrNoTenantContext
	}

	if tc.Tenant.Status == model.TenantDeleted {
		return deny(), storage.ErrTenantDeleted
	}

	if !tc.Tenant.IsReadable() {
		return deny(), fmt.Errorf("tenant %q is not readable (status: %s)", tc.TenantID, tc.Tenant.Status)
	}

	cs, err := e.schemaForTenant(tc)
	if err != nil {
		return deny(), err
	}

	permDef, err := cs.GetPermission(req.ObjectType, req.Permission)
	if err != nil {
		return deny(), err
	}

	visited := newVisitedSet()
	result, err := e.walkPermission(ctx, req, cs, permDef, visited, 0)
	if err != nil {
		return deny(), err
	}

	// Evaluate ABAC condition if present and ReBAC passed
	if result.Allowed && permDef.Condition != nil {
		allowed, err := evaluateCondition(ctx, e.store, req, permDef.Condition)
		if err != nil {
			return deny(), fmt.Errorf("condition evaluation: %w", err)
		}
		if !allowed {
			return deny(), nil
		}
	}

	return result, nil
}
