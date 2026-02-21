---
id: schema-modes
title: Schema Modes
sidebar_position: 3
---

# Schema Modes

A tenant's `schema_mode` decides how the engine resolves schema for checks.

## Modes

| Mode | Current behavior |
|------|------------------|
| `own` | Uses schema loaded for that tenant ID |
| `shared` | Uses `SharedSchemaRef` and engine shared-schema registry |
| `inherited` | Currently treated like `own` (no parent merge) |

## `own`

Fully supported through management API schema upload:

- `PUT /api/v1/tenants/{tenantID}/schema`

This compiles schema and registers it for that tenant in engine memory.

## `shared`

Engine behavior:

- tenant has `schema_mode=shared`
- engine resolves schema from `sharedSchemas[tenant.SharedSchemaRef]`

Current limitation:

- Management API does not currently provide an endpoint to register/update shared schemas in `sharedSchemas`
- `PUT /api/v1/tenants/{tenantID}/schema` loads tenant-local schema (`LoadSchema`), not shared ref schema (`LoadSharedSchema`)

So shared mode currently requires programmatic engine setup.

## `inherited`

Current implementation resolves inherited mode exactly like own mode.

No parent schema merge/override mechanism is active yet.

## Practical Guidance (Current)

- Use `own` for API-driven flows.
- Use `shared` only if you control service startup code and call `LoadSharedSchema`.
- Treat `inherited` as own mode for now.

## Engine Resolution Logic

Current engine logic:

- `own` -> `schemas[tenantID]`
- `shared` -> `sharedSchemas[SharedSchemaRef]`
- `inherited` -> `schemas[tenantID]`

If no schema is found, checks fail.

## See Also

- [Tenant Lifecycle](./lifecycle)
- [Tenant Context](./context)
- [Schema Overview](../schema/overview)
