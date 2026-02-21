---
id: overview
title: Multi-Tenancy Overview
sidebar_position: 1
---

# Multi-Tenancy

ZanGuard isolates authorization data by tenant for tuples, attributes, and changelog reads.

## How Tenant Identity Is Selected

API surfaces use different tenant selectors:

- Management tenant-scoped endpoints use path tenant ID: `/api/v1/t/{tenantID}/...`
- AuthZen runtime endpoints use header tenant ID: `X-Tenant-ID`

In both cases, the server builds a tenant context from the provided tenant ID before data operations.

## Isolation Guarantees in Current APIs

- Tuple/attribute/changelog operations are scoped to tenant context
- Runtime checks operate within the tenant from tenant context
- Normal HTTP APIs do not expose a cross-tenant tuple read/write path

## Tenant Model

```go
type Tenant struct {
    ID              string
    DisplayName     string
    ParentTenantID  string
    Status          TenantStatus   // pending | active | suspended | deleted
    SchemaMode      SchemaMode     // own | shared | inherited
    SharedSchemaRef string
    Config          TenantConfig
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

## Tenant ID Rules

Current validation accepts:

- lowercase letters, digits, hyphen
- must start/end with alphanumeric
- length: 2 to 128

Examples: `acme`, `my-org`, `tenant-42`, `a0`

## Tenant Configuration (Current State)

Tenant config is stored and returned in APIs. Some fields are merged into tenant context defaults.

Not all config fields are currently enforced by runtime code paths.

## Lifecycle Summary

- `pending`: tenant exists, write operations are rejected
- `active`: read/write operations allowed
- `suspended`: read operations allowed, write operations rejected
- `deleted`: operations are rejected

See [Tenant Lifecycle](./lifecycle) for exact behavior details.

## Schema Mode Reality

- `own`: fully supported through management schema endpoint
- `shared`: engine supports shared schema refs, but management API does not currently provide a shared-schema registration endpoint
- `inherited`: currently behaves like own (no parent merge logic yet)

See [Schema Modes](./schema-modes).

## See Also

- [Tenant Lifecycle](./lifecycle)
- [Schema Modes](./schema-modes)
- [Tenant Context](./context)
