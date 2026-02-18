---
id: postgresql
title: PostgreSQL Store
sidebar_position: 3
---

# PostgreSQL Store

The PostgreSQL store is the production backend for ZanGuard. It uses `pgx/v5` with connection pooling and is fully tenant-partitioned via a `tenant_id` column on every table.

## Setup

### 1. Run the migration

```bash
psql -d your_database -f migrations/001_initial.up.sql
```

### 2. Create the store

```go
import "zanguard/pkg/storage/postgres"

store, err := postgres.New(ctx, "postgres://user:pass@localhost:5432/zanguard")
if err != nil {
    log.Fatal(err)
}
```

### 3. With options

```go
store, err := postgres.New(ctx, dsn,
    postgres.WithMaxConns(20),
)
```

## Connection Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithMaxConns(n)` | `10` | Maximum pool size |

## DSN Format

```
postgres://username:password@host:port/database?sslmode=disable
```

Examples:

```
postgres://app:secret@localhost:5432/zanguard
postgres://app:secret@db.prod.internal:5432/zanguard?sslmode=require
```

## Database Schema

The migration (`migrations/001_initial.up.sql`) creates all required tables. Rollback is available via `migrations/001_initial.down.sql`.

Key tables:

| Table | Contents |
|-------|----------|
| `tenants` | Tenant records and configuration |
| `relation_tuples` | All authorization tuples |
| `object_attributes` | Per-object ABAC attribute maps (JSONB) |
| `subject_attributes` | Per-subject ABAC attribute maps (JSONB) |
| `changelog` | Append-only audit log |

All data tables include a `tenant_id` column. Queries always filter by tenant — there is no cross-tenant data leakage at the SQL level.

## Running Migrations

### Apply

```bash
psql -d zanguard -f migrations/001_initial.up.sql
```

### Roll back

```bash
psql -d zanguard -f migrations/001_initial.down.sql
```

## Production Recommendations

| Concern | Recommendation |
|---------|---------------|
| Connection pool | Set `MaxConns` to `(CPU cores × 2) + number_of_disks` |
| SSL | Always use `sslmode=require` in production |
| Credentials | Use environment variables, not hardcoded strings |
| Migrations | Run migrations in a controlled deployment step, not at startup |
| Indexes | The initial migration includes indexes on `tenant_id` + tuple fields |

## Example

```go
package main

import (
    "context"
    "log"
    "os"

    "zanguard/pkg/engine"
    "zanguard/pkg/model"
    "zanguard/pkg/storage/postgres"
    "zanguard/pkg/tenant"
)

func main() {
    ctx := context.Background()

    dsn := os.Getenv("DATABASE_URL")
    store, err := postgres.New(ctx, dsn, postgres.WithMaxConns(20))
    if err != nil {
        log.Fatal(err)
    }

    mgr := tenant.NewManager(store)
    mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)
    mgr.Activate(ctx, "acme")

    tCtx, _ := tenant.BuildContext(ctx, store, "acme")

    eng := engine.New(store, engine.DefaultConfig())
    // ... load schema, write tuples, check permissions
    _ = eng
    _ = tCtx
}
```

## See Also

- [In-Memory Store](./in-memory) — for testing
- [Storage Overview](./overview) — interface reference
- [Changelog](./changelog) — audit log
