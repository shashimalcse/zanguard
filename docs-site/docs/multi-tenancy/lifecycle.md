---
id: lifecycle
title: Tenant Lifecycle
sidebar_position: 2
---

# Tenant Lifecycle

Tenant status controls which operations are accepted.

## States

```text
pending -> active -> suspended -> active
             \
              -> deleted
```

## Effective Behavior by State

| State | Store Reads (tuples/attrs/changelog) | Store Writes | Runtime Check (`/access/v1/...`) |
|-------|--------------------------------------|--------------|-----------------------------------|
| `pending` | allowed | rejected | denied |
| `active` | allowed | allowed | allowed |
| `suspended` | allowed | rejected | allowed |
| `deleted` | rejected | rejected | denied |

## Transition Rules

### Create

`Create` initializes tenant status as `pending`.

### Activate

- Works from `pending`
- Works from `suspended`
- Works from `active` (idempotent update)
- Fails from `deleted`

### Suspend

- Works only from `active`
- Fails from `pending`, `suspended`, `deleted`

### Delete

- Works from `pending`, `active`, `suspended`
- Fails from `deleted`
- This is status-based soft delete (`status = deleted`)

## Important Notes

- Delete is not an automatic physical purge.
- `PurgeTenantData` currently requires writable tenant state (`active`).
- That means purge cannot be executed after tenant is marked `deleted`.

## Examples

```go
// create -> pending
_, err := mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)

// pending -> active
err = mgr.Activate(ctx, "acme")

// active -> suspended
err = mgr.Suspend(ctx, "acme")

// suspended -> active
err = mgr.Activate(ctx, "acme")

// active -> deleted
err = mgr.Delete(ctx, "acme")
```

## See Also

- [Multi-Tenancy Overview](./overview)
- [Schema Modes](./schema-modes)
- [Tenant Context](./context)
