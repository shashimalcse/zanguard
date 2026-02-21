---
id: abac-manual-api
title: ABAC Manual API Test
sidebar_position: 4
---

# Example: ABAC Manual API Test

This example is a full manual API flow for ABAC testing with `curl`.

It covers:

1. Creating and activating a tenant
2. Loading an ABAC schema
3. Inserting attributes and tuples
4. Running runtime check calls with expected outcomes

## Prerequisites

- ZanGuard server running on `http://localhost:1997`
- `curl`

## Variables

```bash
BASE=http://localhost:1997
TENANT=abac-demo
```

## 1) Create and Activate Tenant

```bash
curl -X POST $BASE/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"$TENANT\",
    \"display_name\": \"ABAC Demo\",
    \"schema_mode\": \"own\"
  }"

curl -X POST $BASE/api/v1/tenants/$TENANT/activate
```

## 2) Load ABAC Schema

```bash
cat > schema.yaml <<'YAML'
version: "1.0"
types:
  user:
    attributes:
      clearance_level: int
      department: string

  document:
    attributes:
      min_clearance: int
      department: string
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
        condition: 'subject.clearance_level >= object.min_clearance && subject.department == object.department && request.mfa_verified == true'
      delete:
        resolve: owner
        condition: "subject.clearance_level >= 4"
YAML

curl -X PUT $BASE/api/v1/tenants/$TENANT/schema \
  -H "Content-Type: application/yaml" \
  --data-binary @schema.yaml
```

## 3) Insert Attributes

```bash
# document attributes
curl -X PUT $BASE/api/v1/t/$TENANT/attributes/objects/document/q4-report \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"min_clearance":2,"department":"engineering"}}'

# subject attributes
curl -X PUT $BASE/api/v1/t/$TENANT/attributes/subjects/user/alice \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"clearance_level":3,"department":"engineering"}}'

curl -X PUT $BASE/api/v1/t/$TENANT/attributes/subjects/user/bob \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"clearance_level":5,"department":"engineering"}}'

curl -X PUT $BASE/api/v1/t/$TENANT/attributes/subjects/user/charlie \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"clearance_level":1,"department":"engineering"}}'
```

## 4) Insert Tuples

```bash
# alice + charlie are viewers, bob is owner
curl -X POST $BASE/api/v1/t/$TENANT/tuples \
  -H "Content-Type: application/json" \
  -d '{"object_type":"document","object_id":"q4-report","relation":"viewer","subject_type":"user","subject_id":"alice"}'

curl -X POST $BASE/api/v1/t/$TENANT/tuples \
  -H "Content-Type: application/json" \
  -d '{"object_type":"document","object_id":"q4-report","relation":"viewer","subject_type":"user","subject_id":"charlie"}'

curl -X POST $BASE/api/v1/t/$TENANT/tuples \
  -H "Content-Type: application/json" \
  -d '{"object_type":"document","object_id":"q4-report","relation":"owner","subject_type":"user","subject_id":"bob"}'
```

## 5) Check Calls (AuthZen Runtime)

### A) Alice can view (expected `true`)

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "subject":{"type":"user","id":"alice"},
    "resource":{"type":"document","id":"q4-report"},
    "action":{"name":"view"},
    "context":{"mfa_verified":true}
  }'
```

Expected:

```json
{"decision": true}
```

### B) Alice without MFA (expected `false`)

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "subject":{"type":"user","id":"alice"},
    "resource":{"type":"document","id":"q4-report"},
    "action":{"name":"view"},
    "context":{"mfa_verified":false}
  }'
```

Expected:

```json
{"decision": false}
```

### C) Charlie low clearance (expected `false`)

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "subject":{"type":"user","id":"charlie"},
    "resource":{"type":"document","id":"q4-report"},
    "action":{"name":"view"},
    "context":{"mfa_verified":true}
  }'
```

Expected:

```json
{"decision": false}
```

### D) Bob can delete (expected `true`)

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "subject":{"type":"user","id":"bob"},
    "resource":{"type":"document","id":"q4-report"},
    "action":{"name":"delete"}
  }'
```

Expected:

```json
{"decision": true}
```

### E) Alice cannot delete (expected `false`)

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "subject":{"type":"user","id":"alice"},
    "resource":{"type":"document","id":"q4-report"},
    "action":{"name":"delete"}
  }'
```

Expected:

```json
{"decision": false}
```
