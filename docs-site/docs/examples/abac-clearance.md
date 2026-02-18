---
id: abac-clearance
title: ABAC Clearance Levels
sidebar_position: 3
---

# Example: ABAC Clearance Levels

This example shows how to combine relationship-based access (ReBAC) with attribute-based conditions (ABAC) to implement a clearance-level system — where both group membership and personal clearance are required to view sensitive documents.

## Scenario

- Documents are classified as `public`, `internal`, or `restricted`
- Users have a `clearance_level` (1–5)
- Viewing a document requires: being a viewer/editor/owner **AND** having sufficient clearance
- Deleting always requires clearance ≥ 3, even for owners

## Schema

```yaml
version: "1.0"

types:
  user:
    attributes:
      clearance_level: int
      department: string

  document:
    attributes:
      classification: string   # "public" | "internal" | "restricted"
      min_clearance: int        # minimum required clearance
      department: string
    relations:
      owner:
        types: [user]
      editor:
        types: [user]
      viewer:
        types: [user]
    permissions:
      # ReBAC: must be a viewer/editor/owner
      # ABAC: document must not be restricted, OR user must have clearance >= 4
      view:
        union:
          - viewer
          - editor
          - owner
        condition: >-
          object.classification != "restricted" ||
          subject.clearance_level >= 4

      # Editors and owners can edit — but only if clearance is sufficient
      edit:
        union:
          - editor
          - owner
        condition: "subject.clearance_level >= object.min_clearance"

      # Only owners can delete — and they need clearance >= 3
      delete:
        resolve: owner
        condition: "subject.clearance_level >= 3"
```

## Setup

```go
const clearanceSchema = `
version: "1.0"
types:
  user:
    attributes:
      clearance_level: int
  document:
    attributes:
      classification: string
      min_clearance: int
    relations:
      owner:
        types: [user]
      viewer:
        types: [user]
    permissions:
      view:
        union:
          - viewer
          - owner
        condition: "object.classification != \"restricted\" || subject.clearance_level >= 4"
      delete:
        resolve: owner
        condition: "subject.clearance_level >= 3"
`

store := memory.New()
mgr := tenant.NewManager(store)
mgr.Create(ctx, "demo", "Demo", model.SchemaOwn)
mgr.Activate(ctx, "demo")
tCtx, _ := tenant.BuildContext(ctx, store, "demo")

data := []byte(clearanceSchema)
raw, _ := schema.Parse(data)
cs, _ := schema.Compile(raw, data)
eng := engine.New(store, engine.DefaultConfig())
eng.LoadSchema("demo", cs)
```

## Writing Tuples and Attributes

```go
// Users
store.SetSubjectAttributes(tCtx, "user", "alice", map[string]any{
    "clearance_level": 2,  // low clearance
})
store.SetSubjectAttributes(tCtx, "user", "bob", map[string]any{
    "clearance_level": 4,  // high clearance
})
store.SetSubjectAttributes(tCtx, "user", "carol", map[string]any{
    "clearance_level": 5,  // owner-level clearance
})

// Documents
store.SetObjectAttributes(tCtx, "document", "public-doc", map[string]any{
    "classification": "public",
    "min_clearance":  1,
})
store.SetObjectAttributes(tCtx, "document", "internal-doc", map[string]any{
    "classification": "internal",
    "min_clearance":  2,
})
store.SetObjectAttributes(tCtx, "document", "secret-doc", map[string]any{
    "classification": "restricted",
    "min_clearance":  4,
})

// Relations: everyone is a viewer of all docs
for _, doc := range []string{"public-doc", "internal-doc", "secret-doc"} {
    for _, user := range []string{"alice", "bob"} {
        store.WriteTuple(tCtx, &model.RelationTuple{
            ObjectType: "document", ObjectID: doc,
            Relation: "viewer", SubjectType: "user", SubjectID: user,
        })
    }
}

// carol owns secret-doc
store.WriteTuple(tCtx, &model.RelationTuple{
    ObjectType: "document", ObjectID: "secret-doc",
    Relation: "owner", SubjectType: "user", SubjectID: "carol",
})
```

## Permission Checks

```go
type checkCase struct {
    label, objID, perm, userID string
    want                        bool
}

cases := []checkCase{
    // public-doc: anyone can view (not restricted)
    {"alice views public-doc",    "public-doc",   "view", "alice", true},
    {"bob views public-doc",      "public-doc",   "view", "bob",   true},

    // internal-doc: not restricted, so anyone can view
    {"alice views internal-doc",  "internal-doc", "view", "alice", true},

    // secret-doc: restricted — needs clearance >= 4
    {"alice views secret-doc",    "secret-doc",   "view", "alice", false},  // clearance=2, fail
    {"bob views secret-doc",      "secret-doc",   "view", "bob",   true},   // clearance=4, pass
    {"carol views secret-doc",    "secret-doc",   "view", "carol", true},   // clearance=5, pass

    // delete: must be owner AND clearance >= 3
    {"carol deletes secret-doc",  "secret-doc", "delete", "carol", true},   // owner + clearance=5
    {"bob deletes secret-doc",    "secret-doc", "delete", "bob",   false},  // not owner
}

for _, c := range cases {
    result, _ := eng.Check(tCtx, &engine.CheckRequest{
        ObjectType: "document", ObjectID: c.objID,
        Permission: c.perm,
        SubjectType: "user", SubjectID: c.userID,
    })
    status := "✓"
    if result.Allowed != c.want {
        status = "✗ UNEXPECTED"
    }
    fmt.Printf("%s %-40s → %v\n", status, c.label, result.Allowed)
}
```

## Expected Output

```
✓ alice views public-doc                   → true
✓ bob views public-doc                     → true
✓ alice views internal-doc                 → true
✓ alice views secret-doc                   → false
✓ bob views secret-doc                     → true
✓ carol views secret-doc                   → true
✓ carol deletes secret-doc                 → true
✓ bob deletes secret-doc                   → false
```

## Using Request Context for Dynamic Conditions

You can pass request-time attributes (IP address, time of day, MFA status) via `CheckRequest.Context`:

```go
result, _ := eng.Check(tCtx, &engine.CheckRequest{
    ObjectType: "document", ObjectID: "secret-doc",
    Permission: "view",
    SubjectType: "user", SubjectID: "bob",
    Context: map[string]any{
        "mfa_verified": true,
        "ip":           "10.0.1.5",
    },
})
```

Schema condition:

```yaml
condition: >-
  (object.classification != "restricted" || subject.clearance_level >= 4)
  && request.mfa_verified == true
```

## Key Takeaways

| | ReBAC alone | ABAC alone | ReBAC + ABAC (ZanGuard) |
|-|-------------|------------|------------------------|
| Group membership | ✅ | ❌ | ✅ |
| Attribute conditions | ❌ | ✅ | ✅ |
| Hierarchy traversal | ✅ | ❌ | ✅ |
| Audit trail | ✅ | ❌ | ✅ |
| Schema-driven | ✅ | Varies | ✅ |

## See Also

- [ABAC Conditions](../core-concepts/abac-conditions) — full expression reference
- [Schema: Conditions](../schema/conditions) — condition DSL syntax
- [Group Membership](./group-membership) — combine groups with clearance
