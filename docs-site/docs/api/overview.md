---
id: overview
title: API Overview
sidebar_position: 1
---

# API Overview

ZanGuard exposes two HTTP API surfaces over a single server process:

| Surface | Base path | Purpose |
|---------|-----------|---------|
| Management API | `/api/v1/` | Tenant lifecycle, schema loading, tuple writes, attributes, changelog |
| AuthZen Runtime API | `/access/v1/` | AuthZen 1.0-compliant permission evaluation |

## OpenAPI Specs

- [Management API OpenAPI (`management-v1.yaml`)](/openapi/management-v1.yaml)
- [AuthZen Runtime API OpenAPI (`runtime-authzen-v1.yaml`)](/openapi/runtime-authzen-v1.yaml)

Download example:

```bash
curl -LO http://localhost:1997/openapi/management-v1.yaml
curl -LO http://localhost:1997/openapi/runtime-authzen-v1.yaml
```

## Starting the server

Recommended:

```bash
docker compose up --build
```

Or run directly with Go:

```bash
DATABASE_URL='postgres://zanguard:zanguard@localhost:5432/zanguard?sslmode=disable' go run ./cmd/server/main.go
```

By default the server listens on `:1997`. Override the address with the `ZANGUARD_ADDR` environment variable:

```bash
ZANGUARD_ADDR=:9090 DATABASE_URL='postgres://zanguard:zanguard@localhost:5432/zanguard?sslmode=disable' go run ./cmd/server/main.go
```

## Base URL

All examples in this reference assume the server is reachable at `http://localhost:1997`. Replace this with your deployment address as needed.

## Content type

All request and response bodies are JSON. Set the following header on every request that sends a body:

```
Content-Type: application/json
```

The single exception is `PUT /api/v1/tenants/{tenantID}/schema`, which accepts a raw YAML body. Use:

```
Content-Type: application/yaml
```

## Tenant identification

Tenant identification differs by API surface:

- **Management tenant-scoped endpoints** (`tuples`, `attributes`, `changelog`, `expand`) use a tenant path prefix:

```
/api/v1/t/{tenantID}/...
```

- **AuthZen runtime endpoints** (`/access/v1/evaluation`, `/access/v1/evaluations`) require the `X-Tenant-ID` request header:

```
X-Tenant-ID: acme
```

If the runtime header is missing, the server returns `400 Bad Request` with:

```json
{"error": "missing X-Tenant-ID header"}
```

Tenant lifecycle and schema endpoints (`/api/v1/tenants/{tenantID}/...`) also use path-based tenant identification and do not require `X-Tenant-ID`.

## Error format

All error responses use the same JSON envelope:

```json
{"error": "human-readable message"}
```

Schema validation failures include an additional `details` array:

```json
{
  "error": "schema validation failed",
  "details": [
    "type 'document' has no relations defined",
    "permission 'edit' references unknown relation 'writer'"
  ]
}
```

## HTTP status codes

| Code | Meaning |
|------|---------|
| `200 OK` | Request succeeded; body contains the result |
| `201 Created` | Resource was created; body contains the new resource |
| `204 No Content` | Request succeeded; no body (e.g. DELETE) |
| `400 Bad Request` | Malformed request, missing required field, or missing tenant identifier |
| `404 Not Found` | Tenant, tuple, or schema does not exist |
| `405 Method Not Allowed` | Wrong HTTP method for the path |
| `409 Conflict` | Tuple already exists |
| `410 Gone` | Tenant has been deleted |
| `422 Unprocessable Entity` | Schema parse or compilation error |
| `429 Too Many Requests` | Tenant quota exceeded |
| `403 Forbidden` | Tenant is suspended |
| `500 Internal Server Error` | Unexpected server error |

## Health check

```bash
curl http://localhost:1997/healthz
```

```json
{"status": "ok"}
```

The health endpoint always returns `200 OK` while the server is running. It does not check backing-store connectivity.
