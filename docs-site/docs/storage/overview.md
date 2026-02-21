---
id: overview
title: Storage Overview
sidebar_position: 1
---

# Storage

ZanGuard uses a `TupleStore` interface internally. The server runtime uses PostgreSQL.

## Runtime Backend

- Runtime backend: `pkg/storage/postgres`
- Server startup requires `DATABASE_URL`
- The in-memory store package exists for internal tests/dev code, not for the default server runtime path

## The `TupleStore` Interface

Core operations are grouped into:

- Tenant management (`CreateTenant`, `GetTenant`, `UpdateTenant`, `ListTenants`)
- Tuple CRUD (`WriteTuple`, `WriteTuples`, `DeleteTuple`, `ReadTuples`)
- Engine lookups (`CheckDirect`, `ListSubjects`, `ListRelatedObjects`, `Expand`)
- Attributes (`Get/SetObjectAttributes`, `Get/SetSubjectAttributes`)
- Changelog (`AppendChangelog`, `ReadChangelog`, `LatestSequence`)
- Tenant data operations (`CountTuples`, `PurgeTenantData`, `ExportTenantSnapshot`)

## Tenant Scoping Rules

Tenant-scoped operations derive tenant identity from `context.Context` (`model.TenantFromContext`).

- Missing tenant context returns `model.ErrNoTenantContext`
- Tenant management methods take explicit tenant IDs and do not require tenant context

## Read/Write State Enforcement

Store-level enforcement in current implementation:

- Writes require tenant status `active`
- Reads are allowed for non-deleted tenants (including `pending` and `suspended`)
- Deleted tenants return `ErrTenantDeleted` or `ErrTenantNotFound` depending on access path

Note: runtime permission checks add stricter rules in the engine (`pending` checks are denied).

## Changelog Behavior

- Changelog entries are tenant-filtered on read
- Sequence values come from one database sequence (`BIGSERIAL`)
- Within a tenant stream, sequence values are increasing but may have gaps

## Sentinel Errors

The interface exposes shared sentinel errors:

```go
var (
    ErrTenantNotFound  = errors.New("tenant not found")
    ErrTenantDeleted   = errors.New("tenant has been deleted")
    ErrTenantSuspended = errors.New("tenant is suspended")
    ErrQuotaExceeded   = errors.New("tenant quota exceeded")
    ErrDuplicateTuple  = errors.New("tuple already exists")
    ErrNotFound        = errors.New("not found")
)
```

## Current Notes

- `CheckDirectCrossTenant` exists in the storage interface for internal/service use.
- The exposed HTTP APIs do not provide cross-tenant tuple read endpoints.
- Tenant config fields are persisted and carried in tenant context; not all fields are actively enforced by runtime paths yet.

## See Also

- [PostgreSQL Store](./postgresql)
- [Changelog](./changelog)
