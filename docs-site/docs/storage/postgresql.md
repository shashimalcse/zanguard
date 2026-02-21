---
id: postgresql
title: PostgreSQL Store
sidebar_position: 3
---

# PostgreSQL Store

PostgreSQL is the runtime storage backend for ZanGuard.

It stores tenants, tuples, attributes, and changelog entries in tenant-scoped tables and uses `pgx/v5` connection pooling.

## Quick Start with Docker Compose

From the repository root:

```bash
docker compose up --build
```

This starts:

- `postgres` database
- `zanguard` API server on `:1997`

Stop everything:

```bash
docker compose down
```

## Running the Server Locally (without Docker)

1. Prepare a PostgreSQL database.
2. Initialize schema from `deployments/postgres/init/010_schema.sql`.
3. Start the server with `DATABASE_URL`.

```bash
DATABASE_URL='postgres://user:pass@localhost:5432/zanguard?sslmode=disable' go run ./cmd/server/main.go
```

Optional pool size:

```bash
ZANGUARD_DB_MAX_CONNS=20 DATABASE_URL='postgres://user:pass@localhost:5432/zanguard?sslmode=disable' go run ./cmd/server/main.go
```

## `DATABASE_URL` Format

```text
postgres://username:password@host:port/database?sslmode=disable
```

Examples:

```text
postgres://zanguard:zanguard@localhost:5432/zanguard?sslmode=disable
postgres://app:secret@db.prod.internal:5432/zanguard?sslmode=require
```

## Programmatic Store Creation

```go
import "zanguard/pkg/storage/postgres"

store, err := postgres.New(ctx, dsn, postgres.WithMaxConns(20))
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

## Schema and Tables

Schema init script `deployments/postgres/init/010_schema.sql` creates:

- `tenants`
- `relation_tuples`
- `object_attributes`
- `subject_attributes`
- `changelog`
- `schema_versions`

## Operational Notes

- Use `sslmode=require` in production.
- Keep credentials in environment variables or secret managers.
- Keep schema init SQL aligned with application queries.
- Tune `ZANGUARD_DB_MAX_CONNS` based on DB capacity.

## See Also

- [Storage Overview](./overview)
- [Changelog](./changelog)
