---
id: permissions
title: Permissions DSL
sidebar_position: 3
---

# Permissions DSL

Permissions are declared inside type definitions. Each permission specifies a logical operation over relations, arrows, and conditions.

## Syntax

```yaml
permissions:
  <name>:
    resolve: <relation>       # sugar for union with one item
    union: [<ref>, ...]       # OR — any child grants access
    intersect: [<ref>, ...]   # AND — all children must grant access
    exclusion: [<ref>, ...]   # first MINUS rest
    condition: "<expr>"       # optional final ABAC gate
```

Only one of `resolve`, `union`, `intersect`, or `exclusion` may be used per permission.

## `resolve` — Single Relation Shorthand

```yaml
permissions:
  delete:
    resolve: owner
```

Equivalent to `union: [owner]`. A subject has `delete` if they hold the `owner` relation.

## `union` — OR Logic

```yaml
permissions:
  view:
    union:
      - viewer
      - editor
      - owner
      - parent->view
```

A subject has `view` if they satisfy **any** of the listed refs. Evaluation is left-to-right with short-circuit — the first match returns `allow`.

### Reference types in a list

| Syntax | Kind | Meaning |
|--------|------|---------|
| `viewer` | Relation ref | Subject holds the `viewer` relation |
| `parent->view` | Arrow | Subject can `view` via the `parent` relation |
| `condition: "expr"` | Inline condition | Expression evaluates to true |

## `intersect` — AND Logic

```yaml
permissions:
  sensitive_view:
    intersect:
      - viewer
      - condition: "subject.clearance_level >= object.required_clearance"
```

A subject has `sensitive_view` only if **all** refs are satisfied. Useful for combining ReBAC with mandatory ABAC gates.

:::note
If any single child denies, the whole permission denies immediately (short-circuit AND).
:::

## `exclusion` — Subtraction Logic

```yaml
permissions:
  edit:
    exclusion:
      - editor      # base: must be an editor
      - banned      # excluded: must NOT be banned
```

- The **first** ref is the base condition (must allow)
- **All remaining** refs are exclusions (must deny)
- A subject has `edit` if the base allows AND none of the exclusions allow

```yaml
# More complex example
permissions:
  publish:
    exclusion:
      - editor         # base: editors can publish
      - read_only      # exclusion: but not if they're read-only
      - suspended      # exclusion: and not if they're suspended
```

## Arrow Traversal (`->`)

```yaml
- parent->view
```

Arrow refs follow an object's relation to linked objects, then run a permission check on each linked object.

**Example:** If `document:design` has `parent → folder:root`, and `user:carol` can `view folder:root`, then `user:carol` can `view document:design` via `parent->view`.

```yaml
# Document inherits view from its parent folder
view:
  union:
    - viewer
    - parent->view
```

Arrow targets must be declared as relations on the object type:

```yaml
relations:
  parent:
    types: [folder]    # must exist for parent->view to work
```

## Inline Conditions

Use `condition:` as a ref inside `union` or `intersect`:

```yaml
permissions:
  view:
    union:
      - viewer
      - condition: "object.public == true"
```

The condition is evaluated with `object`, `subject`, and `request` variables. If true, it counts as an allow for that branch.

## Top-Level Condition Gate

A `condition` at the permission top level is evaluated **after** the structural check (union/intersect/exclusion) succeeds. Both must pass:

```yaml
permissions:
  delete:
    resolve: owner                           # ReBAC: must be owner
    condition: "subject.clearance_level >= 3" # ABAC: must have clearance
```

## Full Type Example

```yaml
types:
  document:
    attributes:
      classification: string
      department: string

    relations:
      owner:
        types: [user]
      editor:
        types: [user, group#member]
      viewer:
        types: [user, group#member]
      banned:
        types: [user]
      parent:
        types: [folder]

    permissions:
      # Anyone who is a viewer, editor, or owner can view
      # Also inherits from parent folder
      view:
        union:
          - viewer
          - editor
          - owner
          - parent->view

      # Editors and owners can edit — but not banned users
      edit:
        exclusion:
          - union:
          - editor
          - owner

      # Only owners can delete, and only with sufficient clearance
      delete:
        resolve: owner
        condition: "subject.clearance_level >= 3"

      # Same-department sensitive view
      dept_view:
        intersect:
          - viewer
          - condition: "subject.department == object.department"
```

## Permission Ref Types (Compiled)

After compilation, each permission child becomes a `PermissionRef`:

```go
type PermissionRef struct {
    RelationRef     string  // "viewer" → relation reference
    ArrowRef        string  // "parent" (used with ArrowPermission)
    ArrowPermission string  // "view"
    ConditionExpr   string  // raw expression for inline conditions
}
```

Use `ref.Kind()` to distinguish:

```go
switch child.Kind() {
case "relation":   // RelationRef is set
case "arrow":      // ArrowRef + ArrowPermission are set
case "condition":  // ConditionExpr is set
}
```

## See Also

- [Conditions](./conditions) — full condition expression syntax
- [Engine: Check](../engine/check) — how the engine evaluates these rules
- [Core Concepts: Permissions](../core-concepts/permissions) — conceptual overview
