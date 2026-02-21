---
id: group-membership
title: Group Membership
sidebar_position: 2
---

# Example: Group Membership

This example shows how to use ZanGuard's userset expansion to implement group-based permissions — the most common pattern in real-world authorization systems.

## The Pattern

Instead of granting permissions to individual users, you:

1. Create a `group` type with a `member` relation
2. Add users to the group
3. Grant the group's membership a permission on a resource

The engine automatically expands the group when checking if a user has access.

## Schema

```yaml
version: "1.0"

types:
  user: {}

  group:
    relations:
      member:
        types: [user]
      admin:
        types: [user]
    permissions:
      manage:
        resolve: admin

  project:
    relations:
      owner:
        types: [user]
      contributor:
        types: [user, group#member]   # ← allows group userset
      viewer:
        types: [user, group#member]
    permissions:
      view:
        union:
          - viewer
          - contributor
          - owner
      contribute:
        union:
          - contributor
          - owner
      admin:
        resolve: owner
```

## Setup

Assume imports include `log`, `os`, and `zanguard/pkg/storage/postgres`.

```go
const groupSchema = `
version: "1.0"
types:
  user: {}
  group:
    relations:
      member:
        types: [user]
    permissions:
      has_member:
        resolve: member
  project:
    relations:
      owner:
        types: [user]
      contributor:
        types: [user, group#member]
      viewer:
        types: [user, group#member]
    permissions:
      view:
        union:
          - viewer
          - contributor
          - owner
      contribute:
        union:
          - contributor
          - owner
`

store, err := postgres.New(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}
defer store.Close()
mgr := tenant.NewManager(store)
mgr.Create(ctx, "demo", "Demo", model.SchemaOwn)
mgr.Activate(ctx, "demo")
tCtx, _ := tenant.BuildContext(ctx, store, "demo")

data := []byte(groupSchema)
raw, _ := schema.Parse(data)
cs, _ := schema.Compile(raw, data)
eng := engine.New(store, engine.DefaultConfig())
eng.LoadSchema("demo", cs)
```

## Writing the Tuples

```go
// 1. Build group membership
//    group:backend-team members: alice, bob
store.WriteTuple(tCtx, &model.RelationTuple{
    ObjectType: "group", ObjectID: "backend-team",
    Relation: "member", SubjectType: "user", SubjectID: "alice",
})
store.WriteTuple(tCtx, &model.RelationTuple{
    ObjectType: "group", ObjectID: "backend-team",
    Relation: "member", SubjectType: "user", SubjectID: "bob",
})

// 2. Grant the group access to a project
//    project:api#contributor → group:backend-team#member
store.WriteTuple(tCtx, &model.RelationTuple{
    ObjectType:      "project",
    ObjectID:        "api",
    Relation:        "contributor",
    SubjectType:     "group",
    SubjectID:       "backend-team",
    SubjectRelation: "member",          // ← userset reference
})

// 3. Carol is a direct viewer (not via group)
store.WriteTuple(tCtx, &model.RelationTuple{
    ObjectType: "project", ObjectID: "api",
    Relation: "viewer", SubjectType: "user", SubjectID: "carol",
})
```

## Checking Permissions

```go
// alice can contribute (via group membership)
r1, _ := eng.Check(tCtx, &engine.CheckRequest{
    ObjectType: "project", ObjectID: "api",
    Permission: "contribute", SubjectType: "user", SubjectID: "alice",
})
fmt.Println(r1.Allowed) // true

// bob can also contribute (also in group)
r2, _ := eng.Check(tCtx, &engine.CheckRequest{
    ObjectType: "project", ObjectID: "api",
    Permission: "contribute", SubjectType: "user", SubjectID: "bob",
})
fmt.Println(r2.Allowed) // true

// carol can view (direct relation)
r3, _ := eng.Check(tCtx, &engine.CheckRequest{
    ObjectType: "project", ObjectID: "api",
    Permission: "view", SubjectType: "user", SubjectID: "carol",
})
fmt.Println(r3.Allowed) // true

// carol CANNOT contribute (not in group, not owner)
r4, _ := eng.Check(tCtx, &engine.CheckRequest{
    ObjectType: "project", ObjectID: "api",
    Permission: "contribute", SubjectType: "user", SubjectID: "carol",
})
fmt.Println(r4.Allowed) // false

// dave has no access at all
r5, _ := eng.Check(tCtx, &engine.CheckRequest{
    ObjectType: "project", ObjectID: "api",
    Permission: "view", SubjectType: "user", SubjectID: "dave",
})
fmt.Println(r5.Allowed) // false
```

## Expanding Group Members

Use `Expand` to see who holds a relation:

```go
tree, _ := eng.Expand(tCtx, "project", "api", "contributor")

fmt.Printf("contributor of project:api:\n")
for _, child := range tree.Children {
    if child.Subject.Relation != "" {
        fmt.Printf("  userset: %s:%s#%s\n",
            child.Subject.Type, child.Subject.ID, child.Subject.Relation)
    } else {
        fmt.Printf("  direct:  %s:%s\n",
            child.Subject.Type, child.Subject.ID)
    }
}
// Output:
// contributor of project:api:
//   userset: group:backend-team#member
```

## Nested Groups

ZanGuard supports nested group membership — a group can be a member of another group:

```yaml
group:
  relations:
    member:
      types: [user, group#member]   # ← add group#member here
```

```go
// group:leads is a subgroup of group:engineering
store.WriteTuple(tCtx, &model.RelationTuple{
    ObjectType:      "group", ObjectID: "engineering",
    Relation:        "member",
    SubjectType:     "group", SubjectID: "leads",
    SubjectRelation: "member",
})
```

The engine will recursively expand: `user → group:leads#member → group:engineering#member → resource`.

## See Also

- [Relation Tuples](../core-concepts/relation-tuples) — userset tuple format
- [Engine: Check](../engine/check) — how userset expansion works
- [ABAC Clearance](./abac-clearance) — add attribute conditions on top of groups
