---
id: check
title: Permission Check
sidebar_position: 1
---

# Permission Check

The `Check` method is the core operation of ZanGuard. Given a subject, an object, and a permission name, it traverses the relation graph to determine if access is allowed.

## API

```go
result, err := eng.Check(ctx, &engine.CheckRequest{
    ObjectType:  "document",
    ObjectID:    "readme",
    Permission:  "view",
    SubjectType: "user",
    SubjectID:   "thilina",
    Context:     map[string]any{"ip": "10.0.0.1"}, // optional
})
```

### `CheckRequest`

```go
type CheckRequest struct {
    ObjectType  string         // e.g. "document"
    ObjectID    string         // e.g. "readme"
    Permission  string         // e.g. "view"
    SubjectType string         // e.g. "user"
    SubjectID   string         // e.g. "thilina"
    Context     map[string]any // request-time ABAC context (optional)
}
```

### `CheckResult`

```go
type CheckResult struct {
    Allowed        bool
    ResolutionPath []string // which tuple granted access
    Error          error
}
```

## How the Engine Evaluates a Check

### 1. Tenant validation

The engine reads the `TenantContext` from `ctx`. If none is present, or if the tenant is deleted or unreadable, the check immediately denies.

```go
tc := model.TenantFromContext(ctx)
if tc == nil {
    return deny(), model.ErrNoTenantContext
}
```

### 2. Schema resolution

The correct schema is loaded for the tenant based on its schema mode (own / shared / inherited).

### 3. Permission definition lookup

The engine fetches the `PermissionDef` for `(ObjectType, Permission)` from the compiled schema. If the type or permission doesn't exist, the check denies with a validation error.

### 4. Graph traversal (`walkPermission`)

The engine walks the permission's logical tree:

| Operation | Behavior |
|-----------|----------|
| `union` / `resolve` | Returns `allow` on the first child that allows (short-circuit OR) |
| `intersect` | Returns `deny` on the first child that denies (short-circuit AND) |
| `exclusion` | Allows if the base allows AND no exclusion allows |

### 5. Child evaluation (`walkChild`)

Each child of a permission can be one of three kinds:

| Kind | Action |
|------|--------|
| **Relation ref** | Calls `walkRelation` |
| **Arrow ref** | Calls `walkArrow` |
| **Condition** | Compiles and evaluates the ABAC expression inline |

### 6. Relation walk (`walkRelation`)

1. **Direct lookup** — checks if the exact tuple `(objectType, objectID, relation, subjectType, subjectID)` exists
2. **Userset expansion** — lists all subjects with a `SubjectRelation` set and recursively checks each one

```
document:spec#viewer@group:engineering#member
→ is user:bob a member of group:engineering?
  → document:spec#viewer@user:bob? No
  → group:engineering#member@user:bob? Yes → ALLOW
```

### 7. Arrow walk (`walkArrow`)

1. Lists all related objects via the arrow's relation (e.g. `parent`)
2. For each linked object, runs a recursive `Check` for the arrow's permission

```
document:design#parent@folder:root
→ Can user:carol view folder:root?
  → folder:root#owner@user:carol? Yes → ALLOW
```

### 8. ABAC condition (if present)

If the structural check (step 4–7) returns `allow` AND the permission has a top-level `condition`, the condition is evaluated. If it fails, the result becomes `deny`.

## Cycle Detection

The engine maintains a `visitedSet` of `(objectType, objectID, permission, subjectType, subjectID)` keys. Before processing any node, it checks if the node has been visited. If so, it returns `deny` immediately (cycles do not grant access).

```go
key := visitKey(req.ObjectType, req.ObjectID, permDef.Name, req.SubjectType, req.SubjectID)
if visited.has(key) {
    return deny(), nil  // cycle detected — deny
}
visited.add(key)
```

## Depth Limiting

The engine tracks recursion depth and returns an error if `MaxCheckDepth` is exceeded:

```go
if depth > e.cfg.MaxCheckDepth {
    return deny(), fmt.Errorf("max check depth (%d) exceeded", e.cfg.MaxCheckDepth)
}
```

Default: **25 hops**. Configure via `engine.Config`:

```go
eng := engine.New(store, engine.Config{
    MaxCheckDepth: 50,
})
```

## Engine Configuration

```go
type Config struct {
    MaxCheckDepth int  // default: 25
}

// Use defaults
eng := engine.New(store, engine.DefaultConfig())

// Custom config
eng := engine.New(store, engine.Config{MaxCheckDepth: 50})
```

## Resolution Path

When access is granted, `ResolutionPath` contains the tuple that granted it:

```go
result, _ := eng.Check(ctx, req)
fmt.Println(result.ResolutionPath)
// ["document:readme#viewer@user:thilina"]
```

For nested grants (userset or arrow), the innermost granting tuple is returned.

## Loading Schemas

Register a schema per tenant before checking:

```go
eng.LoadSchema("acme", compiledSchema)

// For shared schemas (used by multiple tenants)
eng.LoadSharedSchema("standard-v1", compiledSchema)
```

Schema loading is protected by a read-write mutex — safe to call concurrently.

## Error Types

| Error | Meaning |
|-------|---------|
| `model.ErrNoTenantContext` | No tenant in context |
| `storage.ErrTenantDeleted` | Tenant is deleted |
| `schema.ValidationError` | Unknown type or permission |
| `"max check depth exceeded"` | Possible cycle or very deep graph |
| `"condition evaluation: ..."` | ABAC expression runtime error |

## Full Example

```go
package main

import (
    "context"
    "fmt"

    "zanguard/pkg/engine"
    "zanguard/pkg/model"
    "zanguard/pkg/schema"
    "zanguard/pkg/storage/memory"
    "zanguard/pkg/tenant"
)

const mySchema = `
version: "1.0"
types:
  user: {}
  group:
    relations:
      member:
        types: [user]
  document:
    relations:
      viewer:
        types: [user, group#member]
      owner:
        types: [user]
    permissions:
      view:
        union:
          - viewer
          - owner
`

func main() {
    ctx := context.Background()
    store := memory.New()

    mgr := tenant.NewManager(store)
    mgr.Create(ctx, "acme", "Acme", model.SchemaOwn)
    mgr.Activate(ctx, "acme")
    tCtx, _ := tenant.BuildContext(ctx, store, "acme")

    data := []byte(mySchema)
    raw, _ := schema.Parse(data)
    cs, _ := schema.Compile(raw, data)

    eng := engine.New(store, engine.DefaultConfig())
    eng.LoadSchema("acme", cs)

    // group:eng#member → user:bob
    store.WriteTuple(tCtx, &model.RelationTuple{
        ObjectType: "group", ObjectID: "eng",
        Relation: "member", SubjectType: "user", SubjectID: "bob",
    })

    // document:spec#viewer → group:eng#member
    store.WriteTuple(tCtx, &model.RelationTuple{
        ObjectType:      "document", ObjectID: "spec",
        Relation:        "viewer",
        SubjectType:     "group", SubjectID: "eng",
        SubjectRelation: "member",
    })

    result, _ := eng.Check(tCtx, &engine.CheckRequest{
        ObjectType: "document", ObjectID: "spec",
        Permission: "view", SubjectType: "user", SubjectID: "bob",
    })

    fmt.Println(result.Allowed) // true — via group membership
}
```

## See Also

- [Expand](./expand) — enumerate all subjects with access
- [Cycle Detection](./cycle-detection) — how loops are handled
- [Core Concepts: Permissions](../core-concepts/permissions) — conceptual overview
