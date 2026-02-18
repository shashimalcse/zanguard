---
id: overview
title: Storage Overview
sidebar_position: 1
---

# Storage

ZanGuard uses a storage abstraction layer so the same engine and business logic work with any backend.

## The `TupleStore` Interface

All storage operations go through a single interface:

```go
type TupleStore interface {
    // Tenant management
    CreateTenant(ctx, tenant) error
    GetTenant(ctx, tenantID) (*Tenant, error)
    UpdateTenant(ctx, tenant) error
    ListTenants(ctx, filter) ([]*Tenant, error)

    // Tuple CRUD (tenant-scoped via ctx)
    WriteTuple(ctx, tuple) error
    WriteTuples(ctx, tuples) error
    DeleteTuple(ctx, tuple) error
    ReadTuples(ctx, filter) ([]*RelationTuple, error)

    // Zanzibar lookups
    CheckDirect(ctx, objectType, objectID, relation, subjectType, subjectID) (bool, error)
    ListRelatedObjects(ctx, objectType, objectID, relation) ([]*ObjectRef, error)
    ListSubjects(ctx, objectType, objectID, relation) ([]*SubjectRef, error)
    Expand(ctx, objectType, objectID, relation) (*SubjectTree, error)

    // Cross-tenant
    CheckDirectCrossTenant(ctx, targetTenantID, ...) (bool, error)

    // Attributes
    GetObjectAttributes(ctx, objectType, objectID) (map[string]any, error)
    SetObjectAttributes(ctx, objectType, objectID, attrs) error
    GetSubjectAttributes(ctx, subjectType, subjectID) (map[string]any, error)
    SetSubjectAttributes(ctx, subjectType, subjectID, attrs) error

    // Changelog
    AppendChangelog(ctx, entry) error
    ReadChangelog(ctx, sinceSeq, limit) ([]*ChangelogEntry, error)
    LatestSequence(ctx) (uint64, error)

    // Tenant data
    CountTuples(ctx) (int64, error)
    PurgeTenantData(ctx) error
    ExportTenantSnapshot(ctx, w io.Writer) error
}
```

## Available Backends

| Backend | Package | Use case |
|---------|---------|---------|
| In-memory | `pkg/storage/memory` | Tests, local development, edge deployments |
| PostgreSQL | `pkg/storage/postgres` | Production |

## Sentinel Errors

All backends return the same typed errors for consistent error handling:

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

## Tenant Scoping

All data operations (tuple CRUD, attributes, changelog) are **automatically scoped** to the tenant in `ctx`. There is no way to accidentally read or write across tenant boundaries for these operations.

Tenant management operations (CreateTenant, GetTenant, etc.) take an explicit `tenantID` parameter and do not require a tenant context.

## Zanzibar Lookups

Beyond basic CRUD, the store exposes three Zanzibar-specific read methods used by the engine:

### `CheckDirect`

Checks for an exact tuple match (no userset expansion):

```go
ok, err := store.CheckDirect(ctx,
    "document", "readme", "viewer", "user", "thilina")
// true only if: document:readme#viewer@user:thilina exists (with no SubjectRelation)
```

### `ListSubjects`

Returns all subjects holding a relation on an object, including userset refs:

```go
subjects, err := store.ListSubjects(ctx, "document", "spec", "viewer")
// [{Type:"user", ID:"alice"}, {Type:"group", ID:"eng", Relation:"member"}]
```

### `ListRelatedObjects`

Returns all objects linked via a relation (used for arrow traversal):

```go
objects, err := store.ListRelatedObjects(ctx, "document", "design", "parent")
// [{Type:"folder", ID:"root"}]
```

## Cross-Tenant Lookup

For cross-tenant subject references:

```go
ok, err := store.CheckDirectCrossTenant(ctx,
    "other-tenant", "document", "readme", "viewer", "user", "alice")
```

This bypasses the context tenant and queries the target tenant directly. Only readable tenants can be queried.

## Snapshot Export

Export all tuples for a tenant as newline-delimited JSON:

```go
var buf bytes.Buffer
err := store.ExportTenantSnapshot(ctx, &buf)
// each line is a JSON-encoded RelationTuple
```

## See Also

- [In-Memory Store](./in-memory) — usage and API details
- [PostgreSQL Store](./postgresql) — production setup
- [Changelog](./changelog) — audit log details
