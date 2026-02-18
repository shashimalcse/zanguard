---
id: schema-modes
title: Schema Modes
sidebar_position: 3
---

# Schema Modes

A tenant's **schema mode** determines where its authorization schema comes from. ZanGuard supports three modes to balance flexibility with operational simplicity.

## Modes

| Mode | Constant | Description |
|------|----------|-------------|
| `own` | `model.SchemaOwn` | Tenant has its own independent schema |
| `shared` | `model.SchemaShared` | Tenant uses a pre-registered shared schema |
| `inherited` | `model.SchemaInherited` | Tenant extends a parent schema (Phase 1: same as `own`) |

## `own` — Independent Schema

Each tenant has a completely independent schema loaded via `eng.LoadSchema(tenantID, cs)`.

```go
// Create tenant with its own schema
mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)
mgr.Activate(ctx, "acme")

// Load an acme-specific schema
data, _ := os.ReadFile("schemas/acme.yaml")
raw, _ := schema.Parse(data)
cs, _ := schema.Compile(raw, data)
eng.LoadSchema("acme", cs)
```

**Use when:** Tenants have meaningfully different authorization models (different object types, relations, or permission logic).

## `shared` — Shared Schema

Multiple tenants share a single compiled schema. The schema is registered once with a reference name.

```go
// Create tenants referencing the same schema
mgr.Create(ctx, "startup-a", "Startup A", model.SchemaShared)
t := &model.Tenant{
    ID:              "startup-a",
    SchemaMode:      model.SchemaShared,
    SharedSchemaRef: "saas-standard-v1",  // ← reference name
}
store.CreateTenant(ctx, t)

// Register the shared schema once
data, _ := os.ReadFile("schemas/saas-standard.yaml")
raw, _ := schema.Parse(data)
cs, _ := schema.Compile(raw, data)
eng.LoadSharedSchema("saas-standard-v1", cs)  // ← registered by ref name
```

When the engine resolves a tenant with `SchemaShared`, it looks up `SharedSchemaRef` in `sharedSchemas` instead of `schemas`.

**Use when:** You have many tenants with identical authorization models (e.g. a SaaS product where all customers use the same object types and permissions).

**Benefits:**
- Schema is compiled once, not N times
- Schema updates affect all tenants simultaneously
- Lower memory usage at scale

## `inherited` — Extended Schema

The `inherited` mode is reserved for tenants that extend a parent tenant's schema with additional types or permissions. In Phase 1, inherited behaves identically to `own` — the tenant's schema is loaded via `eng.LoadSchema(tenantID, cs)`.

```go
mgr.Create(ctx, "acme-eu", "Acme EU", model.SchemaInherited)
eng.LoadSchema("acme-eu", extendedCompiledSchema)
```

Full inheritance merging (parent schema + overrides) is planned for a future phase.

**Use when:** You have a hierarchical tenant structure where child tenants share most of the parent's schema but add custom types or rules.

## Setting the Schema Mode

```go
// Via Manager.Create
mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)

// Via direct Tenant struct
t := &model.Tenant{
    ID:              "startup-x",
    DisplayName:     "Startup X",
    Status:          model.TenantPending,
    SchemaMode:      model.SchemaShared,
    SharedSchemaRef: "standard-v2",
}
store.CreateTenant(ctx, t)
```

## Engine Schema Resolution

The engine calls `schemaForTenant` before every check to select the right schema:

```go
// own / inherited → look up schemas[tenantID]
// shared          → look up sharedSchemas[SharedSchemaRef]
cs, err := e.schemaForTenant(tc)
```

If no matching schema is found, the check returns an error immediately.

## Loading Schemas

```go
// For own/inherited tenants
eng.LoadSchema("acme", compiledSchema)

// For shared schemas
eng.LoadSharedSchema("standard-v2", compiledSchema)
```

Both methods are protected by a `sync.RWMutex` and are safe to call from concurrent goroutines.

## See Also

- [Tenant Lifecycle](./lifecycle) — state machine
- [Schema Overview](../schema/overview) — writing and compiling schemas
- [Engine: Check](../engine/check) — how schemas are used during evaluation
