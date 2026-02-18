---
id: types-and-relations
title: Types & Relations
sidebar_position: 2
---

# Types and Relations

## Types

A **type** represents a category of objects or subjects in your system. Every relation tuple references object types and subject types defined in the schema.

```yaml
types:
  user: {}        # no relations or permissions needed for leaf subjects
  document:
    relations: ...
    permissions: ...
```

Types with no relations or permissions (like `user`) are declared with an empty body `{}`.

### Attribute Declarations

Types can declare attributes that are used in ABAC conditions. Attribute declarations are **informational** — they document what attributes are expected but are not enforced at write time.

```yaml
types:
  user:
    attributes:
      clearance_level: int
      department: string
      region: string

  document:
    attributes:
      classification: string   # "public", "internal", "restricted"
      department: string
```

Supported attribute types (string labels only — no runtime enforcement):

| Label | Example |
|-------|---------|
| `string` | `"engineering"` |
| `int` | `42` |
| `float` | `3.14` |
| `bool` | `true` |
| `timestamp` | `"2024-01-01T00:00:00Z"` |

## Relations

A **relation** defines a named edge from an object type to one or more allowed subject types.

```yaml
types:
  document:
    relations:
      owner:
        types: [user]
      viewer:
        types: [user, group#member]
      parent:
        types: [folder]
```

### `types` list

Each entry in `types` is either:

| Format | Meaning |
|--------|---------|
| `user` | Any `user` object as a direct subject |
| `group#member` | Any member of a `group` (userset) |

The `group#member` form means "this relation can point to a userset: all subjects that hold the `member` relation on a `group` object".

### Examples

```yaml
relations:
  # Simple: only users
  owner:
    types: [user]

  # Mixed: users directly, or via group membership
  editor:
    types: [user, group#member]

  # Object reference: used for arrow traversal
  parent:
    types: [folder]
```

### Writing Tuples for Each Relation Type

**Direct user:**
```go
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType: "document", ObjectID: "spec",
    Relation:    "editor",
    SubjectType: "user", SubjectID: "alice",
})
// → document:spec#editor@user:alice
```

**Userset (`group#member`):**
```go
// First: make bob a member of engineering
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType: "group", ObjectID: "engineering",
    Relation:    "member",
    SubjectType: "user", SubjectID: "bob",
})

// Then: engineering members are editors of spec
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType:      "document", ObjectID: "spec",
    Relation:        "editor",
    SubjectType:     "group", SubjectID: "engineering",
    SubjectRelation: "member",
})
// → document:spec#editor@group:engineering#member
```

**Object reference (for arrows):**
```go
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType: "document", ObjectID: "design",
    Relation:    "parent",
    SubjectType: "folder", SubjectID: "root",
})
// → document:design#parent@folder:root
```

## Compiled Representation

After `schema.Compile`, each type becomes a `TypeDef`:

```go
type TypeDef struct {
    Name        string
    Attributes  map[string]string        // attr name → type label
    Relations   map[string]*RelationDef
    Permissions map[string]*PermissionDef
}

type RelationDef struct {
    Name         string
    AllowedTypes []*AllowedTypeRef
}

type AllowedTypeRef struct {
    Type     string  // e.g. "group"
    Relation string  // e.g. "member" (empty for direct refs)
}
```

## Querying Relations at Runtime

### Check a direct relation

```go
ok, err := store.CheckDirect(ctx,
    "document", "spec", "editor", "user", "alice")
// true if document:spec#editor@user:alice exists
```

### List all subjects of a relation

```go
subjects, err := store.ListSubjects(ctx, "document", "spec", "editor")
// returns []*model.SubjectRef
// e.g. [{Type:"user", ID:"alice"}, {Type:"group", ID:"engineering", Relation:"member"}]
```

### List all related objects

```go
objects, err := store.ListRelatedObjects(ctx, "document", "design", "parent")
// returns []*model.ObjectRef
// e.g. [{Type:"folder", ID:"root"}]
```

## See Also

- [Permissions](./permissions) — build computed rules on top of relations
- [Conditions](./conditions) — add ABAC gates using attribute expressions
