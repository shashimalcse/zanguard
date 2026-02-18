---
id: conditions
title: Conditions
sidebar_position: 4
---

# Conditions

Conditions are ABAC expressions written in **[expr-lang](https://github.com/expr-lang/expr)** and embedded directly in your schema YAML.

## Where Conditions Appear

### 1. Top-level permission gate

Runs after the structural check (union / intersect / exclusion) passes. Both must allow.

```yaml
permissions:
  delete:
    resolve: owner
    condition: "subject.clearance_level >= 3"
```

### 2. Inline in `union` or `intersect`

Participates as one branch of the logical operation.

```yaml
permissions:
  view:
    union:
      - viewer
      - condition: "object.public == true"
```

## Expression Variables

| Variable | Contents |
|----------|----------|
| `object` | Attributes of the object (`map[string]any`) |
| `subject` | Attributes of the subject (`map[string]any`) |
| `request` | Request-time context passed in `CheckRequest.Context` |
| `tenant` | Reserved for future tenant-scoped attributes |

Access attributes with dot notation: `object.classification`, `subject.clearance_level`.

## Supported Operators

### Comparison

```
==   !=   <   <=   >   >=
```

### Logical

```
&&   ||   !
```

### Arithmetic

```
+   -   *   /   %
```

### String

```go
object.name contains "draft"
object.path startsWith "/public/"
subject.email endsWith "@acme.com"
subject.role matches "^admin.*"
```

### Ternary

```go
subject.vip ? "premium" : "standard"
```

### Nil-safe access

```go
subject?.department == "engineering"   // safe even if subject has no department
```

### Array membership

```go
object.tags contains "approved"
```

## Writing Multi-line Conditions

Use YAML block scalars for readability:

```yaml
condition: >-
  object.classification != "restricted" ||
  subject.clearance_level >= 4
```

Or keep it on one line:

```yaml
condition: "object.classification != \"restricted\" || subject.clearance_level >= 4"
```

## Condition Compilation

Conditions are compiled **at schema load time** using `expr.Compile`. The compiled program is stored in `ConditionExpr.Compiled` and reused on every check — there is no per-request parsing overhead.

```go
type ConditionExpr struct {
    Raw      string      // original expression string
    Compiled *vm.Program // pre-compiled bytecode
}
```

If a condition fails to compile (syntax error, type mismatch), `schema.Compile` returns an error immediately and the schema is rejected.

## Runtime Evaluation

At check time, `evaluateCondition` is called:

1. Object attributes are fetched from the store
2. Subject attributes are fetched from the store
3. The `env` map is built: `{object, subject, request}`
4. `expr.Run(compiled, env)` evaluates the expression
5. If the result is not a `bool`, the check returns an error

Unknown attribute keys resolve to `nil` (not a panic). Design your conditions to handle missing attributes gracefully using nil-safe access or default values.

## Example Conditions

```yaml
# Access based on document classification
condition: "object.classification == \"public\""

# Clearance gate
condition: "subject.clearance_level >= object.min_clearance"

# Department matching
condition: "subject.department == object.department"

# Time-of-day restriction (via request context)
condition: "request.hour >= 9 && request.hour < 18"

# OR: public OR high-clearance
condition: "object.public == true || subject.clearance_level >= 4"

# Complex compound
condition: >-
  (object.classification != "top_secret") ||
  (subject.clearance_level >= 5 && subject.department == "security")

# IP allowlist via request context
condition: "request.ip startsWith \"10.0.\""
```

## Passing Request Context

```go
result, _ := eng.Check(ctx, &engine.CheckRequest{
    ObjectType:  "document",
    ObjectID:    "report",
    Permission:  "view",
    SubjectType: "user",
    SubjectID:   "alice",
    Context: map[string]any{
        "ip":   "10.0.1.5",
        "hour": time.Now().Hour(),
        "mfa":  true,
    },
})
```

In the condition:

```yaml
condition: "request.mfa == true && request.hour >= 8"
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Condition returns non-bool | Returns error, check denies |
| Attribute key missing | Evaluates to `nil` (nil-safe ops handle this) |
| Condition fails to compile | `schema.Compile` returns error at startup |
| Runtime evaluation panics | Recovered and returned as error |

## See Also

- [ABAC Conditions](../core-concepts/abac-conditions) — conceptual overview
- [Examples: ABAC Clearance](../examples/abac-clearance) — full worked example
