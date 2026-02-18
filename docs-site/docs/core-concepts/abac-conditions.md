---
id: abac-conditions
title: ABAC Conditions
sidebar_position: 3
---

# ABAC Conditions

ZanGuard extends Zanzibar's pure relationship-based model with **Attribute-Based Access Control (ABAC)**. Conditions are safe expressions evaluated at check time against attributes stored on objects and subjects.

## How it Works

1. Attributes are stored separately from relation tuples — one map per object or subject
2. At check time, ZanGuard loads the relevant attributes and evaluates the condition expression
3. The condition must return a boolean

## Expression Language

Conditions use **[expr-lang](https://github.com/expr-lang/expr)** — a safe, sandboxed expression evaluator. It supports:

- Comparison operators: `==`, `!=`, `<`, `<=`, `>`, `>=`
- Logical operators: `&&`, `||`, `!`
- String operations: `contains`, `startsWith`, `endsWith`, `matches`
- Arithmetic: `+`, `-`, `*`, `/`
- Ternary: `condition ? trueVal : falseVal`
- Nil-safe access: `?.` (optional chaining)

## Available Variables

| Variable | Type | Contents |
|----------|------|----------|
| `object` | `map[string]any` | Attributes of the object being checked |
| `subject` | `map[string]any` | Attributes of the subject performing the action |
| `request` | `map[string]any` | Request-time context (IP, time, custom fields) |

## Setting Attributes

### Object Attributes

```go
store.SetObjectAttributes(ctx, "document", "readme", map[string]any{
    "classification": "internal",
    "department":     "engineering",
    "public":         false,
})
```

### Subject Attributes

```go
store.SetSubjectAttributes(ctx, "user", "thilina", map[string]any{
    "clearance_level": 4,
    "department":      "engineering",
    "region":          "us-west",
})
```

### Request Context

Pass context at check time via `CheckRequest.Context`:

```go
result, _ := eng.Check(ctx, &engine.CheckRequest{
    ObjectType:  "document",
    ObjectID:    "readme",
    Permission:  "view",
    SubjectType: "user",
    SubjectID:   "thilina",
    Context: map[string]any{
        "ip":   "10.0.0.1",
        "time": time.Now().UTC().Hour(),
    },
})
```

## Writing Conditions

### In the Schema (top-level gate)

A top-level condition is a final gate applied *after* the ReBAC check succeeds:

```yaml
permissions:
  delete:
    resolve: owner
    condition: "subject.clearance_level >= 3"
```

Both the relation check (is the subject an owner?) and the condition must pass.

### In a `union` list (inline)

An inline condition participates as one branch of a union:

```yaml
permissions:
  view:
    union:
      - viewer
      - condition: "object.public == true"
```

The document is viewable if the user is a `viewer` **or** if the document is public.

### In an `intersect` list

```yaml
permissions:
  sensitive_view:
    intersect:
      - viewer
      - condition: "subject.clearance_level >= object.required_clearance"
```

Both the viewer relation *and* the clearance condition must hold.

## Condition Examples

```yaml
# Simple equality
condition: "object.status == \"active\""

# Numeric comparison
condition: "subject.clearance_level >= 4"

# Logical AND
condition: "subject.department == object.department && subject.clearance_level >= 2"

# Logical OR
condition: "object.classification != \"restricted\" || subject.clearance_level >= 4"

# Request context (time-of-day access)
condition: "request.hour >= 9 && request.hour <= 17"

# Nil-safe access (won't panic if field is missing)
condition: "subject?.region == \"us-west\""

# String contains
condition: "object.tags contains \"public\""
```

## Full Example

**Schema:**

```yaml
version: "1.0"
types:
  user:
    attributes:
      clearance_level: int
      department: string

  document:
    attributes:
      classification: string
      department: string
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
```

**Go code:**

```go
// Set document as restricted
store.SetObjectAttributes(ctx, "document", "secret", map[string]any{
    "classification": "restricted",
})

// Bob has low clearance
store.SetSubjectAttributes(ctx, "user", "bob", map[string]any{
    "clearance_level": 2,
})

// Bob is a viewer
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType: "document", ObjectID: "secret",
    Relation: "viewer", SubjectType: "user", SubjectID: "bob",
})

// Check: Bob is a viewer (ReBAC passes) but condition fails (clearance too low)
result, _ := eng.Check(ctx, &engine.CheckRequest{
    ObjectType: "document", ObjectID: "secret",
    Permission: "view", SubjectType: "user", SubjectID: "bob",
})
fmt.Println(result.Allowed) // false

// Charlie has high clearance
store.SetSubjectAttributes(ctx, "user", "charlie", map[string]any{
    "clearance_level": 4,
})
store.WriteTuple(ctx, &model.RelationTuple{
    ObjectType: "document", ObjectID: "secret",
    Relation: "viewer", SubjectType: "user", SubjectID: "charlie",
})

result2, _ := eng.Check(ctx, &engine.CheckRequest{
    ObjectType: "document", ObjectID: "secret",
    Permission: "view", SubjectType: "user", SubjectID: "charlie",
})
fmt.Println(result2.Allowed) // true
```

## Compilation

Conditions are compiled **once at schema load time** into bytecode programs. At check time, only the attribute fetch and bytecode execution occur — no string parsing overhead.

## Safety

- Expressions run in a **sandboxed evaluator** — no file I/O, no network, no reflection
- Expressions must return a `bool` — the engine rejects non-boolean results with an error
- Unknown variables resolve to `nil` / zero values (no panics)

## Attribute Storage API

```go
// Objects
store.SetObjectAttributes(ctx, objectType, objectID, attrs)
attrs, err := store.GetObjectAttributes(ctx, objectType, objectID)

// Subjects
store.SetSubjectAttributes(ctx, subjectType, subjectID, attrs)
attrs, err := store.GetSubjectAttributes(ctx, subjectType, subjectID)
```

## See Also

- [Schema: Conditions](../schema/conditions) — condition syntax in the DSL
- [Examples: ABAC Clearance](../examples/abac-clearance) — full worked example
