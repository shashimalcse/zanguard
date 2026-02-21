---
id: context
title: Tenant Context
sidebar_position: 4
---

# Tenant Context

Tenant context is how ZanGuard binds operations to a tenant in Go code.

## Struct

```go
type TenantContext struct {
    TenantID   string
    Tenant     *Tenant
    SchemaHash string
    Config     *TenantConfig
}
```

Current note:

- `BuildContext` sets `TenantID`, `Tenant`, and merged `Config`
- `SchemaHash` exists but is not populated by current `BuildContext` path

## Building Context

```go
tenantCtx, err := tenant.BuildContext(ctx, store, "acme")
```

`BuildContext`:

- loads tenant from store
- merges tenant config defaults
- injects `TenantContext` into returned `context.Context`

## Reading Context

```go
tc := model.TenantFromContext(ctx)
if tc == nil {
    return model.ErrNoTenantContext
}
```

## API-Level Tenant Context Sources

In the built-in HTTP server:

- Management tenant-scoped endpoints use path tenant ID (`{tenantID}`)
- AuthZen runtime endpoints use `X-Tenant-ID` header

Both are converted into tenant context before store/engine calls.

## Manual Injection

If needed, you can inject manually:

```go
tc := &model.TenantContext{
    TenantID: "acme",
    Tenant:   t,
    Config:   &t.Config,
}
ctx = model.WithTenantContext(ctx, tc)
```

## Concurrency

Contexts are independent per request/goroutine.

You can safely run checks for different tenants concurrently as long as each request uses its own context.

## See Also

- [Multi-Tenancy Overview](./overview)
- [Tenant Lifecycle](./lifecycle)
- [Storage Overview](../storage/overview)
