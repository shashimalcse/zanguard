---
id: lifecycle
title: Tenant Lifecycle
sidebar_position: 2
---

# Tenant Lifecycle

Every tenant moves through a defined set of states. The state machine enforces which operations are allowed at each stage.

## States

```
pending → active → suspended → active
                ↘           ↗
                  deleted
```

| State | Reads | Writes | Description |
|-------|-------|--------|-------------|
| `pending` | ✅ | ❌ | Just created, not yet ready |
| `active` | ✅ | ✅ | Fully operational |
| `suspended` | ✅ | ❌ | Read-only; writes are rejected |
| `deleted` | ❌ | ❌ | Soft-deleted; all operations rejected |

## Transitions

### `Create` → pending

```go
t, err := mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)
// t.Status == model.TenantPending
```

The tenant exists in the store but cannot accept writes until activated.

### `Activate` → active

```go
err := mgr.Activate(ctx, "acme")
```

Valid from: `pending`, `suspended`. Invalid from: `deleted`.

### `Suspend` → suspended

```go
err := mgr.Suspend(ctx, "acme")
```

Valid only from: `active`. Suspending puts the tenant in read-only mode — useful during maintenance, billing issues, or compliance holds.

### Re-Activate from suspended

```go
err := mgr.Activate(ctx, "acme")
// Works from suspended state too
```

### `Delete` → deleted

```go
err := mgr.Delete(ctx, "acme")
```

Valid from any non-deleted state. This is a **soft delete** — the tenant record and all its data are retained in the store per the retention policy. No physical deletion occurs at this step.

:::warning
Attempting to activate a deleted tenant returns an error. Deleted tenants cannot be restored through the normal lifecycle.
:::

## Error Handling

```go
err := mgr.Suspend(ctx, "acme")
// Returns error if tenant is not active:
// "can only suspend active tenants, current status: pending"

err = mgr.Activate(ctx, "acme")
// Returns error if tenant is deleted:
// "cannot activate deleted tenant \"acme\""
```

## Listing Tenants by Status

```go
active, err := mgr.List(ctx, &model.TenantFilter{
    Status: model.TenantActive,
})

suspended, err := mgr.List(ctx, &model.TenantFilter{
    Status: model.TenantSuspended,
})
```

## Checking Tenant State Programmatically

```go
t, _ := mgr.Get(ctx, "acme")

fmt.Println(t.IsWritable())  // true only if active
fmt.Println(t.IsReadable())  // true if active or suspended
```

## Parent Tenants

Tenants can have a parent for organizational grouping:

```go
t := &model.Tenant{
    ID:             "acme-eu",
    DisplayName:    "Acme Corp — EU",
    ParentTenantID: "acme",
    Status:         model.TenantPending,
    SchemaMode:     model.SchemaInherited,
}
store.CreateTenant(ctx, t)
```

List child tenants:

```go
children, _ := mgr.List(ctx, &model.TenantFilter{
    ParentID: "acme",
})
```

## Data Purge

To explicitly wipe all data for a tenant (independent of soft-delete):

```go
tCtx, _ := tenant.BuildContext(ctx, store, "acme")
err := store.PurgeTenantData(tCtx)
```

This removes all tuples, attributes, and changelog entries for the tenant. The tenant record itself is preserved.

## See Also

- [Overview](./overview) — tenant model and isolation
- [Schema Modes](./schema-modes) — own vs shared vs inherited
- [Context](./context) — building tenant-scoped contexts
