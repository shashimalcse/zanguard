---
id: ai-agent-authorization
title: AI Agent Authorization
sidebar_position: 5
---

# Example: AI Agent Authorization

This example shows how to authorize AI agents with both ReBAC and ABAC:

- ReBAC: agent must hold a relation on the target object
- ABAC: agent attributes + object attributes + request context must pass

## Use Case

- `agent:refund-bot` can execute `tool:refund` only when:
  - clearance is high enough
  - team and model are allowed
  - request is in correct environment
  - human approval is present when required
- Agents can read tickets only if their clearance meets ticket sensitivity.

## Variables

```bash
BASE=http://localhost:1997
TENANT=ai-authz
```

## 1) Create and Activate Tenant

```bash
curl -X POST $BASE/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"$TENANT\",\"display_name\":\"AI AuthZ Demo\",\"schema_mode\":\"own\"}"

curl -X POST $BASE/api/v1/tenants/$TENANT/activate
```

## 2) Load Schema

```bash
cat > ai-agent-schema.yaml <<'YAML'
version: "1.0"
types:
  agent:
    attributes:
      clearance_level: int
      team: string
      model: string

  tool:
    attributes:
      required_clearance: int
      requires_human_approval: bool
      environment: string
    relations:
      executor:
        types: [agent]
    permissions:
      execute:
        resolve: executor
        condition: >-
          subject.clearance_level >= object.required_clearance &&
          subject.team == "billing-ai" &&
          subject.model == "gpt-4.1" &&
          (object.requires_human_approval == false || request.human_approved == true) &&
          request.environment == object.environment &&
          (request.purpose == "customer_refund" || request.purpose == "charge_correction")

  ticket:
    attributes:
      sensitivity: int
    relations:
      viewer:
        types: [agent]
    permissions:
      read:
        resolve: viewer
        condition: "subject.clearance_level >= object.sensitivity"
YAML

curl -X PUT $BASE/api/v1/tenants/$TENANT/schema \
  -H "Content-Type: application/yaml" \
  --data-binary @ai-agent-schema.yaml
```

## 3) Insert Attributes and Tuples

```bash
# Agent attributes
curl -X PUT $BASE/api/v1/t/$TENANT/attributes/subjects/agent/refund-bot \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"clearance_level":5,"team":"billing-ai","model":"gpt-4.1"}}'

curl -X PUT $BASE/api/v1/t/$TENANT/attributes/subjects/agent/triage-bot \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"clearance_level":2,"team":"support-ai","model":"gpt-4.1"}}'

# Tool and ticket attributes
curl -X PUT $BASE/api/v1/t/$TENANT/attributes/objects/tool/refund \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"required_clearance":4,"requires_human_approval":true,"environment":"prod"}}'

curl -X PUT $BASE/api/v1/t/$TENANT/attributes/objects/ticket/t-100 \
  -H "Content-Type: application/json" \
  -d '{"attributes":{"sensitivity":3}}'

# Relations
curl -X POST $BASE/api/v1/t/$TENANT/tuples \
  -H "Content-Type: application/json" \
  -d '{"object_type":"tool","object_id":"refund","relation":"executor","subject_type":"agent","subject_id":"refund-bot"}'

curl -X POST $BASE/api/v1/t/$TENANT/tuples \
  -H "Content-Type: application/json" \
  -d '{"object_type":"tool","object_id":"refund","relation":"executor","subject_type":"agent","subject_id":"triage-bot"}'

curl -X POST $BASE/api/v1/t/$TENANT/tuples \
  -H "Content-Type: application/json" \
  -d '{"object_type":"ticket","object_id":"t-100","relation":"viewer","subject_type":"agent","subject_id":"refund-bot"}'

curl -X POST $BASE/api/v1/t/$TENANT/tuples \
  -H "Content-Type: application/json" \
  -d '{"object_type":"ticket","object_id":"t-100","relation":"viewer","subject_type":"agent","subject_id":"triage-bot"}'
```

## 4) Runtime Check Calls

### A) `refund-bot` executes refund with approval (expected `true`)

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "subject":{"type":"agent","id":"refund-bot"},
    "resource":{"type":"tool","id":"refund"},
    "action":{"name":"execute"},
    "context":{"human_approved":true,"environment":"prod","purpose":"customer_refund"}
  }'
```

Expected:

```json
{"decision": true}
```

### B) `refund-bot` without human approval (expected `false`)

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "subject":{"type":"agent","id":"refund-bot"},
    "resource":{"type":"tool","id":"refund"},
    "action":{"name":"execute"},
    "context":{"human_approved":false,"environment":"prod","purpose":"customer_refund"}
  }'
```

Expected:

```json
{"decision": false}
```

### C) `triage-bot` execute refund (expected `false`)

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "subject":{"type":"agent","id":"triage-bot"},
    "resource":{"type":"tool","id":"refund"},
    "action":{"name":"execute"},
    "context":{"human_approved":true,"environment":"prod","purpose":"customer_refund"}
  }'
```

Expected:

```json
{"decision": false}
```

### D) Ticket read checks

`refund-bot` (expected `true`):

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"subject":{"type":"agent","id":"refund-bot"},"resource":{"type":"ticket","id":"t-100"},"action":{"name":"read"}}'
```

`triage-bot` (expected `false`):

```bash
curl -X POST $BASE/access/v1/evaluation \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"subject":{"type":"agent","id":"triage-bot"},"resource":{"type":"ticket","id":"t-100"},"action":{"name":"read"}}'
```
