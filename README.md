# ZanGuard

A lightweight, [Zanzibar](https://research.google/pubs/pub48190/)-inspired authorization engine built in Go, designed for native multi-tenancy and [AuthZen 1.0](https://openid.github.io/authzen/) compliance.

ZanGuard provides **Relationship-Based Access Control (ReBAC)** as its core model, with attribute overlays for ABAC-like expressiveness.

## Features

- **Multi-tenant** — Every tuple, schema, and event is tenant-scoped with full data isolation
- **ReBAC + ABAC** — Define relations and permissions in a simple YAML schema; add attribute conditions where needed
- **AuthZen 1.0** — Standards-compliant evaluation API out of the box
- **PostgreSQL storage** — Durable, queryable storage with soft-deletes and a changelog
- **In-memory caching** — Per-instance, tenant-partitioned cache with configurable TTLs
- **Admin console** — Next.js web UI for managing schemas, tuples, and attributes
- **Embeddable or standalone** — Run as a sidecar, a standalone service, or an embedded Go library

## Quick Start

The fastest way to run ZanGuard locally is with Docker Compose:

```bash
docker compose up --build -d
```

This starts a PostgreSQL instance and the ZanGuard server on port **1997**.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | *(required)* | PostgreSQL connection string |
| `ZANGUARD_ADDR` | `:1997` | Listen address |
| `ZANGUARD_DB_MAX_CONNS` | `10` | PostgreSQL connection pool size |

### Run Without Docker

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/zanguard?sslmode=disable"
make run
```

## Schema

Schemas are written in YAML and define object types, relations, and permissions. Load one via the management API.

```yaml
# configs/examples/gdrive.zanguard.yaml
version: "1.0"

types:
  user:
    attributes:
      clearance_level: int
      department: string

  document:
    attributes:
      classification: string
    relations:
      owner:
        types: [user]
      editor:
        types: [user]
      viewer:
        types: [user]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - editor
          - owner
          - parent->view   # inherited from parent folder
      edit:
        union:
          - editor
          - owner
      delete:
        resolve: owner
```

Permissions support:
- `resolve` — direct relation lookup
- `union` — logical OR over multiple relations or sub-permissions
- `->` — permission inheritance across relation hops (e.g. `parent->view`)

## API

### Management API

All management endpoints require a `X-Tenant-ID` header.

| Method | Path | Description |
|---|---|---|
| `POST` | `/tenants` | Create a tenant |
| `GET` | `/tenants` | List tenants |
| `POST` | `/tenants/{id}/activate` | Activate a tenant |
| `POST` | `/tenants/{id}/suspend` | Suspend a tenant |
| `POST` | `/schema` | Load a schema for the current tenant |
| `GET` | `/schema` | Get the current schema |
| `POST` | `/tuples` | Write relationship tuples |
| `GET` | `/tuples` | Read / filter tuples |
| `DELETE` | `/tuples` | Delete tuples |
| `POST` | `/check` | Check a permission |
| `POST` | `/expand` | Expand a permission tree |
| `GET` | `/changelog` | Stream the tuple changelog |

### AuthZen Evaluation API

```
POST /access/v1/evaluation
POST /access/v1/evaluations   (batch)
```

Example request:

```json
{
  "subject":  { "type": "user", "id": "alice" },
  "action":   { "name": "view" },
  "resource": { "type": "document", "id": "budget-2025" }
}
```

Example response:

```json
{ "decision": true }
```

## Development

```bash
make build          # compile
make test           # unit tests
make test-integration  # integration tests (requires DATABASE_URL)
make lint           # go vet
make tidy           # go mod tidy
```

### Project Layout

```
cmd/server/         # main entrypoint
pkg/
  api/              # HTTP handlers and routing
  engine/           # ReBAC check engine
  model/            # domain types
  schema/           # schema parser and compiler
  storage/          # storage interfaces and Postgres implementation
  tenant/           # tenant lifecycle management
configs/examples/   # example schemas
console/            # Next.js admin console
deployments/        # Docker / Postgres init scripts
```

## License

MIT
