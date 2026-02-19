# ZanGuard — Recommended Improvements

Prioritized list of improvements across security, reliability, performance,
testing, and maintainability. Each item includes the affected files and a
brief rationale.

---

## 1. Security (Critical)

### 1.1 Add request body size limits

**Files:** `pkg/api/management.go:128`, all handlers using `readJSON`

`handleLoadSchema` calls `io.ReadAll(r.Body)` with no size cap. An attacker
can send an arbitrarily large payload and exhaust server memory. The same risk
applies to every handler that decodes JSON from the request body.

**Recommendation:** Wrap incoming bodies with `http.MaxBytesReader` in a
middleware or at the top of each handler:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
```

### 1.2 Configure HTTP server timeouts

**File:** `pkg/api/server.go:105`

`http.ListenAndServe` creates a server with zero timeouts. This leaves the
service vulnerable to slow-client and connection-exhaustion attacks.

**Recommendation:** Use an explicit `http.Server`:

```go
srv := &http.Server{
    Addr:         addr,
    Handler:      s,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  60 * time.Second,
}
return srv.ListenAndServe()
```

### 1.3 Add input validation on tuple and attribute fields

**Files:** `pkg/api/types.go`, `pkg/api/management.go`

`TupleRequest` fields (`object_type`, `object_id`, `relation`, etc.) are
accepted verbatim with no length, character, or format checks. Malformed
values propagate to storage and can cause unexpected behaviour.

**Recommendation:** Add a validation function that enforces:
- Max length (e.g. 128 chars)
- Allowed character set (`[a-zA-Z0-9_\-.]`)
- Non-empty required fields

### 1.4 Add authentication to the API

The only tenant isolation mechanism is the `X-Tenant-ID` header, which any
caller can set to any value. There is no authentication or authorization layer
protecting management or AuthZen endpoints.

**Recommendation:** Introduce API-key or JWT-based authentication middleware
before tenant resolution. Even a shared bearer token is better than open
access.

### 1.5 Replace `MustTenantFromContext` panic with error return

**File:** `pkg/model/tenant.go:89-95`

`MustTenantFromContext` panics if the tenant context is missing. In a server
process this crashes the entire service instead of returning a 500 to the
single request.

**Recommendation:** Remove the `Must` variant or convert it to return
`(*TenantContext, error)`. Callers should handle the error path explicitly.

---

## 2. Error Handling (Critical)

### 2.1 Stop ignoring errors during tenant creation

**File:** `pkg/api/management.go:45`

```go
_ = s.store.UpdateTenant(r.Context(), t)
```

If the update of `ParentTenantID` / `SharedSchemaRef` fails, the caller
receives a 201 but the tenant is in an incomplete state.

**Recommendation:** Return an error to the client or roll back the tenant
creation.

### 2.2 Handle errors after activate/suspend

**Files:** `pkg/api/management.go:102, 113`

```go
t, _ := s.mgr.Get(r.Context(), tenantID)
writeJSON(w, http.StatusOK, t)
```

If `Get` fails after a successful state transition, `t` is nil, and the
JSON encoder will write `null`. This is confusing and hides real errors.

**Recommendation:** Check the error and return an appropriate status code.

### 2.3 Return errors for invalid query parameters

**Files:** `pkg/api/management.go:62, 65, 386`

```go
filter.Limit, _ = strconv.Atoi(v)
```

Non-numeric values silently default to 0. Callers have no way to tell
whether their pagination parameters were accepted.

**Recommendation:** Return `400 Bad Request` when parsing fails.

### 2.4 Do not swallow attribute retrieval errors in condition evaluation

**File:** `pkg/engine/condition.go:19-22`

```go
objAttrs, err := store.GetObjectAttributes(ctx, req.ObjectType, req.ObjectID)
if err != nil {
    objAttrs = map[string]any{} // silent fallback
}
```

A transient storage error here causes the ABAC condition to evaluate with
empty attributes, potentially granting or denying access incorrectly. For an
authorization engine, this is a safety-critical bug.

**Recommendation:** Propagate the error and deny the request.

### 2.5 Handle JSON encoding failures in `writeJSON`

**File:** `pkg/api/helpers.go:17`

```go
_ = json.NewEncoder(w).Encode(v)
```

If encoding fails (e.g. cyclic type), the client receives a truncated
response with no error signal.

**Recommendation:** Log the error. If the header has not yet been flushed,
return a 500.

---

## 3. Observability (Critical)

### 3.1 Add structured logging to the authorization engine

**Files:** `pkg/engine/check.go`, `pkg/engine/condition.go`

The engine is the most critical path in the system, yet it produces zero log
output. Permission denials, cycle detections, depth-exceeded errors, and
condition failures are all invisible in production.

**Recommendation:** Accept a `*slog.Logger` in `engine.New()` and emit
`Debug`-level logs for each check step. Emit `Warn` for cycles and depth
exceeded.

### 3.2 Log AuthZen evaluation errors

**File:** `pkg/api/authzen.go:36-39`

Errors from `s.eng.Check` are silently converted to `decision: false`.
While the AuthZen spec requires a 200, the error should still be logged
server-side so operators can diagnose failures.

### 3.3 Add request-level correlation IDs

No request ID is injected into the context. When multiple requests are
processed concurrently, log lines are impossible to correlate.

**Recommendation:** Add middleware that generates or reads an
`X-Request-ID` header and attaches it to the logger and context.

### 3.4 Add metrics (latency, error rates, cache hits)

There are no application-level metrics. Basic Prometheus counters and
histograms for `check_latency`, `check_total`, `tuple_count`, and
`db_query_duration` would provide essential production visibility.

---

## 4. Testing (Critical)

### 4.1 Add API handler tests

**Files:** `pkg/api/` (no test files exist)

All 16+ HTTP handlers have zero test coverage. This is the primary public
interface and needs at least:
- Happy-path tests for each endpoint
- Error-path tests (bad JSON, missing headers, not-found, etc.)
- Tenant isolation tests (verify cross-tenant data is unreachable)

### 4.2 Add PostgreSQL integration tests

**File:** `pkg/storage/postgres/store.go` (no test file)

The Postgres store is the production storage backend, yet it has no
integration tests. Queries with dynamic parameter building
(`pkg/storage/postgres/store.go:225-254`) are especially risky without
coverage.

**Recommendation:** Use a test container or `pgx` mock to cover:
- CRUD operations
- Filter combinations
- Concurrent writes
- Soft-delete correctness

### 4.3 Fix ignored errors in test setup

**File:** `pkg/engine/check_test.go:71-72`

```go
_, _ = mgr.Create(ctx, "test-tenant", "Test", model.SchemaOwn)
_ = mgr.Activate(ctx, "test-tenant")
```

Tests that ignore setup errors can pass vacuously. Use `t.Fatal` or
`require.NoError`.

### 4.4 Add schema parser and compiler edge-case tests

**Files:** `pkg/schema/parser.go`, `pkg/schema/compiler.go`

While basic tests exist, edge cases are not covered:
- Malformed YAML
- Unknown/extra fields
- Circular permission references
- Empty schema

---

## 5. Performance (Medium)

### 5.1 Batch tuple inserts

**File:** `pkg/storage/postgres/store.go:187-193`

`WriteTuples` executes one `INSERT` per tuple in a loop. For the batch API
endpoint this means N round-trips to Postgres.

**Recommendation:** Use `pgx.Batch` or a single multi-row `INSERT`:

```go
batch := &pgx.Batch{}
for _, t := range tuples {
    batch.Queue(insertSQL, args...)
}
results := s.pool.SendBatch(ctx, batch)
defer results.Close()
```

### 5.2 Use a transaction for `PurgeTenantData`

**File:** `pkg/storage/postgres/store.go:533-552`

Four separate `DELETE` statements run without a transaction. If the process
crashes mid-purge, the tenant ends up in a partially deleted state.

**Recommendation:** Wrap in `pool.BeginTx`.

### 5.3 Add pagination limits

**Files:** `pkg/api/management.go` (ReadTuples, ListTenants)

Neither `ReadTuples` nor `ListTenants` enforces a maximum page size. A
request for all tuples in a large tenant will attempt to load them all into
memory.

**Recommendation:** Cap `limit` at a reasonable default (e.g. 1000) and
document the maximum.

### 5.4 Evaluate batch AuthZen checks concurrently

**File:** `pkg/api/authzen.go:63-78`

Batch evaluations run sequentially in a loop. For large batches this is
unnecessarily slow.

**Recommendation:** Use a bounded worker pool (`errgroup` with `SetLimit`)
to evaluate items concurrently.

---

## 6. Code Quality / Maintainability (Medium)

### 6.1 Extract tenant context resolution into middleware

**File:** `pkg/api/management.go` (10+ handlers)

Every handler that needs a tenant context repeats:

```go
tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
if err != nil {
    writeError(w, http.StatusBadRequest, err.Error())
    return
}
```

**Recommendation:** Create a middleware that resolves the tenant and injects
it into the context. Handlers that need it can call
`model.TenantFromContext(r.Context())`.

### 6.2 Remove dead code in condition evaluation

**File:** `pkg/engine/condition.go:29-30`

```go
tc := ctx.Value(struct{}{}) // not used directly
_ = tc
```

This reads a value from context using an anonymous struct key (which will
never match `tenantContextKey`) and discards it. Remove.

### 6.3 Consolidate duplicate attribute handlers

**Files:** `pkg/api/management.go:296-370`

The four attribute handlers (get/set for objects/subjects) are nearly
identical. Consider a generic helper:

```go
func (s *Server) handleSetAttributes(kind string) http.HandlerFunc { ... }
```

### 6.4 Use a query builder or squirrel for dynamic SQL

**File:** `pkg/storage/postgres/store.go:127-164, 212-274`

Dynamic `WHERE` clause construction with `fmt.Sprintf` and manual `argN`
tracking is error-prone (note: `argN` is not incremented after the last
filter in `ListTenants`). A lightweight builder such as
`github.com/Masterminds/squirrel` eliminates this class of bug.

### 6.5 Improve `errStatus` to cover more sentinel errors

**File:** `pkg/api/helpers.go`

Only `ErrTenantNotFound` is mapped. Storage errors like
`ErrDuplicateTuple` or `ErrNoTenantContext` should also map to specific
HTTP status codes instead of defaulting to 500.

---

## 7. Configuration (Low)

### 7.1 Support a configuration file or full env-var set

**File:** `cmd/server/main.go:18-20`

Only `ZANGUARD_ADDR` is configurable via environment. All other values
(max check depth, default connection pool size, changelog limit, default
tenant config) are hardcoded.

**Recommendation:** Introduce a config struct loaded from environment
variables or a YAML file, covering at minimum:
- Listen address and TLS
- Postgres DSN and pool size
- Max check depth
- Default pagination limits
- Tenant config defaults

### 7.2 Make MaxCheckDepth configurable per tenant

**File:** `pkg/engine/engine.go:19-22`

The hardcoded depth of 25 is reasonable for most schemas but may be too
shallow for deeply nested hierarchies. Allow per-tenant override via
`TenantConfig`.

---

## Summary by Priority

| Priority | Area | Items |
|----------|------|-------|
| **P0** | Security | 1.1, 1.2, 1.3, 1.4, 1.5 |
| **P0** | Error handling | 2.1, 2.2, 2.3, 2.4, 2.5 |
| **P0** | Observability | 3.1, 3.2, 3.3, 3.4 |
| **P1** | Testing | 4.1, 4.2, 4.3, 4.4 |
| **P2** | Performance | 5.1, 5.2, 5.3, 5.4 |
| **P2** | Code quality | 6.1, 6.2, 6.3, 6.4, 6.5 |
| **P3** | Configuration | 7.1, 7.2 |
