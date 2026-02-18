---
id: context
title: Tenant Context
sidebar_position: 4
---

# Tenant Context

The `TenantContext` is the mechanism that binds every storage and engine operation to a specific tenant. It is stored in Go's `context.Context` and propagated through the call stack.

## The `TenantContext` Struct

```go
type TenantContext struct {
    TenantID   string        // the tenant's ID
    Tenant     *Tenant       // the full tenant record
    SchemaHash string        // SHA-256 hash of the tenant's schema
    Config     *TenantConfig // tenant quotas and configuration
}
```

## Building a Context

Use `tenant.BuildContext` to create a tenant-scoped context from a base context:

```go
tenantCtx, err := tenant.BuildContext(ctx, store, "acme")
if err != nil {
    // returns storage.ErrTenantNotFound if tenant doesn't exist
}
```

`BuildContext` fetches the tenant from the store and injects a `TenantContext` into the returned context. All subsequent store and engine calls using this context will be scoped to `"acme"`.

## Injecting Manually

If you already have a `Tenant` object, inject it directly:

```go
tc := &model.TenantContext{
    TenantID: "acme",
    Tenant:   tenant,
    Config:   &tenant.Config,
}
ctx = model.WithTenantContext(ctx, tc)
```

## Reading the Context

Anywhere in the call stack:

```go
// Returns nil if not present
tc := model.TenantFromContext(ctx)
if tc == nil {
    return model.ErrNoTenantContext
}

// Panics if not present (use in middleware that guarantees injection)
tc := model.MustTenantFromContext(ctx)
```

## Context Propagation Pattern

The typical pattern in an HTTP handler or gRPC interceptor:

```go
func TenantMiddleware(store storage.TupleStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID := r.Header.Get("X-Tenant-ID")
            if tenantID == "" {
                http.Error(w, "missing tenant", http.StatusBadRequest)
                return
            }

            ctx, err := tenant.BuildContext(r.Context(), store, tenantID)
            if err != nil {
                http.Error(w, "tenant not found", http.StatusNotFound)
                return
            }

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

Downstream handlers call `model.TenantFromContext(r.Context())` to get the tenant.

## Context Key Safety

ZanGuard uses an unexported struct type as the context key:

```go
type tenantContextKeyType struct{}
var tenantContextKey = tenantContextKeyType{}
```

This prevents key collisions with any other package using `context.WithValue`.

## Multiple Tenants in One Process

Each goroutine or request can carry its own tenant context:

```go
// These two contexts are independent — no shared state
acmeCtx, _ := tenant.BuildContext(ctx, store, "acme")
globexCtx, _ := tenant.BuildContext(ctx, store, "globex")

// Runs in parallel — each scoped to its own tenant
go func() { eng.Check(acmeCtx, req) }()
go func() { eng.Check(globexCtx, req) }()
```

## Error: No Tenant Context

If you call the engine or store without a tenant context, you get:

```go
var ErrNoTenantContext = errors.New("no tenant context in request")
```

This is a programming error — always inject the tenant context before making storage or engine calls.

## See Also

- [Tenant Lifecycle](./lifecycle) — creating and managing tenants
- [Schema Modes](./schema-modes) — how the schema is resolved from context
- [Storage Overview](../storage/overview) — how context is used in storage ops
