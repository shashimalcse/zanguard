---
id: authzen
title: AuthZen Runtime API
sidebar_position: 3
---

# AuthZen Runtime API

ZanGuard implements the [AuthZen 1.0](https://openid.net/wg/authzen/) specification for runtime permission evaluation. The AuthZen API provides a standardised interface that lets any AuthZen-compatible Policy Enforcement Point (PEP) query ZanGuard without coupling to its internal data model.

The AuthZen API lives under `/access/v1/` and is separate from the Management API.

---

## Key concepts

### Subject, resource, and action

AuthZen models an access decision as: *"Can this **subject** perform this **action** on this **resource**?"*

ZanGuard maps these AuthZen concepts to its internal model as follows:

| AuthZen field | ZanGuard field |
|---------------|----------------|
| `subject.type` + `subject.id` | `subject_type` + `subject_id` |
| `resource.type` + `resource.id` | `object_type` + `object_id` |
| `action.name` | `permission` |

### Tenant identification

The target tenant is identified by the `X-Tenant-ID` request header. This header is required on every AuthZen request.

### Error behaviour

Per the AuthZen 1.0 specification, evaluation errors **do not** produce HTTP error responses. Instead, the engine returns `{"decision": false}` with `200 OK`. This ensures that PEPs receive a safe-deny rather than a transport failure on engine errors.

The only exception is a missing or invalid `X-Tenant-ID` header, which returns `400 Bad Request` (a request configuration error, not an evaluation error).

---

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/access/v1/evaluation` | Evaluate a single access decision |
| `POST` | `/access/v1/evaluations` | Evaluate multiple access decisions in one request |

---

## Single evaluation

**`POST /access/v1/evaluation`**

Evaluates whether a subject is allowed to perform an action on a resource.

### Request body

```json
{
  "subject": {
    "type": "user",
    "id": "alice",
    "properties": {}
  },
  "resource": {
    "type": "document",
    "id": "readme",
    "properties": {}
  },
  "action": {
    "name": "view",
    "properties": {}
  },
  "context": {}
}
```

#### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subject` | object | yes | The entity requesting access |
| `subject.type` | string | yes | Subject type, mapped to ZanGuard `subject_type` |
| `subject.id` | string | yes | Subject identifier, mapped to ZanGuard `subject_id` |
| `subject.properties` | object | no | Additional subject properties (passed to the engine context) |
| `resource` | object | yes | The resource being accessed |
| `resource.type` | string | yes | Resource type, mapped to ZanGuard `object_type` |
| `resource.id` | string | yes | Resource identifier, mapped to ZanGuard `object_id` |
| `resource.properties` | object | no | Additional resource properties |
| `action` | object | yes | The action being performed |
| `action.name` | string | yes | Action name, mapped to ZanGuard `permission` |
| `action.properties` | object | no | Additional action properties |
| `context` | object | no | Arbitrary context passed to ABAC condition evaluation |

### Response — `200 OK`

```json
{
  "decision": true
}
```

| Field | Type | Description |
|-------|------|-------------|
| `decision` | boolean | `true` if access is granted, `false` if denied or if an evaluation error occurred |
| `context` | object | Optional context returned by the engine (omitted when empty) |

### Example — access granted

```bash
curl -X POST http://localhost:8080/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "subject": {
      "type": "user",
      "id": "alice"
    },
    "resource": {
      "type": "document",
      "id": "readme"
    },
    "action": {
      "name": "view"
    }
  }'
```

```json
{"decision": true}
```

### Example — access denied

```bash
curl -X POST http://localhost:8080/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "subject": {
      "type": "user",
      "id": "charlie"
    },
    "resource": {
      "type": "document",
      "id": "readme"
    },
    "action": {
      "name": "delete"
    }
  }'
```

```json
{"decision": false}
```

### Example — with ABAC context

Pass additional context values for ABAC condition evaluation using the `context` field:

```bash
curl -X POST http://localhost:8080/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "subject": {
      "type": "user",
      "id": "alice"
    },
    "resource": {
      "type": "document",
      "id": "classified-report"
    },
    "action": {
      "name": "view"
    },
    "context": {
      "ip_address": "10.0.0.1",
      "time_of_day": "business_hours"
    }
  }'
```

### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Missing `X-Tenant-ID` header, or malformed JSON body |

All other errors (unknown tenant, engine failure, missing schema) yield `200 OK` with `{"decision": false}`.

---

## Batch evaluation

**`POST /access/v1/evaluations`**

Evaluates multiple access decisions in a single request. All evaluations share the same top-level subject. Each item in the `evaluations` array specifies a distinct resource and action pair, and may include its own `context`.

### Request body

```json
{
  "subject": {
    "type": "user",
    "id": "alice",
    "properties": {}
  },
  "evaluations": [
    {
      "resource": {
        "type": "document",
        "id": "readme"
      },
      "action": {
        "name": "view"
      },
      "context": {}
    },
    {
      "resource": {
        "type": "document",
        "id": "readme"
      },
      "action": {
        "name": "delete"
      }
    }
  ]
}
```

#### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subject` | object | yes | The entity making all requests in this batch |
| `subject.type` | string | yes | Subject type |
| `subject.id` | string | yes | Subject identifier |
| `subject.properties` | object | no | Additional subject properties |
| `evaluations` | array | yes | One or more evaluation items |
| `evaluations[].resource` | object | yes | Resource for this item |
| `evaluations[].resource.type` | string | yes | Resource type |
| `evaluations[].resource.id` | string | yes | Resource identifier |
| `evaluations[].resource.properties` | object | no | Additional resource properties |
| `evaluations[].action` | object | yes | Action for this item |
| `evaluations[].action.name` | string | yes | Action name |
| `evaluations[].action.properties` | object | no | Additional action properties |
| `evaluations[].context` | object | no | Per-item context for ABAC evaluation |

### Response — `200 OK`

The response contains one result per evaluation item, in the same order as the request.

```json
{
  "evaluations": [
    {"decision": true},
    {"decision": false}
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `evaluations` | array | Results in the same order as the request items |
| `evaluations[].decision` | boolean | `true` if access is granted; `false` if denied or if this item's evaluation errored |
| `evaluations[].context` | object | Optional per-item context from the engine (omitted when empty) |

Per-item errors do not affect other items in the batch. A failing evaluation for one item yields `decision: false` for that item while other items continue to be evaluated normally.

### Example

```bash
curl -X POST http://localhost:8080/access/v1/evaluations \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: acme" \
  -d '{
    "subject": {
      "type": "user",
      "id": "alice"
    },
    "evaluations": [
      {
        "resource": {"type": "document", "id": "readme"},
        "action":   {"name": "view"}
      },
      {
        "resource": {"type": "document", "id": "readme"},
        "action":   {"name": "delete"}
      },
      {
        "resource": {"type": "document", "id": "budget"},
        "action":   {"name": "view"}
      }
    ]
  }'
```

```json
{
  "evaluations": [
    {"decision": true},
    {"decision": false},
    {"decision": true}
  ]
}
```

### Error responses

| Status | Condition |
|--------|-----------|
| `400` | Missing `X-Tenant-ID` header, or malformed JSON body |

Individual item errors produce `decision: false` for that item — they do not cause an HTTP error response.

---

## AuthZen compliance notes

- ZanGuard implements the AuthZen 1.0 draft specification.
- The `subject.properties`, `resource.properties`, and `action.properties` fields are accepted but are not forwarded to the engine at this time. Use the top-level `context` field to pass values for ABAC condition evaluation.
- The response `context` field is returned only when the engine produces context output; it is omitted otherwise.
- Batch requests with an empty `evaluations` array return an empty `evaluations` array with `200 OK`.
