---
id: changelog
title: Audit Changelog
sidebar_position: 4
---

# Audit Changelog

ZanGuard includes a built-in append-only changelog for tracking all mutations to tenant data. Every entry is assigned a monotonically increasing sequence number, making it suitable for audit trails, event streaming, and change detection.

## Data Model

```go
type ChangelogEntry struct {
    TenantID  string
    Sequence  uint64        // monotonically increasing per tenant
    Timestamp time.Time
    Action    string        // e.g. "write_tuple", "delete_tuple"
    TupleKey  string        // canonical tuple key affected
    Actor     string        // optional: who performed the action
    Metadata  map[string]any
}
```

## Writing to the Changelog

```go
err := store.AppendChangelog(ctx, &model.ChangelogEntry{
    Action:   "write_tuple",
    TupleKey: tuple.TupleKey(),
    Actor:    "api:service-account",
    Metadata: map[string]any{
        "reason": "user granted access by admin",
    },
})
```

The store automatically assigns `TenantID`, `Sequence`, and `Timestamp` — you do not need to set them.

## Reading the Changelog

Read entries since a given sequence number (use `0` for all entries):

```go
entries, err := store.ReadChangelog(ctx, sinceSeq, limit)
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `sinceSeq` | `uint64` | Return entries with sequence > this value |
| `limit` | `int` | Max entries to return (0 = unlimited) |

### Example: Read all entries

```go
entries, _ := store.ReadChangelog(ctx, 0, 0)
for _, e := range entries {
    fmt.Printf("[%d] %s %s by %s at %s\n",
        e.Sequence, e.Action, e.TupleKey, e.Actor, e.Timestamp)
}
```

### Example: Poll for new entries

```go
var lastSeq uint64

for {
    entries, _ := store.ReadChangelog(ctx, lastSeq, 100)
    for _, e := range entries {
        processEntry(e)
        lastSeq = e.Sequence
    }
    time.Sleep(1 * time.Second)
}
```

## Latest Sequence

Get the current highest sequence number for the tenant:

```go
seq, err := store.LatestSequence(ctx)
fmt.Printf("Latest sequence: %d\n", seq)
```

Use this to establish a starting point for polling or to verify the changelog is advancing.

## Properties

| Property | Detail |
|----------|--------|
| **Append-only** | Entries are never modified or deleted |
| **Per-tenant** | Sequences are independent per tenant (start at 1) |
| **Monotonic** | Sequence numbers always increase, never repeat |
| **Tenant-scoped** | Only entries for the context tenant are visible |

## Use Cases

| Use Case | How |
|----------|-----|
| Compliance audit | Read full changelog and export to SIEM |
| Change detection | Poll `ReadChangelog(lastSeq, 100)` |
| Event streaming | Tail changelog, emit to Kafka/webhook |
| Debugging | Trace exactly when a tuple was added/removed |

## See Also

- [Storage Overview](./overview) — full interface reference
- [Multi-Tenancy](../multi-tenancy/overview) — per-tenant data isolation
