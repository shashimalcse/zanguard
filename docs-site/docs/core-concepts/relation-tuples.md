---
id: relation-tuples
title: Relation Tuples
sidebar_position: 1
---

# Relation Tuples

A **relation tuple** is the atomic unit of authorization in ZanGuard. Every authorization fact — "Alice is an editor of document:spec", "Group:engineering has member user:bob" — is stored as a relation tuple.

## Format

```
<object_type>:<object_id>#<relation>@<subject_type>:<subject_id>
```

With an optional subject relation (for usersets):

```
<object_type>:<object_id>#<relation>@<subject_type>:<subject_id>#<subject_relation>
```

## Examples

| Tuple | Meaning |
|-------|---------|
| `document:readme#viewer@user:thilina` | thilina is a viewer of readme |
| `document:spec#editor@group:engineering#member` | All members of group:engineering are editors of spec |
| `folder:root#owner@user:carol` | carol owns the root folder |
| `document:design#parent@folder:root` | design's parent is folder:root |

## The `RelationTuple` Struct

```go
type RelationTuple struct {
    TenantID        string         // tenant this tuple belongs to
    ObjectType      string         // e.g. "document"
    ObjectID        string         // e.g. "readme"
    Relation        string         // e.g. "viewer"
    SubjectType     string         // e.g. "user"
    SubjectID       string         // e.g. "thilina"
    SubjectRelation string         // optional: e.g. "member" (for usersets)
    Attributes      map[string]any // optional: tuple-level metadata
    CreatedAt       time.Time
    UpdatedAt       time.Time
    SourceSystem    string         // optional: external provenance
    ExternalID      string         // optional: external system ID
}
```

## Direct Tuples

A direct tuple links a subject directly to an object:

```go
// user:alice is a viewer of document:report
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType:  "document",
    ObjectID:    "report",
    Relation:    "viewer",
    SubjectType: "user",
    SubjectID:   "alice",
})
```

## Userset Tuples

A userset tuple links a *group of subjects* (identified by a relation on another object) to an object. This enables group membership patterns.

```go
// All members of group:engineering are viewers of document:spec
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType:      "document",
    ObjectID:        "spec",
    Relation:        "viewer",
    SubjectType:     "group",
    SubjectID:       "engineering",
    SubjectRelation: "member",   // ← makes this a userset tuple
})
```

When the engine checks if `user:bob` can view `document:spec`, it:
1. Finds the userset tuple `document:spec#viewer@group:engineering#member`
2. Recursively checks if `user:bob` is a `member` of `group:engineering`
3. If yes → allow

## Tuple Key

Every tuple has a canonical string representation:

```go
// Direct:  "document:readme#viewer@user:thilina"
// Userset: "document:spec#viewer@group:engineering#member"
key := tuple.TupleKey()
```

Keys are used internally for cycle detection and deduplication.

## Writing Tuples

```go
// Write a single tuple
err := store.WriteTuple(ctx, tuple)

// Write multiple tuples atomically
err := store.WriteTuples(ctx, []*model.RelationTuple{t1, t2, t3})
```

Writing a duplicate returns `storage.ErrDuplicateTuple`.

## Deleting Tuples

```go
err := store.DeleteTuple(ctx, &model.RelationTuple{
    ObjectType:  "document",
    ObjectID:    "readme",
    Relation:    "viewer",
    SubjectType: "user",
    SubjectID:   "thilina",
})
```

Returns `storage.ErrNotFound` if the tuple does not exist.

## Reading Tuples

Use a `TupleFilter` to query tuples. All fields are optional; omit to match all.

```go
tuples, err := store.ReadTuples(ctx, &model.TupleFilter{
    ObjectType: "document",
    ObjectID:   "readme",
    Relation:   "viewer",
})
```

## Filtering Reference

```go
type TupleFilter struct {
    ObjectType      string  // filter by object type
    ObjectID        string  // filter by object ID
    Relation        string  // filter by relation name
    SubjectType     string  // filter by subject type
    SubjectID       string  // filter by subject ID
    SubjectRelation string  // filter by subject relation (userset)
}
```

## Tenant Isolation

Every tuple is automatically scoped to the tenant in the request context. A tuple written under `tenant:acme` is **never visible** to queries under `tenant:globex`.

```go
// Written under acme
store.WriteTuple(acmeCtx, tuple)

// Returns nothing — globex has no tuples
store.ReadTuples(globexCtx, &model.TupleFilter{})
```

## See Also

- [Permissions](./permissions) — how tuples are used to evaluate permissions
- [Engine: Check](../engine/check) — the full traversal algorithm
