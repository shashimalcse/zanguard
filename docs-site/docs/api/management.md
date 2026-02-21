---
id: management
title: Management API
sidebar_position: 2
---

# Management API

The Management API lives under `/api/v1/` and covers the full operational surface: tenant lifecycle, schema loading, relation-tuple writes, attribute management, the audit changelog, and subject-tree expansion.

All endpoints return `application/json`. Tenant-scoped management endpoints use the path prefix `/api/v1/t/{tenantID}/...` (tuples, attributes, changelog, expand). Tenant lifecycle and schema endpoints use `/api/v1/tenants/{tenantID}/...`.

OpenAPI: [Download `management-v1.yaml`](/openapi/management-v1.yaml)

---

## Tenants

### Endpoint summary

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/tenants` | Create a tenant |
| `GET` | `/api/v1/tenants` | List tenants |
| `GET` | `/api/v1/tenants/{tenantID}` | Get a tenant |
| `DELETE` | `/api/v1/tenants/{tenantID}` | Delete a tenant |
| `POST` | `/api/v1/tenants/{tenantID}/activate` | Activate a tenant |
| `POST` | `/api/v1/tenants/{tenantID}/suspend` | Suspend a tenant |

---

### Create tenant

**`POST /api/v1/tenants`**

Creates a new tenant. The tenant starts in the `pending` state. Call `/activate` before writing any data.

#### Request body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Unique identifier for the tenant |
| `display_name` | string | no | Human-readable name |
| `schema_mode` | string | no | `"own"` (default), `"shared"`, or `"inherited"` |
| `parent_tenant_id` | string | no | ID of a parent tenant (for inherited/shared schemas) |
| `shared_schema_ref` | string | no | Reference to the shared schema tenant |

```bash
curl -X POST http://localhost:1997/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "id": "acme",
    "display_name": "Acme Corp",
    "schema_mode": "own"
  }'
```

#### Response — `201 Created`

```json
{
  "id": "acme",
  "display_name": "Acme Corp",
  "status": "pending",
  "schema_mode": "own",
  "config": {
    "max_tuples": 0,
    "max_requests_per_sec": 0,
    "retention_days": 0,
    "sync_enabled": false
  },
  "created_at": "2026-02-19T10:00:00Z",
  "updated_at": "2026-02-19T10:00:00Z"
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | `id` is missing, or a tenant with that ID already exists |

---

### List tenants

**`GET /api/v1/tenants`**

Returns all tenants, with optional filtering and pagination.

#### Query parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status: `pending`, `active`, `suspended`, `deleted` |
| `parent_id` | string | Filter by parent tenant ID |
| `limit` | integer | Maximum number of results to return |
| `offset` | integer | Number of results to skip (for pagination) |

```bash
curl "http://localhost:1997/api/v1/tenants?status=active&limit=10"
```

#### Response — `200 OK`

```json
{
  "tenants": [
    {
      "id": "acme",
      "display_name": "Acme Corp",
      "status": "active",
      "schema_mode": "own",
      "config": {
        "max_tuples": 0,
        "max_requests_per_sec": 0,
        "retention_days": 0,
        "sync_enabled": false
      },
      "created_at": "2026-02-19T10:00:00Z",
      "updated_at": "2026-02-19T10:00:00Z"
    }
  ],
  "count": 1
}
```

---

### Get tenant

**`GET /api/v1/tenants/{tenantID}`**

Returns a single tenant by ID.

```bash
curl http://localhost:1997/api/v1/tenants/acme
```

#### Response — `200 OK`

```json
{
  "id": "acme",
  "display_name": "Acme Corp",
  "status": "active",
  "schema_mode": "own",
  "config": {
    "max_tuples": 0,
    "max_requests_per_sec": 0,
    "retention_days": 0,
    "sync_enabled": false
  },
  "created_at": "2026-02-19T10:00:00Z",
  "updated_at": "2026-02-19T10:00:00Z"
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `404` | Tenant not found |

---

### Delete tenant

**`DELETE /api/v1/tenants/{tenantID}`**

Soft-deletes a tenant by transitioning it to `deleted`. Returns `204 No Content` on success.

```bash
curl -X DELETE http://localhost:1997/api/v1/tenants/acme
```

#### Response — `204 No Content`

No response body.

#### Error responses

| Status | Condition |
|--------|-----------|
| `404` | Tenant not found |

---

### Activate tenant

**`POST /api/v1/tenants/{tenantID}/activate`**

Transitions the tenant from `pending` or `suspended` to `active`. A tenant must be active before tuples, attributes, or permission checks can be performed against it.

```bash
curl -X POST http://localhost:1997/api/v1/tenants/acme/activate
```

#### Response — `200 OK`

Returns the updated tenant object.

```json
{
  "id": "acme",
  "display_name": "Acme Corp",
  "status": "active",
  "schema_mode": "own",
  "config": {
    "max_tuples": 0,
    "max_requests_per_sec": 0,
    "retention_days": 0,
    "sync_enabled": false
  },
  "created_at": "2026-02-19T10:00:00Z",
  "updated_at": "2026-02-19T10:05:00Z"
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `404` | Tenant not found |
| `410` | Tenant has been deleted |

---

### Suspend tenant

**`POST /api/v1/tenants/{tenantID}/suspend`**

Transitions the tenant to `suspended`. Suspended tenants allow reads but reject writes.

```bash
curl -X POST http://localhost:1997/api/v1/tenants/acme/suspend
```

#### Response — `200 OK`

Returns the updated tenant object.

```json
{
  "id": "acme",
  "display_name": "Acme Corp",
  "status": "suspended",
  "schema_mode": "own",
  "config": {
    "max_tuples": 0,
    "max_requests_per_sec": 0,
    "retention_days": 0,
    "sync_enabled": false
  },
  "created_at": "2026-02-19T10:00:00Z",
  "updated_at": "2026-02-19T10:10:00Z"
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `404` | Tenant not found |
| `410` | Tenant has been deleted |

---

## Schema

A schema defines the object types, relations, and permissions that ZanGuard enforces. It is written in the ZanGuard YAML DSL and loaded per tenant. The engine compiles and validates the schema before accepting it.

### Endpoint summary

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/api/v1/tenants/{tenantID}/schema` | Load (or replace) a tenant's schema |
| `GET` | `/api/v1/tenants/{tenantID}/schema` | Get the currently loaded schema |

---

### Load schema

**`PUT /api/v1/tenants/{tenantID}/schema`**

Uploads a raw YAML schema for the tenant. The schema is parsed, compiled, and validated before it is accepted. Loading a new schema replaces any previously loaded schema for that tenant.

The request body is the raw YAML text — not JSON-encoded.

```bash
curl -X PUT http://localhost:1997/api/v1/tenants/acme/schema \
  -H "Content-Type: application/yaml" \
  --data-binary @schema.yaml
```

Inline alternative:

```bash
curl -X PUT http://localhost:1997/api/v1/tenants/acme/schema \
  -H "Content-Type: application/yaml" \
  --data-binary @- <<'YAML'
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

#### Response — `200 OK`

```json
{
  "tenant_id": "acme",
  "hash": "sha256:a3f1...",
  "version": "1.0",
  "source": "version: \"1.0\"\ntypes:\n  user: {}\n  document:\n    ...",
  "compiled_at": "2026-02-19T10:15:00Z"
}
```

#### Response fields

| Field | Description |
|-------|-------------|
| `tenant_id` | The tenant this schema is loaded for |
| `hash` | SHA-256 content hash of the compiled schema |
| `version` | The `version` field from the YAML |
| `source` | The raw YAML text as uploaded |
| `compiled_at` | UTC timestamp when compilation completed |

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Could not read the request body |
| `404` | Tenant not found |
| `422` | YAML parse error, compilation error, or schema validation failure |

On validation failure the response body includes a `details` array:

```json
{
  "error": "schema validation failed",
  "details": [
    "type 'document' permission 'edit' references unknown relation 'writer'"
  ]
}
```

---

### Get schema

**`GET /api/v1/tenants/{tenantID}/schema`**

Returns the currently loaded schema for the tenant.

```bash
curl http://localhost:1997/api/v1/tenants/acme/schema
```

#### Response — `200 OK`

```json
{
  "tenant_id": "acme",
  "hash": "sha256:a3f1...",
  "version": "1.0",
  "source": "version: \"1.0\"\ntypes:\n  user: {}\n  document:\n    ...",
  "compiled_at": "2026-02-19T10:15:00Z"
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `404` | Tenant not found, or no schema has been loaded for the tenant |

---

## Tuples

Relation tuples are the fundamental authorization facts. Each tuple asserts that a subject holds a relation on an object — for example, `user:alice` is a `viewer` of `document:readme`.

All tuple endpoints are tenant-scoped via `/api/v1/t/{tenantID}/...`.

### Endpoint summary

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/t/{tenantID}/tuples` | Write a single tuple |
| `POST` | `/api/v1/t/{tenantID}/tuples/batch` | Write multiple tuples atomically |
| `DELETE` | `/api/v1/t/{tenantID}/tuples` | Delete a single tuple |
| `GET` | `/api/v1/t/{tenantID}/tuples` | Read tuples with optional filters |

---

### Write tuple

**`POST /api/v1/t/{tenantID}/tuples`**

Writes a single relation tuple for the tenant.

#### Request body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `object_type` | string | yes | Type of the object (e.g. `"document"`) |
| `object_id` | string | yes | ID of the object (e.g. `"readme"`) |
| `relation` | string | yes | Relation name (e.g. `"viewer"`) |
| `subject_type` | string | yes | Type of the subject (e.g. `"user"`) |
| `subject_id` | string | yes | ID of the subject (e.g. `"alice"`) |
| `subject_relation` | string | no | Subject relation for userset references (e.g. `"member"`) |
| `ttl_seconds` | integer | no | Relative expiry in seconds (must be > 0, max 86400). Mutually exclusive with `expires_at` |
| `expires_at` | string | no | Absolute expiry timestamp (RFC3339). Mutually exclusive with `ttl_seconds` |
| `attributes` | object | no | Arbitrary key-value metadata attached to the tuple |

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

Time-bound consent grant example (expires after 5 minutes):

```bash
curl -X POST http://localhost:1997/api/v1/t/acme/tuples \
  -H "Content-Type: application/json" \
  -d '{
    "object_type": "mailbox",
    "object_id": "alice",
    "relation": "reader",
    "subject_type": "agent",
    "subject_id": "support-bot",
    "ttl_seconds": 300
  }'
```

#### Response — `201 Created`

```json
{"status": "ok"}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Malformed request body |
| `403` | Tenant is suspended |
| `404` | Tenant not found |
| `409` | Tuple already exists |
| `429` | Tenant tuple quota exceeded |

---

### Batch write tuples

**`POST /api/v1/t/{tenantID}/tuples/batch`**

Writes multiple tuples in a single request. All tuples are written atomically — either all succeed or none are persisted.

#### Request body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tuples` | array | yes | Array of tuple objects (same fields as single write) |

```bash
curl -X POST http://localhost:1997/api/v1/t/acme/tuples/batch \
  -H "Content-Type: application/json" \
  -d '{
    "tuples": [
      {
        "object_type": "document",
        "object_id": "readme",
        "relation": "viewer",
        "subject_type": "user",
        "subject_id": "alice"
      },
      {
        "object_type": "document",
        "object_id": "readme",
        "relation": "owner",
        "subject_type": "user",
        "subject_id": "bob"
      }
    ]
  }'
```

#### Response — `201 Created`

```json
{"status": "ok", "count": 2}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Malformed request body |
| `403` | Tenant is suspended |
| `404` | Tenant not found |
| `409` | One or more tuples already exist |
| `429` | Tenant tuple quota exceeded |

---

### Delete tuple

**`DELETE /api/v1/t/{tenantID}/tuples`**

Deletes a single relation tuple. The tuple to delete is identified by the request body.

```bash
curl -X DELETE http://localhost:1997/api/v1/t/acme/tuples \
  -H "Content-Type: application/json" \
  -d '{
    "object_type": "document",
    "object_id": "readme",
    "relation": "viewer",
    "subject_type": "user",
    "subject_id": "alice"
  }'
```

#### Response — `204 No Content`

No response body.

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Malformed request body |
| `403` | Tenant is suspended |
| `404` | Tenant not found, or tuple does not exist |

---

### Read tuples

**`GET /api/v1/t/{tenantID}/tuples`**

Reads tuples for the tenant. All query parameters are optional — omitting all parameters returns all tuples for the tenant.

#### Query parameters

| Parameter | Description |
|-----------|-------------|
| `object_type` | Filter by object type |
| `object_id` | Filter by object ID |
| `relation` | Filter by relation name |
| `subject_type` | Filter by subject type |
| `subject_id` | Filter by subject ID |
| `subject_relation` | Filter by subject relation (userset references) |
| `include_expired` | Include expired tuples (`true` or `false`). Default is `false` |

```bash
curl "http://localhost:1997/api/v1/t/acme/tuples?object_type=document&object_id=readme"
```

To inspect expired grants for debugging or audit:

```bash
curl "http://localhost:1997/api/v1/t/acme/tuples?include_expired=true"
```

#### Response — `200 OK`

```json
{
  "tuples": [
    {
      "tenant_id": "acme",
      "object_type": "document",
      "object_id": "readme",
      "relation": "viewer",
      "subject_type": "user",
      "subject_id": "alice",
      "created_at": "2026-02-19T10:20:00Z",
      "updated_at": "2026-02-19T10:20:00Z"
    },
    {
      "tenant_id": "acme",
      "object_type": "document",
      "object_id": "readme",
      "relation": "owner",
      "subject_type": "user",
      "subject_id": "bob",
      "created_at": "2026-02-19T10:20:00Z",
      "updated_at": "2026-02-19T10:20:00Z"
    }
  ],
  "count": 2
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Invalid request |
| `404` | Tenant not found |

---

## Attributes

Attributes store arbitrary key-value metadata on objects and subjects. They are used by ABAC (attribute-based access control) conditions in permission rules.

All attribute endpoints are tenant-scoped via `/api/v1/t/{tenantID}/...`.

### Endpoint summary

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/api/v1/t/{tenantID}/attributes/objects/{type}/{id}` | Set attributes on an object |
| `GET` | `/api/v1/t/{tenantID}/attributes/objects/{type}/{id}` | Get attributes of an object |
| `PUT` | `/api/v1/t/{tenantID}/attributes/subjects/{type}/{id}` | Set attributes on a subject |
| `GET` | `/api/v1/t/{tenantID}/attributes/subjects/{type}/{id}` | Get attributes of a subject |

---

### Set object attributes

**`PUT /api/v1/t/{tenantID}/attributes/objects/{type}/{id}`**

Replaces the attributes for the object identified by `{type}` and `{id}`. The provided map fully replaces any previously stored attributes.

#### Path parameters

| Parameter | Description |
|-----------|-------------|
| `type` | Object type (e.g. `document`) |
| `id` | Object ID (e.g. `readme`) |

#### Request body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `attributes` | object | yes | Key-value map of attribute names to values |

```bash
curl -X PUT \
  "http://localhost:1997/api/v1/t/acme/attributes/objects/document/readme" \
  -H "Content-Type: application/json" \
  -d '{
    "attributes": {
      "sensitivity": "confidential",
      "department": "engineering"
    }
  }'
```

#### Response — `200 OK`

```json
{
  "attributes": {
    "sensitivity": "confidential",
    "department": "engineering"
  }
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Malformed request body |
| `403` | Tenant is suspended |
| `404` | Tenant not found |

---

### Get object attributes

**`GET /api/v1/t/{tenantID}/attributes/objects/{type}/{id}`**

Returns the attributes stored for the specified object.

```bash
curl "http://localhost:1997/api/v1/t/acme/attributes/objects/document/readme"
```

#### Response — `200 OK`

```json
{
  "attributes": {
    "sensitivity": "confidential",
    "department": "engineering"
  }
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Invalid request |
| `404` | Tenant not found, or no attributes stored for this object |

---

### Set subject attributes

**`PUT /api/v1/t/{tenantID}/attributes/subjects/{type}/{id}`**

Replaces the attributes for the subject identified by `{type}` and `{id}`.

#### Path parameters

| Parameter | Description |
|-----------|-------------|
| `type` | Subject type (e.g. `user`) |
| `id` | Subject ID (e.g. `alice`) |

```bash
curl -X PUT \
  "http://localhost:1997/api/v1/t/acme/attributes/subjects/user/alice" \
  -H "Content-Type: application/json" \
  -d '{
    "attributes": {
      "clearance_level": 3,
      "region": "us-west"
    }
  }'
```

#### Response — `200 OK`

```json
{
  "attributes": {
    "clearance_level": 3,
    "region": "us-west"
  }
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Malformed request body |
| `403` | Tenant is suspended |
| `404` | Tenant not found |

---

### Get subject attributes

**`GET /api/v1/t/{tenantID}/attributes/subjects/{type}/{id}`**

Returns the attributes stored for the specified subject.

```bash
curl "http://localhost:1997/api/v1/t/acme/attributes/subjects/user/alice"
```

#### Response — `200 OK`

```json
{
  "attributes": {
    "clearance_level": 3,
    "region": "us-west"
  }
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Invalid request |
| `404` | Tenant not found, or no attributes stored for this subject |

---

## Changelog

The changelog is an append-only, monotonically sequenced log of every tuple mutation within a tenant. It is suitable for event streaming, audit trails, and cache invalidation.

The changelog endpoint is tenant-scoped via `/api/v1/t/{tenantID}/changelog`.

### Read changelog

**`GET /api/v1/t/{tenantID}/changelog`**

Returns changelog entries for the tenant starting after the given sequence number.

#### Query parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `since_seq` | uint64 | `0` | Return entries with sequence number greater than this value. Use `0` to start from the beginning. |
| `limit` | integer | `100` | Maximum number of entries to return |

```bash
curl "http://localhost:1997/api/v1/t/acme/changelog?since_seq=0&limit=50"
```

#### Response — `200 OK`

```json
{
  "entries": [
    {
      "seq": 1,
      "tenant_id": "acme",
      "op": "INSERT",
      "tuple": {
        "tenant_id": "acme",
        "object_type": "document",
        "object_id": "readme",
        "relation": "viewer",
        "subject_type": "user",
        "subject_id": "alice",
        "created_at": "2026-02-19T10:20:00Z",
        "updated_at": "2026-02-19T10:20:00Z"
      },
      "ts": "2026-02-19T10:20:00Z",
      "actor": "",
      "source": "api"
    }
  ],
  "count": 1,
  "latest_sequence": 1
}
```

#### Response fields

| Field | Description |
|-------|-------------|
| `entries` | Array of changelog entries |
| `entries[].seq` | Monotonically increasing sequence number |
| `entries[].op` | Operation type: `INSERT`, `DELETE`, or `UPDATE` |
| `entries[].tuple` | The tuple that was affected |
| `entries[].ts` | UTC timestamp of the change |
| `entries[].actor` | Identity that made the change (may be empty) |
| `entries[].source` | Source of the change: `api`, `import`, or `sync` |
| `count` | Number of entries returned |
| `latest_sequence` | Highest sequence number currently in the changelog |

Use `latest_sequence` from one response as the `since_seq` value in the next poll to implement efficient streaming.

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Invalid request |
| `404` | Tenant not found |

---

## Expand

The expand endpoint returns the direct subject tree for a given relation on an object. This is useful for debugging permission models and for building UI components that display "who has access."

The expand endpoint is tenant-scoped via `/api/v1/t/{tenantID}/expand`.

### Expand relation

**`POST /api/v1/t/{tenantID}/expand`**

Expands direct subjects for `object_type:object_id#relation`.

#### Request body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `object_type` | string | yes | Object type to expand |
| `object_id` | string | yes | Object ID to expand |
| `relation` | string | yes | Relation to expand |

```bash
curl -X POST http://localhost:1997/api/v1/t/acme/expand \
  -H "Content-Type: application/json" \
  -d '{
    "object_type": "document",
    "object_id": "readme",
    "relation": "viewer"
  }'
```

#### Response — `200 OK`

The response contains direct relation subjects. Userset nodes are returned as direct children and are not recursively expanded by this endpoint.

```json
{
  "subject": {
    "type": "document",
    "id": "readme",
    "relation": "viewer"
  },
  "children": [
    {
      "subject": {
        "type": "user",
        "id": "alice"
      },
      "children": null
    },
    {
      "subject": {
        "type": "group",
        "id": "eng",
        "relation": "member"
      },
      "children": null
    }
  ]
}
```

#### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Malformed request body |
| `404` | Tenant not found |
| `500` | Engine error during expansion |
