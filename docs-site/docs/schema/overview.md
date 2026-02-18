---
id: overview
title: Schema Overview
sidebar_position: 1
---

# Schema DSL

ZanGuard uses a **YAML-based schema DSL** to define your authorization model. The schema declares:

- **Types** — the kinds of objects and subjects in your system (e.g. `user`, `document`, `folder`)
- **Relations** — named edges between object types and subject types (e.g. `viewer`, `owner`)
- **Permissions** — computed rules derived from relations (e.g. `view = viewer | editor | parent->view`)
- **Conditions** — ABAC expressions evaluated against attributes

The schema is compiled once at startup into an optimized in-memory structure. All permission checks at runtime reference the compiled schema — never the raw YAML.

## Structure

```yaml
version: "1.0"

types:
  <type_name>:
    attributes:                  # optional: ABAC attribute declarations
      <attr_name>: <attr_type>

    relations:                   # optional: named subject relationships
      <relation_name>:
        types: [<type>, ...]     # allowed subject types

    permissions:                 # optional: computed access rules
      <permission_name>:
        resolve: <relation>      # OR
        union: [...]             # OR
        intersect: [...]         # OR
        exclusion: [...]
        condition: "<expr>"      # optional ABAC gate
```

## Minimal Example

```yaml
version: "1.0"

types:
  user: {}

  document:
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
      delete:
        resolve: owner
```

## Full Google Drive–Style Example

```yaml
version: "1.0"

types:
  user:
    attributes:
      clearance_level: int
      department: string
      region: string

  group:
    relations:
      member:
        types: [user]
      admin:
        types: [user]
    permissions:
      manage:
        resolve: admin

  folder:
    relations:
      owner:
        types: [user]
      viewer:
        types: [user, group#member]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - owner
          - parent->view
      edit:
        resolve: owner

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
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - editor
          - owner
          - parent->view
      edit:
        union:
          - editor
          - owner
      delete:
        resolve: owner
      share:
        resolve: owner
```

## Loading a Schema

```go
import (
    "os"
    "zanguard/pkg/schema"
)

// From a file
raw, err := schema.ParseFile("configs/examples/gdrive.zanguard.yaml")

// From bytes
data, _ := os.ReadFile("schema.yaml")
raw, err := schema.Parse(data)

// Compile
cs, err := schema.Compile(raw, data)

// Validate
if errs := schema.Validate(cs); len(errs) > 0 {
    for _, e := range errs {
        log.Printf("schema error: %v", e)
    }
}

// Register with the engine
eng.LoadSchema("acme", cs)
```

## Schema Hashing

Each compiled schema contains a SHA-256 hash of the source YAML:

```go
fmt.Println(cs.Hash)       // e.g. "a3f1b2c4..."
fmt.Println(cs.CompiledAt) // time of compilation
fmt.Println(cs.Version)    // "1.0"
```

This hash is used for cache invalidation — if the schema changes, cached check results for the old hash are stale.

## Validation

`schema.Validate` checks the compiled schema for:

- Arrow refs pointing to undefined relations
- Permission refs pointing to undefined relations
- Userset type refs pointing to undefined types or relations

```go
errs := schema.Validate(cs)
// errs is []error — empty means the schema is valid
```

## Schema Modes

When multiple tenants share a schema, you can use **shared** or **inherited** schema modes instead of loading the same schema per tenant. See [Schema Modes](../multi-tenancy/schema-modes).

## Reference Pages

- [Types and Relations](./types-and-relations)
- [Permissions](./permissions)
- [Conditions](./conditions)
