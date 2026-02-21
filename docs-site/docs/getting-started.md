---
id: getting-started
title: Getting Started
sidebar_position: 2
---

# Getting Started

This guide starts ZanGuard with PostgreSQL using Docker Compose, then walks through a complete API flow:

1. Create and activate a tenant
2. Load a schema
3. Write tuples
4. Run a permission check

## Prerequisites

- Docker + Docker Compose
- `curl`

## Step 1: Start ZanGuard + PostgreSQL

From the project root:

```bash
docker compose up --build
```

When startup is complete, the API is available at:

- `http://localhost:1997`

Useful commands:

```bash
# Run in background
docker compose up --build -d

# Follow service logs
docker compose logs -f zanguard

# Stop everything
docker compose down
```

## Step 2: Create and Activate a Tenant

```bash
curl -X POST http://localhost:1997/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "id": "acme",
    "display_name": "Acme Corp",
    "schema_mode": "own"
  }'
```

Activate it:

```bash
curl -X POST http://localhost:1997/api/v1/tenants/acme/activate
```

## Step 3: Load a Schema (YAML)

Create a schema file:

```bash
cat > schema.yaml <<'YAML'
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
YAML
```

Upload it to the tenant:

```bash
curl -X PUT http://localhost:1997/api/v1/tenants/acme/schema \
  -H "Content-Type: application/yaml" \
  --data-binary @schema.yaml
```

Important: schema upload expects raw YAML, not JSON.

## Step 4: Write a Tuple

```bash
curl -X POST http://localhost:1997/api/v1/t/acme/tuples \
  -H "Content-Type: application/json" \
  -d '{
    "object_type": "document",
    "object_id": "readme",
    "relation": "viewer",
    "subject_type": "user",
    "subject_id": "alice"
  }'
```

## Step 5: Evaluate Access via AuthZen API

```bash
curl -X POST http://localhost:1997/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "subject": {"type": "user", "id": "alice"},
    "resource": {"type": "document", "id": "readme"},
    "action": {"name": "view"}
  }'
```

Expected response:

```json
{"decision": true}
```

## Common Errors

- `{"error":"missing X-Tenant-ID header"}`
Use `X-Tenant-ID` for AuthZen runtime endpoints (`/access/v1/...`).

- `parse schema: ... cannot unmarshal ...`
Send raw YAML with `--data-binary` and `Content-Type: application/yaml`.

## Next Steps

- [Management API](./api/management)
- [AuthZen Runtime API](./api/authzen)
- [Schema DSL](./schema/overview)
