---
id: getting-started
title: Getting Started
sidebar_position: 2
---

# Getting Started

This guide walks you through running your first ZanGuard permission check from scratch using the in-memory backend.

## Prerequisites

- Go 1.23 or later
- The `zanguard` module available locally

## Step 1 — Create a Store

The store holds all relation tuples and tenant data. Use the in-memory store for local development and testing.

```go
import "zanguard/pkg/storage/memory"

store := memory.New()
```

## Step 2 — Create and Activate a Tenant

All data in ZanGuard is scoped to a tenant. Create one and activate it before writing any tuples.

```go
import (
    "context"
    "zanguard/pkg/model"
    "zanguard/pkg/tenant"
)

ctx := context.Background()
mgr := tenant.NewManager(store)

// Creates tenant in "pending" state
acme, err := mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)

// Transition to "active" — required before any reads or writes
err = mgr.Activate(ctx, acme.ID)
```

## Step 3 — Build a Tenant Context

Every store operation and engine check requires a tenant-scoped `context.Context`.

```go
tenantCtx, err := tenant.BuildContext(ctx, store, "acme")
```

## Step 4 — Define a Schema

ZanGuard needs a schema to know what types, relations, and permissions exist. Write it in YAML:

```yaml
# schema.yaml
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

Load and compile it in Go:

```go
import (
    "os"
    "zanguard/pkg/schema"
)

data, _ := os.ReadFile("schema.yaml")
raw, _  := schema.Parse(data)
cs, _   := schema.Compile(raw, data)
```

## Step 5 — Create the Engine

```go
import "zanguard/pkg/engine"

eng := engine.New(store, engine.DefaultConfig())
eng.LoadSchema("acme", cs)
```

## Step 6 — Write a Relation Tuple

```go
import "zanguard/pkg/model"

store.WriteTuple(tenantCtx, &model.RelationTuple{
    ObjectType:  "document",
    ObjectID:    "readme",
    Relation:    "viewer",
    SubjectType: "user",
    SubjectID:   "thilina",
})
```

This asserts: **`user:thilina` is a `viewer` of `document:readme`**.

## Step 7 — Check a Permission

```go
result, err := eng.Check(tenantCtx, &engine.CheckRequest{
    ObjectType:  "document",
    ObjectID:    "readme",
    Permission:  "view",
    SubjectType: "user",
    SubjectID:   "thilina",
})

fmt.Println(result.Allowed)          // true
fmt.Println(result.ResolutionPath)   // [document:readme#viewer@user:thilina]
```

## Full Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "zanguard/pkg/engine"
    "zanguard/pkg/model"
    "zanguard/pkg/schema"
    "zanguard/pkg/storage/memory"
    "zanguard/pkg/tenant"
)

const mySchema = `
version: "1.0"
types:
  user: {}
  document:
    relations:
      viewer:
        types: [user]
    permissions:
      view:
        resolve: viewer
`

func main() {
    ctx := context.Background()
    store := memory.New()

    mgr := tenant.NewManager(store)
    mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)
    mgr.Activate(ctx, "acme")

    tenantCtx, _ := tenant.BuildContext(ctx, store, "acme")

    data := []byte(mySchema)
    raw, _ := schema.Parse(data)
    cs, _ := schema.Compile(raw, data)

    eng := engine.New(store, engine.DefaultConfig())
    eng.LoadSchema("acme", cs)

    store.WriteTuple(tenantCtx, &model.RelationTuple{
        ObjectType: "document", ObjectID: "readme",
        Relation: "viewer", SubjectType: "user", SubjectID: "thilina",
    })

    result, err := eng.Check(tenantCtx, &engine.CheckRequest{
        ObjectType: "document", ObjectID: "readme",
        Permission: "view", SubjectType: "user", SubjectID: "thilina",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("allowed=%v path=%v\n", result.Allowed, result.ResolutionPath)
    // allowed=true path=[document:readme#viewer@user:thilina]
}
```

Run it:

```bash
go run ./cmd/server/main.go
```

## Running the API Server

ZanGuard ships with a standalone HTTP server that exposes the Management API and the AuthZen 1.0 Runtime API. Start it with:

```bash
go run ./cmd/server/main.go
```

The server listens on `:8080` by default. Override the address with the `ZANGUARD_ADDR` environment variable:

```bash
ZANGUARD_ADDR=:9090 go run ./cmd/server/main.go
```

All data-plane and runtime endpoints require a `X-Tenant-ID` request header that identifies the target tenant. For example, to write a tuple via the API after starting the server and creating a tenant named `acme`:

```bash
curl -X POST http://localhost:8080/api/v1/tuples \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "object_type": "document",
    "object_id": "readme",
    "relation": "viewer",
    "subject_type": "user",
    "subject_id": "thilina"
  }'
```

See the [API Reference](./api/overview) for the full endpoint listing.

## What's Next?

- [Relation Tuples](./core-concepts/relation-tuples) — understand the data model
- [Schema DSL](./schema/overview) — write richer authorization models
- [ABAC Conditions](./core-concepts/abac-conditions) — add attribute-based rules
- [Multi-Tenancy](./multi-tenancy/overview) — manage multiple tenants
- [API Reference](./api/overview) — HTTP API endpoints
