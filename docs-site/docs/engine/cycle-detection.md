---
id: cycle-detection
title: Cycle Detection & Depth Limits
sidebar_position: 3
---

# Cycle Detection & Depth Limits

ZanGuard is safe in the presence of circular relations. The engine will never hang or panic due to cycles in the relation graph.

## Why Cycles Happen

Cycles can arise naturally in hierarchical data:

```
folder:A#parent → folder:B
folder:B#parent → folder:A   ← cycle!
```

Or via misconfigured data:

```
group:admins#member → user:superuser
user:superuser#group  → group:admins  ← circular reference
```

## Cycle Detection via Visited Set

The engine maintains a `visitedSet` for each top-level `Check` call. Before processing any `(objectType, objectID, permission, subjectType, subjectID)` node, the engine checks if it has already been visited in the current traversal.

```
visitKey = "objectType:objectID#permission@subjectType:subjectID"
```

If the key is already in the set, the engine returns `deny` immediately — **cycles do not grant access**.

```go
key := visitKey(req.ObjectType, req.ObjectID, permDef.Name, req.SubjectType, req.SubjectID)
if visited.has(key) {
    return deny(), nil   // silently deny — not an error
}
visited.add(key)
```

The visited set is **per-request** — it is not shared between concurrent checks and does not persist across calls.

## Depth Limiting

In addition to cycle detection, the engine enforces a maximum traversal depth:

```go
if depth > e.cfg.MaxCheckDepth {
    return deny(), fmt.Errorf("max check depth (%d) exceeded — possible cycle", e.cfg.MaxCheckDepth)
}
```

The default limit is **25 hops**. If this limit is hit, the check returns `deny` with an error. This protects against degenerate graphs where the cycle detection key alone might not catch a problem early enough.

## Configuring the Depth Limit

```go
eng := engine.New(store, engine.Config{
    MaxCheckDepth: 50, // increase for deeper hierarchies
})
```

For typical authorization graphs (3–5 hops), the default of 25 is more than sufficient.

## Behavior Summary

| Scenario | Result |
|----------|--------|
| Cycle detected by visited set | `deny`, no error |
| Depth limit exceeded | `deny`, error returned |
| Deep but acyclic graph within limit | Normal evaluation |

## Example: Cycle-Safe Check

```go
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType: "folder", ObjectID: "a",
    Relation: "parent", SubjectType: "folder", SubjectID: "b",
})
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType: "folder", ObjectID: "b",
    Relation: "parent", SubjectType: "folder", SubjectID: "a",
})

// Despite the cycle, this returns cleanly with allowed=false
result, err := eng.Check(ctx, &engine.CheckRequest{
    ObjectType: "folder", ObjectID: "a",
    Permission: "view", SubjectType: "user", SubjectID: "nobody",
})
// result.Allowed = false
// err = nil (cycle was silently broken)
```

## See Also

- [Check](./check) — full permission check algorithm
- [Engine Configuration](./check#engine-configuration) — `MaxCheckDepth` setting
