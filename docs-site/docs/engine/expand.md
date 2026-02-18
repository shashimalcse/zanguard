---
id: expand
title: Expand
sidebar_position: 2
---

# Expand

`Expand` returns the **subject tree** for a given object and relation — the complete set of subjects that hold that relation on the object.

## API

```go
tree, err := eng.Expand(ctx, objectType, objectID, relation)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `objectType` | `string` | Type of the object (e.g. `"document"`) |
| `objectID` | `string` | ID of the object (e.g. `"spec"`) |
| `relation` | `string` | Relation to expand (e.g. `"viewer"`) |

### Return value — `SubjectTree`

```go
type SubjectTree struct {
    Subject  *SubjectRef    // the root (the object+relation itself)
    Children []*SubjectTree // direct subjects
}

type SubjectRef struct {
    Type     string  // e.g. "user"
    ID       string  // e.g. "alice"
    Relation string  // e.g. "member" (set for userset refs)
}
```

## Example

Given these tuples:

```
document:spec#viewer@user:alice
document:spec#viewer@group:engineering#member
```

Calling `Expand(ctx, "document", "spec", "viewer")` returns:

```
SubjectTree{
  Subject: {Type: "document", ID: "spec", Relation: "viewer"},
  Children: [
    {Subject: {Type: "user", ID: "alice"}},
    {Subject: {Type: "group", ID: "engineering", Relation: "member"}},
  ]
}
```

The children are **direct** subjects only — Expand does not recursively resolve group members.

## Code Example

```go
tree, err := eng.Expand(ctx, "document", "spec", "viewer")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Root: %s:%s#%s\n", tree.Subject.Type, tree.Subject.ID, tree.Subject.Relation)

for _, child := range tree.Children {
    if child.Subject.Relation != "" {
        fmt.Printf("  userset: %s:%s#%s\n",
            child.Subject.Type, child.Subject.ID, child.Subject.Relation)
    } else {
        fmt.Printf("  direct:  %s:%s\n",
            child.Subject.Type, child.Subject.ID)
    }
}
```

Output:

```
Root: document:spec#viewer
  direct:  user:alice
  userset: group:engineering#member
```

## Use Cases

- **Audit**: "Who has access to this document?"
- **UI display**: Show the access list for a resource in a permissions panel
- **Debugging**: Inspect what tuples exist for a relation without running a full check
- **Sync**: Enumerate all subjects for replication or export

## Expand vs Check

| | `Check` | `Expand` |
|--|---------|---------|
| Question answered | "Can this specific user do this?" | "Who has this relation?" |
| Input | Object + permission + subject | Object + relation |
| Output | Allow/deny boolean + path | Subject tree |
| Recurses into groups | Yes | No (direct only) |
| Evaluates ABAC conditions | Yes | No |

## Store-Level Expand

`Engine.Expand` is a thin wrapper over `store.Expand`. You can also call the store directly:

```go
tree, err := store.Expand(ctx, "document", "spec", "viewer")
```

The store's `Expand` returns the same `SubjectTree` structure without any schema or engine involvement.

## See Also

- [Check](./check) — evaluate a specific subject's access
- [Storage: Overview](../storage/overview) — `ListSubjects` for raw tuple queries
