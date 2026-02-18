---
id: overview
title: Multi-Tenancy Overview
sidebar_position: 1
---

# Multi-Tenancy

ZanGuard is built for multi-tenant SaaS from the ground up. Every tuple, attribute, and changelog entry is scoped to a tenant. There is no shared data between tenants.

## Tenant Isolation

| Layer | Isolation mechanism |
|-------|-------------------|
| **Storage** | Every table has a `tenant_id` column; queries always filter by it |
| **Engine** | `Check` reads the tenant from `ctx` before any lookup |
| **Schema** | Each tenant has its own compiled schema (or references a shared one) |
| **Changelog** | Sequences are per-tenant and independent |

It is architecturally impossible for one tenant to read another tenant's tuples through the normal API — the tenant ID comes from the request context, not from user input.

## Tenant Model

```go
type Tenant struct {
    ID              string       // e.g. "acme" — unique, immutable
    DisplayName     string       // e.g. "Acme Corp"
    ParentTenantID  string       // optional: for hierarchical tenants
    Status          TenantStatus // pending | active | suspended | deleted
    SchemaMode      SchemaMode   // own | shared | inherited
    SharedSchemaRef string       // used when SchemaMode == "shared"
    Config          TenantConfig // quotas, retention, webhooks
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

## Tenant ID Rules

- Lowercase alphanumeric characters and hyphens only
- Minimum 2 characters, maximum 128 characters
- Must start and end with an alphanumeric character
- Examples: `acme`, `my-org`, `tenant-42`, `a0`

## Per-Tenant Configuration

```go
type TenantConfig struct {
    MaxTuples          int64          // tuple quota (0 = unlimited)
    MaxRequestsPerSec  int            // rate limit (0 = unlimited)
    CacheTTLOverride   *time.Duration // per-tenant cache TTL (phase 2)
    AllowedObjectTypes []string       // restrict which types can be used
    RetentionDays      int            // changelog retention
    SyncEnabled        bool           // enable real-time sync (phase 5)
    WebhookURL         string         // change notification endpoint
    Metadata           map[string]any // arbitrary key-value pairs
}
```

## Creating and Managing Tenants

```go
mgr := tenant.NewManager(store)

// Create (starts in "pending" state)
t, err := mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)

// Activate (required before any reads or writes)
err = mgr.Activate(ctx, "acme")

// Suspend (read-only mode, writes rejected)
err = mgr.Suspend(ctx, "acme")

// Delete (soft-delete, data retained per retention policy)
err = mgr.Delete(ctx, "acme")

// Get a tenant
t, err = mgr.Get(ctx, "acme")

// List tenants with optional filter
tenants, err := mgr.List(ctx, &model.TenantFilter{
    Status:   model.TenantActive,
    ParentID: "parent-org",
    Limit:    100,
    Offset:   0,
})
```

## Tenant Context

Every request must carry a `TenantContext` in `ctx`. This context is the source of truth for all tenant-scoped operations.

```go
// Build a context for a specific tenant
tenantCtx, err := tenant.BuildContext(ctx, store, "acme")

// Extract the context in downstream code
tc := model.TenantFromContext(ctx)
if tc == nil {
    return model.ErrNoTenantContext
}
fmt.Println(tc.TenantID) // "acme"
fmt.Println(tc.Tenant.Status) // "active"
```

See [Tenant Context](./context) for full details.

## Tenant Isolation in Practice

```go
// Write a tuple under acme
store.WriteTuple(acmeCtx, &model.RelationTuple{
    ObjectType: "document", ObjectID: "secret",
    Relation: "viewer", SubjectType: "user", SubjectID: "alice",
})

// Check under globex — DENIED (no tuples for globex)
result, _ := eng.Check(globexCtx, &engine.CheckRequest{
    ObjectType: "document", ObjectID: "secret",
    Permission: "view", SubjectType: "user", SubjectID: "alice",
})
fmt.Println(result.Allowed) // false — total isolation
```

## Reference Pages

- [Tenant Lifecycle](./lifecycle) — state machine details
- [Schema Modes](./schema-modes) — own, shared, inherited
- [Tenant Context](./context) — context propagation
