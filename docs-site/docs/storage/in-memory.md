---
id: in-memory
title: In-Memory Store
sidebar_position: 2
---

# In-Memory Store

The in-memory store is a fully-featured, thread-safe implementation of `TupleStore` that keeps all data in process memory.

## When to Use It

- **Unit tests** — fast, no external dependencies, fully isolated
- **Integration tests** — set up and tear down in milliseconds
- **Local development** — run ZanGuard without a database
- **Edge deployments** — embedded use cases with small datasets

## Creating a Store

```go
import "zanguard/pkg/storage/memory"

store := memory.New()
```

No configuration required.

## Thread Safety

All operations are protected by a single `sync.RWMutex`. Read operations (`CheckDirect`, `ListSubjects`, `ReadTuples`, attribute reads, changelog reads) use a read lock. Write operations use a write lock.

The store is safe for use from concurrent goroutines:

```go
// Concurrent writes from 50 goroutines are safe
var wg sync.WaitGroup
for i := 0; i < 50; i++ {
    wg.Add(1)
    go func(i int) {
        defer wg.Done()
        store.WriteTuple(ctx, &model.RelationTuple{
            ObjectType: "document", ObjectID: fmt.Sprintf("doc-%d", i),
            Relation: "viewer", SubjectType: "user", SubjectID: "alice",
        })
    }(i)
}
wg.Wait()
```

## Data Layout

Data is stored in per-tenant buckets for O(1) purge:

```
tuples    : tenantID → []RelationTuple
objAttrs  : tenantID → "type:id" → map[string]any
subAttrs  : tenantID → "type:id" → map[string]any
changelog : tenantID → []ChangelogEntry
seqCounter: tenantID → uint64
```

## Tenant Lifecycle Guards

The store enforces tenant status on every operation:

| Status | Reads | Writes |
|--------|-------|--------|
| `active` | ✅ | ✅ |
| `suspended` | ✅ | ❌ (`ErrTenantSuspended`) |
| `pending` | ✅ | ❌ (`ErrTenantSuspended`) |
| `deleted` | ❌ (`ErrTenantDeleted`) | ❌ |

## Tuple Uniqueness

Writing a duplicate tuple returns `storage.ErrDuplicateTuple`:

```go
store.WriteTuple(ctx, tuple) // ok
store.WriteTuple(ctx, tuple) // returns ErrDuplicateTuple
```

Duplicate detection compares: `(ObjectType, ObjectID, Relation, SubjectType, SubjectID, SubjectRelation)`.

## Purge (O(1) Reset)

Wipe all data for the tenant context instantly:

```go
err := store.PurgeTenantData(ctx)
```

This nils the tuple slice and replaces the attribute maps with empty maps — it is O(1) regardless of how much data exists.

## Limitations

| Limitation | Detail |
|------------|--------|
| **No persistence** | Data is lost on process restart |
| **No pagination** | `ReadTuples` returns all matching tuples |
| **Linear scan** | Tuple lookups are O(n) — not suitable for large datasets |
| **No transactions** | Batch writes are not atomic across errors |

For production, use the [PostgreSQL store](./postgresql).

## Example: Full Test Setup

```go
func setupTestStore(t *testing.T) (*memory.Store, context.Context) {
    t.Helper()

    store := memory.New()
    ctx := context.Background()

    mgr := tenant.NewManager(store)
    mgr.Create(ctx, "test", "Test Tenant", model.SchemaOwn)
    mgr.Activate(ctx, "test")

    tCtx, _ := tenant.BuildContext(ctx, store, "test")
    return store, tCtx
}
```

## See Also

- [PostgreSQL Store](./postgresql) — production backend
- [Storage Overview](./overview) — full interface reference
