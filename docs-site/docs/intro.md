---
id: intro
slug: /
title: Introduction
sidebar_position: 1
---

# ZanGuard

ZanGuard is a Go authorization service that combines:

- relationship-based access control (ReBAC)
- attribute checks (ABAC conditions)
- tenant isolation

It runs as a single HTTP service with management and runtime APIs.

## Why This Matters

- **Prevents authorization drift:** policies live in schema + tuples, not scattered ad-hoc checks.
- **Reduces security risk:** every check goes through one engine with cycle/depth protections.
- **Improves auditability:** tuple mutations are recorded and queryable through changelog APIs.
- **Supports multi-tenant SaaS safely:** tenant-scoped data access is enforced at API and store layers.
- **Speeds integration:** standards-based runtime API (AuthZen) plus Management API and OpenAPI specs.

## What Works Today

- Tenant lifecycle management (`pending`, `active`, `suspended`, `deleted`)
- YAML schema upload and compile
- Tuple write/read/delete APIs
- Object and subject attribute APIs
- Runtime authorization checks via AuthZen endpoints
- Tuple changelog read API
- PostgreSQL-backed persistence for tenant and authorization data

## Current Behavior and Limits

- Runtime storage is PostgreSQL (`DATABASE_URL` required).
- Default server port is `1997`.
- Schemas are loaded into process memory per tenant.
- After server restart, schemas must be uploaded again before checks succeed.
- Changelog is append-only for tuple mutations (insert/delete).
- Check traversal uses cycle detection and a max depth limit (default `25`).

## Data Model

The core authorization fact is a relation tuple:

```text
<object_type>:<object_id>#<relation>@<subject_type>:<subject_id>
```

Example:

```text
document:readme#viewer@user:alice
```

ABAC conditions can also reference:

- `object` attributes
- `subject` attributes
- `request` context values

## API Surfaces

- Management API: `/api/v1/...`
- AuthZen Runtime API: `/access/v1/...`

OpenAPI specs:

- [Management OpenAPI](/openapi/management-v1.yaml)
- [Runtime OpenAPI](/openapi/runtime-authzen-v1.yaml)

## Start Here

- [Getting Started](./getting-started)
- [API Overview](./api/overview)
- [ABAC Manual API Test](./examples/abac-manual-api)
