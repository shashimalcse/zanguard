---
id: permissions
title: Permissions
sidebar_position: 2
---

# Permissions

A **permission** is a named rule defined on an object type in your schema. It describes *how* access is derived from the underlying relation tuples using logical operations.

## Permissions vs Relations

| Concept | Definition | Stored as tuple? |
|---------|------------|-----------------|
| **Relation** | A named edge between an object and a subject | Yes |
| **Permission** | A computed rule over one or more relations | No — evaluated at check time |

Relations are facts. Permissions are derived from those facts.

## Permission Operations

### `resolve` — Direct relation

The simplest permission: a subject has the permission if they hold the named relation.

```yaml
permissions:
  delete:
    resolve: owner   # allowed if the subject is an owner
```

### `union` — Any match wins

A subject has the permission if they satisfy **any** of the listed relations or arrows.

```yaml
permissions:
  view:
    union:
      - viewer        # direct viewer relation
      - editor        # editors can also view
      - owner         # owners can also view
      - parent->view  # or if they can view the parent folder
```

Short-circuit evaluation: the first match returns `allow` immediately.

### `intersect` — All must match

A subject has the permission only if they satisfy **all** listed conditions.

```yaml
permissions:
  sensitive_view:
    intersect:
      - viewer
      - condition: "subject.clearance_level >= 3"
```

### `exclusion` — Base minus excluded

A subject has the permission if they satisfy the **first** condition but **none** of the subsequent ones.

```yaml
permissions:
  edit:
    exclusion:
      - editor        # base: must be an editor
      - banned        # but: not banned
```

## Arrow Traversal (`->`)

The `->` operator follows an object's relation to another object, then checks a permission on that linked object. This enables **inherited permissions** through hierarchies.

```yaml
# A user can view a document if they can view its parent folder
view:
  union:
    - viewer
    - parent->view   # follow "parent" relation, then check "view" on that object
```

**How it works:**
1. Find all objects linked via `document:doc#parent`
2. For each linked `folder:X`, run `Check(folder:X, view, subject)`
3. If any allows → allow

## ABAC Conditions as Permission Children

Inline conditions can appear anywhere in a `union` or `intersect` list:

```yaml
permissions:
  view:
    union:
      - viewer
      - condition: "object.public == true"
```

See [ABAC Conditions](./abac-conditions) for full syntax.

## Top-Level Condition

A condition can also be placed at the top level of any permission as a *final gate*. The ReBAC check runs first; the ABAC condition runs only if ReBAC passes.

```yaml
permissions:
  delete:
    resolve: owner
    condition: "subject.clearance_level >= 3"
```

Both must be true for the permission to allow.

## Permission Resolution Path

When a check succeeds, ZanGuard returns a `ResolutionPath` showing which tuple granted access:

```go
result, _ := eng.Check(ctx, req)

fmt.Println(result.Allowed)          // true
fmt.Println(result.ResolutionPath)
// ["document:readme#viewer@user:thilina"]
```

This is useful for debugging and audit trails.

## Evaluation Order

For `union`, children are evaluated **left to right** with short-circuit:

```yaml
view:
  union:
    - viewer          # checked first
    - editor          # checked only if viewer fails
    - owner           # checked only if editor fails
    - parent->view    # checked last
```

For `intersect`, all children are evaluated and **all must pass**:

```yaml
sensitive_view:
  intersect:
    - viewer          # must pass
    - condition: "..." # must also pass
```

## Defining Permissions in Schema

Permissions are declared inside type definitions in your schema YAML. See [Schema: Permissions](../schema/permissions) for the full syntax reference.

## See Also

- [ABAC Conditions](./abac-conditions) — attribute-based rules
- [Schema: Permissions](../schema/permissions) — full DSL reference
- [Engine: Check](../engine/check) — how the engine evaluates permissions
