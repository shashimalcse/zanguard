---
id: intro
slug: /
title: Introduction
sidebar_position: 1
---

# ZanGuard

**ZanGuard** is a Zanzibar-inspired authorization engine written in Go. It provides fine-grained, relationship-based access control (ReBAC) with attribute-based access control (ABAC) extensions — designed for multi-tenant SaaS and enterprise environments.

## What is it?

Authorization is the question: *"Can this user do this action on this resource?"*

Most systems answer this with flat role checks (`user.role == "admin"`). ZanGuard answers it by traversing a **graph of relationships** — enabling expressive, auditable, and infinitely composable access policies.

```
Can user:thilina view document:readme?
  → Is thilina a viewer of readme?           ✓ direct tuple found
  → Is thilina a member of a viewer group?   (checked)
  → Does thilina own a parent folder?        (checked)
  → Does an ABAC condition pass?             (checked)
```

## Core Model

All authorization facts are stored as **relation tuples**:

```
<object_type>:<object_id>#<relation>@<subject_type>:<subject_id>
```

| Part | Example | Meaning |
|------|---------|---------|
| `object_type:object_id` | `document:readme` | The resource being protected |
| `#relation` | `#viewer` | The relationship |
| `subject_type:subject_id` | `user:thilina` | Who holds the relationship |

A permission check traverses these tuples — following userset expansions, arrow inheritance, and ABAC conditions — to return allow or deny.

## Key Features

| Feature | Description |
|---------|-------------|
| **ReBAC Engine** | Zanzibar-style graph traversal with union, intersect, and exclusion |
| **Userset Expansion** | Group membership chains (`group:eng#member`) |
| **Arrow Traversal** | Permission inheritance across hierarchies (`parent->view`) |
| **ABAC Conditions** | Safe expression evaluation on object/subject attributes |
| **Cycle Detection** | Circular relations terminate cleanly without hanging |
| **Multi-Tenancy** | Complete data isolation per tenant with three schema modes |
| **Schema DSL** | Declarative YAML schema for types, relations, and permissions |
| **Audit Changelog** | Append-only, sequenced log of all data mutations |
| **Dual Backends** | In-memory (testing/edge) and PostgreSQL (production) |

## Design Goals

- **Correctness first** — no authorization bugs, ever
- **Predictable performance** — depth-limited traversal, cycle detection
- **Tenant isolation** — hard boundary between tenant data at every layer
- **Declarative schema** — policies are code, not magic strings
- **Auditable** — every change is logged with a monotonic sequence number

## Architecture

```
cmd/server/         ← Demo / entry point
pkg/
  engine/           ← Permission check algorithm
  schema/           ← YAML DSL parser & compiler
  storage/          ← PostgreSQL + in-memory backends
  tenant/           ← Multi-tenancy lifecycle & context
  model/            ← Core types (RelationTuple, Tenant, …)
configs/examples/   ← Example schemas
migrations/         ← PostgreSQL migrations
```

## Tech Stack

- **Go 1.23**
- **PostgreSQL** via `pgx/v5`
- **YAML** via `gopkg.in/yaml.v3`
- **expr-lang** for safe ABAC expression evaluation

## Next Steps

- [Getting Started](./getting-started) — run your first permission check in minutes
- [Core Concepts](./core-concepts/relation-tuples) — understand the data model
- [Schema DSL](./schema/overview) — define your authorization model
