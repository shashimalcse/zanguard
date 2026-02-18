# ZanGuard — Zanzibar-Inspired Authorization Engine

## Architecture & Implementation Plan

---

## 1. Vision & Design Principles

ZanGuard is a lightweight, Zanzibar-inspired authorization engine built in Go, designed for **native multi-tenancy**, **easy data migration**, **seamless data sync**, and **AuthZen 1.0 compliance**. It provides Relationship-Based Access Control (ReBAC) as its core model with attribute overlays for ABAC-like expressiveness.

### Core Principles

- **Tenant-native architecture** — Multi-tenancy is not an afterthought. Every tuple, schema, cache entry, changelog event, and API call is tenant-scoped. Tenants get full data isolation with the option of shared or per-tenant schemas.
- **Migration-first design** — Every schema and storage decision optimizes for bulk import/export and incremental sync from existing systems (LDAP, RBAC databases, Casbin policies, OPA bundles, SpiceDB schemas). Migrations are tenant-aware — import into specific tenants or migrate entire tenant hierarchies.
- **Simple policy language** — A human-readable YAML/DSL schema for defining object types, relations, and permissions — no Rego, no CEL unless needed for ABAC conditions.
- **Local caching only** — Per-instance in-memory caching with configurable TTLs, tenant-partitioned. No global distributed cache layer (Redis/Memcached), keeping the deployment footprint minimal.
- **Standards-compliant** — Full AuthZen 1.0 Evaluation API support out of the box.
- **Embeddable or standalone** — Deployable as a sidecar, a standalone service, or an embedded Go library.

---

## 2. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         API Gateway / gRPC                          │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────────────┐ │
│  │  AuthZen 1.0 │  │  Admin API   │  │  Sync & Migration API     │ │
│  │  /access/v1/ │  │  /admin/v1/  │  │  /sync/v1/                │ │
│  │  evaluate     │  │  schema      │  │  import / export / watch  │ │
│  │  evaluations  │  │  relations   │  │  changelog / snapshots    │ │
│  └──────┬───────┘  └──────┬───────┘  └────────────┬──────────────┘ │
│         │                 │                        │                │
│  ┌──────▼─────────────────▼────────────────────────▼──────────────┐ │
│  │              Tenant Resolution Middleware                       │ │
│  │  (Header / Path / JWT claim / API key → tenant_id)             │ │
│  └──────┬─────────────────┬────────────────────────┬──────────────┘ │
│         │                 │                        │                │
│  ┌──────▼─────────────────▼────────────────────────▼──────────────┐ │
│  │                    Request Router                               │ │
│  └──────┬─────────────────┬────────────────────────┬──────────────┘ │
└─────────┼─────────────────┼────────────────────────┼────────────────┘
          │                 │                        │
┌─────────▼───────┐ ┌──────▼──────────┐ ┌───────────▼───────────────┐
│  Check Engine   │ │  Schema Engine  │ │  Sync Engine              │
│  (tenant-scoped)│ │  (tenant-scoped)│ │  (tenant-scoped)          │
│                 │ │                 │ │                           │
│  • Graph Walk   │ │  • Validation   │ │  • Changelog (WAL)       │
│  • ABAC Eval    │ │  • Compilation  │ │  • Snapshot export       │
│  • Caching      │ │  • Per-tenant   │ │  • Incremental sync      │
│  • Zanzibar     │ │    or shared    │ │  • Bulk import pipeline  │
│    algorithm    │ │    schema modes │ │  • Webhook notifications │
│                 │ │                 │ │  • Conflict resolution   │
└────────┬────────┘ └────────┬────────┘ └───────────┬───────────────┘
         │                   │                      │
┌────────▼───────────────────▼──────────────────────▼───────────────┐
│                     Storage Abstraction Layer                      │
│            (all queries automatically scoped by tenant_id)         │
│  ┌──────────────┐  ┌─────────────────┐  ┌──────────────────────┐ │
│  │  Relation    │  │  Schema Store   │  │  Changelog Store     │ │
│  │  Tuple Store │  │                 │  │  (Append-only WAL)   │ │
│  └──────────────┘  └─────────────────┘  └──────────────────────┘ │
│                                                                   │
│  Backends: PostgreSQL (primary) │ SQLite (embedded) │ Memory      │
└───────────────────────────────────────────────────────────────────┘
```

---

## 3. Multi-Tenancy Model

### 3.1 Tenancy Architecture

ZanGuard supports multi-tenancy as a first-class concept. Every authorization operation executes within a **tenant context** — there is no "global" namespace for relation tuples or permissions.

```
┌─────────────────────────────────────────────────────────┐
│                    Platform Level                         │
│  ┌─────────────────────────────────────────────────────┐ │
│  │  Tenant Registry                                     │ │
│  │  • Tenant CRUD, lifecycle (active/suspended/deleted) │ │
│  │  • Per-tenant configuration & quotas                 │ │
│  │  • Tenant hierarchy (parent → child organizations)   │ │
│  └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
         │                    │                    │
┌────────▼──────┐   ┌────────▼──────┐   ┌────────▼──────┐
│  Tenant: acme │   │ Tenant: globex│   │ Tenant: initech│
│               │   │               │   │               │
│ ┌───────────┐ │   │ ┌───────────┐ │   │ ┌───────────┐ │
│ │  Schema   │ │   │ │  Schema   │ │   │ │  Schema   │ │
│ │ (own or   │ │   │ │ (shared)  │ │   │ │ (own)     │ │
│ │  shared)  │ │   │ └───────────┘ │   │ └───────────┘ │
│ └───────────┘ │   │ ┌───────────┐ │   │ ┌───────────┐ │
│ ┌───────────┐ │   │ │  Tuples   │ │   │ │  Tuples   │ │
│ │  Tuples   │ │   │ │ (isolated)│ │   │ │ (isolated)│ │
│ │ (isolated)│ │   │ └───────────┘ │   │ └───────────┘ │
│ └───────────┘ │   │ ┌───────────┐ │   │ ┌───────────┐ │
│ ┌───────────┐ │   │ │  Cache    │ │   │ │  Cache    │ │
│ │  Cache    │ │   │ │(partitioned)│   │ │(partitioned)│
│ │(partitioned)│   │ └───────────┘ │   │ └───────────┘ │
│ └───────────┘ │   │ ┌───────────┐ │   │ ┌───────────┐ │
│ ┌───────────┐ │   │ │ Changelog │ │   │ │ Changelog │ │
│ │ Changelog │ │   │ │ (isolated)│ │   │ │ (isolated)│ │
│ │ (isolated)│ │   │ └───────────┘ │   │ └───────────┘ │
│ └───────────┘ │   └───────────────┘   └───────────────┘
└───────────────┘
```

### 3.2 Tenant Data Model

```go
type Tenant struct {
    ID              string            `json:"id"`              // unique slug: "acme", "globex"
    DisplayName     string            `json:"display_name"`
    ParentTenantID  string            `json:"parent_tenant_id,omitempty"` // for hierarchical orgs
    Status          TenantStatus      `json:"status"`          // active, suspended, deleted
    SchemaMode      SchemaMode        `json:"schema_mode"`     // own, shared, inherited
    SharedSchemaRef string            `json:"shared_schema_ref,omitempty"` // if mode=shared/inherited
    Config          TenantConfig      `json:"config"`
    CreatedAt       time.Time         `json:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at"`
}

type TenantStatus string
const (
    TenantActive    TenantStatus = "active"
    TenantSuspended TenantStatus = "suspended"    // read-only, checks still work
    TenantDeleted   TenantStatus = "deleted"       // soft-deleted, data retained per policy
)

type SchemaMode string
const (
    SchemaOwn       SchemaMode = "own"        // tenant has its own schema
    SchemaShared    SchemaMode = "shared"      // uses a platform-wide shared schema
    SchemaInherited SchemaMode = "inherited"   // inherits parent tenant's schema + can extend
)

type TenantConfig struct {
    MaxTuples          int64         `json:"max_tuples"`           // quota: 0 = unlimited
    MaxRequestsPerSec  int           `json:"max_requests_per_sec"` // rate limit
    CacheTTLOverride   *Duration     `json:"cache_ttl_override"`   // override global cache TTL
    AllowedObjectTypes []string      `json:"allowed_object_types"` // restrict to subset of schema
    RetentionDays      int           `json:"retention_days"`       // changelog retention
    SyncEnabled        bool          `json:"sync_enabled"`         // enable/disable sync API
    WebhookURL         string        `json:"webhook_url"`          // per-tenant change notifications
    Metadata           map[string]any `json:"metadata"`            // custom tenant metadata
}
```

### 3.3 Tenant Resolution

Tenant context is resolved from incoming requests via a middleware chain. Multiple strategies are supported simultaneously with configurable priority:

```go
type TenantResolver interface {
    Resolve(ctx context.Context, r *http.Request) (string, error)
}

// Resolution strategies (evaluated in order until one succeeds)
type TenantResolverChain struct {
    resolvers []TenantResolver
}
```

| Strategy | Source | Example | Best For |
|----------|--------|---------|----------|
| **Header** | `X-Tenant-ID` header | `X-Tenant-ID: acme` | Service-to-service calls |
| **Path prefix** | URL path segment | `/t/acme/access/v1/evaluation` | REST API consumers |
| **JWT claim** | Token claim (configurable) | `"org_id": "acme"` in JWT | OIDC-integrated deployments |
| **API key** | Key → tenant mapping | `zg_acme_sk_xxx` → `acme` | Simple integrations |
| **Subdomain** | Host header | `acme.zanguard.example.com` | SaaS deployments |
| **Static** | Server config | `default_tenant: acme` | Single-tenant embedded mode |

```go
// Tenant context flows through the entire call chain
type TenantContext struct {
    TenantID   string
    Tenant     *Tenant           // full tenant object (cached)
    SchemaHash string            // active schema version for this tenant
    Config     *TenantConfig     // resolved config (with inheritance)
}

// Extracted from context anywhere in the codebase
func TenantFromContext(ctx context.Context) *TenantContext {
    return ctx.Value(tenantContextKey).(*TenantContext)
}
```

### 3.4 Schema Modes

**Own schema:** Tenant has full control over their authorization schema. Best for enterprises with unique access models.

```yaml
# Tenant "acme" has their own schema
tenants:
  acme:
    schema_mode: own
    schema_path: ./schemas/acme.zanguard.yaml
```

**Shared schema:** Multiple tenants use the same schema definition. The platform operator defines the schema, and tenants just populate tuples. Best for SaaS platforms where all tenants have the same resource model.

```yaml
# Tenants "globex" and "initech" share the platform schema
tenants:
  globex:
    schema_mode: shared
    shared_schema_ref: "platform-v1"
  initech:
    schema_mode: shared
    shared_schema_ref: "platform-v1"
```

**Inherited schema:** Tenant inherits a parent's schema and can extend it with additional types/relations. Useful for hierarchical organizations (e.g., parent company with subsidiaries).

```yaml
# "acme-eu" inherits from "acme" and adds region-specific types
tenants:
  acme-eu:
    schema_mode: inherited
    parent_tenant_id: acme
    schema_extensions:
      types:
        gdpr_data_subject:
          relations:
            controller:
              types: [user, organization]
          permissions:
            access:
              intersect:
                - controller
                - condition: subject.region == "EU"
```

### 3.5 Cross-Tenant Operations

By default, tenant data is fully isolated — a check in tenant A can never traverse tuples belonging to tenant B. However, for hierarchical organizations, controlled cross-tenant references are supported:

```yaml
# Platform config
cross_tenant:
  enabled: true
  mode: "explicit"   # explicit | hierarchical | disabled

  # Explicit grants: tenant A can reference subjects from tenant B
  grants:
    - from_tenant: "acme-eu"
      to_tenant: "acme"
      allowed_subject_types: ["user", "group"]
      allowed_relations: ["member", "viewer"]

  # Hierarchical: child tenants can reference parent tenant subjects
  hierarchical:
    inherit_subjects: true        # child can reference parent's users/groups
    inherit_tuples: false          # child does NOT see parent's relation tuples
```

**Implementation:** Cross-tenant references use a prefixed subject format:

```
document:readme#viewer@[acme]user:thilina
                        ^^^^^
                    tenant prefix (only for cross-tenant refs)
```

The check engine resolves cross-tenant subjects by:
1. Detecting the tenant prefix
2. Validating against the cross-tenant grant policy
3. Performing the subject lookup in the referenced tenant's tuple store

### 3.6 Tenant Lifecycle

```
  create        activate       suspend         delete
    │               │              │               │
    ▼               ▼              ▼               ▼
┌────────┐    ┌──────────┐   ┌───────────┐   ┌─────────┐
│ pending │───▶│  active  │──▶│ suspended │──▶│ deleted │
│         │    │          │   │           │   │         │
│ schema  │    │ full r/w │   │ read-only │   │ data    │
│ setup   │    │ access   │   │ checks OK │   │ purged  │
│         │    │          │   │ no writes │   │ after   │
│         │    │          │◀──│           │   │ retention│
└────────┘    └──────────┘   └───────────┘   └─────────┘
                                                   │
                                              ┌────▼─────┐
                                              │  purged  │
                                              │ (hard    │
                                              │  delete) │
                                              └──────────┘
```

---

## 4. Data Model

### 4.1 Relation Tuples (Core Storage Unit)

Following Zanzibar's model, every authorization fact is a **tuple**:

```
<object_type>:<object_id>#<relation>@<subject_type>:<subject_id>[#<subject_relation>]
```

**Examples:**

```
document:readme#viewer@user:thilina
document:readme#viewer@group:engineering#member
folder:root#parent@document:readme
organization:acme#admin@user:thilina
```

**Go struct:**

```go
type RelationTuple struct {
    TenantID        string    `json:"tenant_id"`              // tenant isolation key
    ObjectType      string    `json:"object_type"`
    ObjectID        string    `json:"object_id"`
    Relation        string    `json:"relation"`
    SubjectType     string    `json:"subject_type"`
    SubjectID       string    `json:"subject_id"`
    SubjectRelation string    `json:"subject_relation,omitempty"` // for userset rewrites
    SubjectTenantID string    `json:"subject_tenant_id,omitempty"` // cross-tenant refs only
    Attributes      MapClaims `json:"attributes,omitempty"`       // ABAC overlay
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    SourceSystem    string    `json:"source_system,omitempty"`    // migration tracking
    ExternalID      string    `json:"external_id,omitempty"`      // original ID from source
}
```

> **Key migration fields:** `SourceSystem` and `ExternalID` allow tracking the origin of every tuple, enabling bidirectional sync and deduplication during migration.

### 4.2 Attribute Overlay (ABAC Extension)

Rather than building a separate ABAC engine, attributes attach directly to tuples and objects:

```go
type ObjectAttributes struct {
    TenantID    string            `json:"tenant_id"`
    ObjectType  string            `json:"object_type"`
    ObjectID    string            `json:"object_id"`
    Attributes  map[string]any    `json:"attributes"`  // e.g., {"classification": "confidential", "department": "engineering"}
    UpdatedAt   time.Time         `json:"updated_at"`
}

type SubjectAttributes struct {
    TenantID    string           `json:"tenant_id"`
    SubjectType string           `json:"subject_type"`
    SubjectID   string           `json:"subject_id"`
    Attributes  map[string]any   `json:"attributes"`  // e.g., {"clearance_level": 3, "region": "APAC"}
    UpdatedAt   time.Time        `json:"updated_at"`
}
```

Conditions in the policy schema reference these attributes for hybrid ReBAC+ABAC checks.

---

## 5. Policy Schema Definition

### 5.1 Schema DSL (YAML-based)

The policy schema uses a declarative YAML format — intentionally simpler than Rego or CEL for day-one readability:

```yaml
# schema.zanguard.yaml
version: "1.0"

types:
  user:
    attributes:
      clearance_level: int
      department: string
      region: string

  group:
    relations:
      member:
        types: [user]
      admin:
        types: [user]
    permissions:
      manage:
        resolve: admin

  folder:
    relations:
      owner:
        types: [user]
      viewer:
        types: [user, group#member]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - owner
          - parent->view           # inherited from parent folder
      edit:
        resolve: owner
      delete:
        intersect:
          - owner
          - condition: subject.clearance_level >= 3

  document:
    attributes:
      classification: string       # public, internal, confidential, restricted
      department: string
    relations:
      owner:
        types: [user]
      editor:
        types: [user, group#member]
      viewer:
        types: [user, group#member]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - editor
          - owner
          - parent->view           # folder inheritance
        condition: >               # ABAC gate on top of ReBAC
          object.classification != "restricted"
          OR subject.clearance_level >= 4
      edit:
        union:
          - editor
          - owner
        condition: >
          object.department == subject.department
          OR subject.clearance_level >= 3
      delete:
        resolve: owner
      share:
        intersect:
          - owner
          - condition: object.classification != "restricted"
```

### 5.2 Schema Compilation

The YAML schema compiles to an in-memory graph structure:

```go
type CompiledSchema struct {
    Types       map[string]*TypeDef
    Version     string
    Hash        string          // content hash for cache invalidation
    CompiledAt  time.Time
}

type TypeDef struct {
    Name        string
    Attributes  map[string]AttributeDef
    Relations   map[string]*RelationDef
    Permissions map[string]*PermissionDef
}

type PermissionDef struct {
    Name       string
    Operation  PermOp              // UNION, INTERSECT, EXCLUSION
    Children   []*PermissionRef    // relation refs, arrow ops, nested perms
    Condition  *ConditionExpr      // optional ABAC condition
}

type ConditionExpr struct {
    Raw        string              // original expression
    Compiled   *vm.Program         // compiled expr-lang program
}
```

**Condition evaluation** uses [`expr-lang/expr`](https://github.com/expr-lang/expr) — a safe, fast expression evaluator for Go. No full CEL/Rego overhead; just simple boolean expressions over attributes.

---

## 6. Check Engine (Zanzibar Algorithm)

### 6.1 Permission Check Flow

```
                    Check Request
                         │
                    ┌────▼─────┐
                    │  Cache    │──hit──▶ Return cached result
                    │  Lookup   │
                    └────┬─────┘
                         │ miss
                    ┌────▼─────────────┐
                    │  Schema Resolve   │
                    │  (get permission  │
                    │   definition)     │
                    └────┬─────────────┘
                         │
              ┌──────────▼──────────┐
              │  Permission Tree    │
              │  Walker             │
              └──────────┬──────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐    ┌─────▼─────┐   ┌─────▼─────┐
    │ Direct  │    │  Computed │   │ Inherited │
    │ Lookup  │    │  Userset  │   │ (Arrow)   │
    │         │    │  Rewrite  │   │  Traverse │
    └────┬────┘    └─────┬─────┘   └─────┬─────┘
         │               │               │
         └───────────────┼───────────────┘
                         │
                  ┌──────▼──────┐
                  │  Condition  │──false──▶ DENY
                  │  Evaluator  │
                  │  (ABAC)     │
                  └──────┬──────┘
                         │ true
                    ┌────▼─────┐
                    │  Cache   │
                    │  Store   │
                    └────┬─────┘
                         │
                      ALLOW
```

### 6.2 Graph Walk Algorithm

```go
func (e *CheckEngine) Check(ctx context.Context, req *CheckRequest) (*CheckResult, error) {
    // 0. Tenant context (injected by middleware, enforced everywhere)
    tc := TenantFromContext(ctx)
    if tc.Tenant.Status == TenantDeleted {
        return Deny(), ErrTenantDeleted
    }

    // 1. Cache lookup (tenant-scoped key)
    cacheKey := e.cache.checkCacheKey(tc.TenantID, req.ObjectType, req.ObjectID,
        req.Permission, req.SubjectType, req.SubjectID)
    if cached, ok := e.cache.Get(cacheKey); ok {
        return cached, nil
    }

    // 2. Get compiled permission definition (per-tenant or shared schema)
    schema := e.schemaForTenant(tc)
    permDef, err := schema.GetPermission(req.ObjectType, req.Permission)
    if err != nil {
        return Deny(), err
    }

    // 3. Walk the permission tree (recursive with cycle detection)
    result, err := e.walkPermission(ctx, req, permDef, newVisitedSet())
    if err != nil {
        return Deny(), err
    }

    // 4. Evaluate ABAC condition if present
    if result.Allowed && permDef.Condition != nil {
        objAttrs, _ := e.store.GetObjectAttributes(ctx, req.ObjectType, req.ObjectID)
        subAttrs, _ := e.store.GetSubjectAttributes(ctx, req.SubjectType, req.SubjectID)

        env := map[string]any{
            "object":  objAttrs,
            "subject": subAttrs,
            "request": req.Context,       // request-time context (IP, time, etc.)
            "tenant":  tc.Tenant.Metadata, // tenant-level attributes available in conditions
        }
        condResult, _ := expr.Run(permDef.Condition.Compiled, env)
        result.Allowed = condResult.(bool)
    }

    // 5. Cache and return (tenant-scoped, respects per-tenant TTL override)
    ttl := e.config.CacheTTL
    if tc.Config.CacheTTLOverride != nil {
        ttl = *tc.Config.CacheTTLOverride
    }
    e.cache.Set(cacheKey, result, ttl)
    return result, nil
}
```

### 6.3 Arrow Operations (Inherited Permissions)

The `parent->view` syntax traverses relationships:

```go
// parent->view means:
// 1. Find all tuples: <object_type>:<object_id>#parent@<target_type>:<target_id>
// 2. For each target, check: does subject have "view" on <target_type>:<target_id>?
func (e *CheckEngine) walkArrow(ctx context.Context, req *CheckRequest,
    relation string, permission string, visited *VisitedSet) (*CheckResult, error) {

    // Get all objects connected via the relation
    targets, err := e.store.ListRelatedObjects(ctx, req.ObjectType, req.ObjectID, relation)
    if err != nil {
        return Deny(), err
    }

    for _, target := range targets {
        subReq := &CheckRequest{
            ObjectType:  target.ObjectType,
            ObjectID:    target.ObjectID,
            Permission:  permission,
            SubjectType: req.SubjectType,
            SubjectID:   req.SubjectID,
        }

        if visited.Has(subReq.CacheKey()) {
            continue // cycle detection
        }
        visited.Add(subReq.CacheKey())

        result, err := e.Check(ctx, subReq)
        if err == nil && result.Allowed {
            return Allow(), nil
        }
    }
    return Deny(), nil
}
```

---

## 7. Caching Strategy

### 7.1 Per-Instance In-Memory Cache (Tenant-Partitioned)

No global distributed cache — each instance maintains its own cache. All cache keys are prefixed with `tenant_id` for strict isolation.

```go
type CacheConfig struct {
    MaxEntries          int           `yaml:"max_entries"`        // default: 100,000 (global)
    MaxEntriesPerTenant int           `yaml:"max_entries_per_tenant"` // default: 0 (unlimited, capped by global)
    DefaultTTL          time.Duration `yaml:"default_ttl"`        // default: 60s
    SchemaTTL           time.Duration `yaml:"schema_ttl"`         // default: 300s
    NegativeCacheTTL    time.Duration `yaml:"negative_cache_ttl"` // default: 30s
    EvictionPolicy      string        `yaml:"eviction_policy"`    // "lru" | "lfu" | "arc"
}

type CacheLayer struct {
    // L1: Hot path — recent check results (key: tenant:object:relation:subject)
    checkCache   *ristretto.Cache  // dgraph-io/ristretto for concurrent LFU

    // L2: Tuple existence — "does this relation exist?" (key: tenant:tuple_hash)
    tupleCache   *ristretto.Cache

    // L3: Schema — compiled permission definitions per tenant
    schemaCache  *sync.Map         // key: tenant_id → *CompiledSchema

    // L4: Tenant config — resolved tenant objects
    tenantCache  *ristretto.Cache  // key: tenant_id → *TenantContext

    config       CacheConfig
}

// Cache key always includes tenant for isolation
func (c *CacheLayer) checkCacheKey(tenantID, objectType, objectID, relation, subjectType, subjectID string) string {
    return fmt.Sprintf("%s:%s:%s#%s@%s:%s", tenantID, objectType, objectID, relation, subjectType, subjectID)
}
```

### 7.2 Cache Invalidation

```
Tuple Write ──▶ Changelog Entry ──▶ Invalidation Fan-out
                                          │
                     ┌────────────────────┼──────────────────┐
                     │                    │                   │
              ┌──────▼──────┐    ┌───────▼──────┐   ┌───────▼──────┐
              │ Invalidate  │    │ Invalidate   │   │  Publish     │
              │ check cache │    │ tuple cache  │   │  to watchers │
              │ (prefix)    │    │ (exact key)  │   │  (changelog) │
              └─────────────┘    └──────────────┘   └──────────────┘
```

- **Tuple writes** invalidate all check cache entries whose key prefixes match the affected tenant + object or subject.
- **Schema changes** flush check cache entries for the affected tenant only (not all tenants).
- **Tenant suspension** freezes the tenant's cache (no new writes, reads still served).
- Uses **Zanzibar zookies (tokens)** — each check response includes a snapshot token. Clients can demand "at least as fresh as" consistency.

### 7.3 Consistency Tokens (Zookies)

```go
type ConsistencyToken struct {
    TenantID       string    `json:"tid"`
    SchemaVersion  string    `json:"sv"`
    ChangelogSeq   uint64    `json:"seq"`   // per-tenant changelog sequence number
    Timestamp      time.Time `json:"ts"`
}

// Encoded as opaque base64 token in API responses
// Client sends back: "consistency": "at_least_as_fresh" with token
```

---

## 8. AuthZen 1.0 API

### 8.1 Evaluation Endpoint

Tenant is resolved from request context (header, path, JWT), not from the body.

```
POST /access/v1/evaluation
X-Tenant-ID: acme

# Or path-based:
POST /t/acme/access/v1/evaluation
```

```json
{
  "subject": {
    "type": "user",
    "id": "thilina",
    "properties": {
      "clearance_level": 3,
      "department": "engineering"
    }
  },
  "resource": {
    "type": "document",
    "id": "design-doc-42",
    "properties": {
      "classification": "confidential"
    }
  },
  "action": {
    "name": "view"
  },
  "context": {
    "ip_address": "10.0.0.1",
    "request_time": "2025-02-06T10:00:00Z"
  }
}
```

**Response:**

```json
{
  "decision": true,
  "context": {
    "reason_admin": {
      "tenant_id": "acme",
      "resolution_path": [
        "document:design-doc-42#viewer@group:engineering#member",
        "group:engineering#member@user:thilina"
      ],
      "condition_evaluated": true,
      "cache_hit": false,
      "latency_ms": 2.4,
      "consistency_token": "eyJ0aWQiOiJhY21lIiwic3YiOiIxLjAiLCJzZXEiOjQyfQ=="
    }
  }
}
```

### 8.2 Batch Evaluations

```
POST /access/v1/evaluations
```

Accepts an array of evaluation requests, returns an array of decisions. Internally parallelized with goroutine pool.

### 8.3 AuthZen Subject & Resource Discovery (Optional)

```
GET /access/v1/evaluation/subject/{subjectType}/{subjectId}/resources?action=view&resourceType=document
```

Returns all resources a subject can access (reverse lookup / list objects).

---

## 9. Data Migration & Sync Engine

**This is the primary differentiator.** The entire storage layer is designed around making migration seamless.

### 9.1 Migration Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Migration Pipeline                   │
│                                                       │
│  ┌─────────┐   ┌──────────┐   ┌─────────┐   ┌─────┐│
│  │ Extract │──▶│Transform │──▶│  Load   │──▶│Verify││
│  │         │   │          │   │         │   │      ││
│  │ Source  │   │ Mapping  │   │ Batch   │   │Diff  ││
│  │ Adapter │   │ Rules    │   │ Writer  │   │Check ││
│  └─────────┘   └──────────┘   └─────────┘   └─────┘│
│       │                                              │
│  ┌────▼────────────────────────────────────────────┐ │
│  │            Source Adapters                        │ │
│  │                                                  │ │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐ │ │
│  │  │SpiceDB │ │ Casbin │ │  OPA   │ │  Custom  │ │ │
│  │  │ import │ │ import │ │ import │ │   CSV/   │ │ │
│  │  │        │ │        │ │        │ │   JSON   │ │ │
│  │  └────────┘ └────────┘ └────────┘ └──────────┘ │ │
│  │  ┌────────┐ ┌────────┐ ┌────────┐              │ │
│  │  │  LDAP  │ │  RBAC  │ │Keycloak│              │ │
│  │  │ groups │ │  DB    │ │ export │              │ │
│  │  └────────┘ └────────┘ └────────┘              │ │
│  └──────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

### 9.2 Universal Import Format

Every migration flows through a universal intermediate format. Imports are always targeted at a specific tenant.

```yaml
# migration-bundle.yaml
metadata:
  source_system: "spicedb"
  source_version: "1.30.0"
  exported_at: "2025-02-06T10:00:00Z"
  total_tuples: 15420
  target_tenant: "acme"          # which tenant to import into (required)
  create_tenant: true             # auto-create tenant if it doesn't exist
  tenant_config:                  # optional: configure tenant on creation
    display_name: "Acme Corp"
    schema_mode: "own"
    max_tuples: 1000000

schema:
  # Optional: schema mapping hints
  type_mappings:
    "spicedb/user": "user"
    "spicedb/document": "document"
  relation_mappings:
    "reader": "viewer"          # rename relations during import

tuples:
  - object: "document:readme"
    relation: "viewer"
    subject: "user:thilina"
    source_id: "spicedb-tuple-001"      # original ID for tracking
  - object: "document:readme"
    relation: "viewer"
    subject: "group:engineering#member"
    source_id: "spicedb-tuple-002"

attributes:
  - object: "document:readme"
    attrs:
      classification: "internal"
      department: "engineering"
```

**Multi-tenant bulk import:** For migrating an entire platform with multiple tenants:

```yaml
# multi-tenant-bundle.yaml
metadata:
  source_system: "keycloak"
  multi_tenant: true

tenants:
  - id: "acme"
    display_name: "Acme Corp"
    config:
      schema_mode: shared
      shared_schema_ref: "platform-v1"
    tuples:
      - object: "document:readme"
        relation: "viewer"
        subject: "user:thilina"
    attributes:
      - object: "document:readme"
        attrs: { classification: "internal" }

  - id: "globex"
    display_name: "Globex Inc"
    config:
      schema_mode: shared
      shared_schema_ref: "platform-v1"
    tuples:
      - object: "project:alpha"
        relation: "owner"
        subject: "user:hank"
```

### 9.3 Source Adapters

```go
type SourceAdapter interface {
    // Connect to the source system
    Connect(ctx context.Context, config map[string]any) error

    // Extract schema (if available)
    ExtractSchema(ctx context.Context) (*MigrationSchema, error)

    // Stream tuples in batches
    StreamTuples(ctx context.Context, batchSize int) (<-chan TupleBatch, error)

    // Get a point-in-time snapshot token for consistency
    Snapshot(ctx context.Context) (string, error)

    // Watch for changes since a snapshot (for incremental sync)
    Watch(ctx context.Context, sinceSnapshot string) (<-chan ChangeEvent, error)

    // Validate all data was migrated
    Validate(ctx context.Context, snapshot string) (*ValidationReport, error)
}
```

**Built-in adapters:**

| Adapter | Source | Migration Type |
|---------|--------|----------------|
| `spicedb` | SpiceDB gRPC | Full + incremental |
| `casbin` | Casbin CSV/DB policies | Full (one-time) |
| `opa` | OPA bundle (data.json) | Full (one-time) |
| `ldap` | LDAP groups → relation tuples | Full + incremental |
| `csv` | Generic CSV/JSON files | Full (one-time) |
| `keycloak` | Keycloak REST API | Full + incremental |
| `rbac-db` | Generic role_assignments table | Full (one-time) |

### 9.4 Changelog & Sync Protocol

Every tuple mutation generates an append-only changelog entry:

```go
type ChangelogEntry struct {
    Sequence    uint64          `json:"seq"`
    Operation   ChangeOp        `json:"op"`     // INSERT, DELETE, UPDATE
    Tuple       RelationTuple   `json:"tuple"`
    Timestamp   time.Time       `json:"ts"`
    Actor       string          `json:"actor"`  // who made the change
    Source      string          `json:"source"` // "api", "migration", "sync"
    Metadata    map[string]any  `json:"meta,omitempty"`
}
```

**Sync Endpoints (all tenant-scoped):**

```
# Tenant resolved from header/path/JWT for all endpoints

GET  /sync/v1/changelog?since_seq=100&limit=1000
     → Stream tenant's changelog entries for incremental sync

POST /sync/v1/snapshot
     → Create a consistent snapshot of tenant's data (returns snapshot_id)

GET  /sync/v1/snapshot/{id}/export?format=yaml
     → Download full tenant snapshot as migration bundle

POST /sync/v1/import
     → Bulk import into tenant from migration bundle (streaming, resumable)

POST /sync/v1/watch
     → WebSocket/SSE for real-time change notifications (tenant-filtered)

GET  /sync/v1/diff?source=spicedb&snapshot=abc123
     → Compare tenant's current state with source system

# Tenant management endpoints (platform admin)
POST   /admin/v1/tenants                    → Create tenant
GET    /admin/v1/tenants                    → List tenants
GET    /admin/v1/tenants/{id}               → Get tenant details + stats
PATCH  /admin/v1/tenants/{id}               → Update tenant config
POST   /admin/v1/tenants/{id}/suspend       → Suspend tenant (read-only mode)
POST   /admin/v1/tenants/{id}/activate      → Reactivate suspended tenant
DELETE /admin/v1/tenants/{id}               → Soft-delete tenant
POST   /admin/v1/tenants/{id}/purge         → Hard-delete all tenant data
GET    /admin/v1/tenants/{id}/stats          → Tuple count, cache stats, quota usage
POST   /admin/v1/tenants/{id}/export         → Export entire tenant for migration
POST   /admin/v1/tenants/bulk-import         → Import multi-tenant bundle
```

### 9.5 Bidirectional Sync

```
┌──────────┐        Changelog         ┌──────────┐
│ ZanGuard │◄────── Stream ──────────▶│  Source   │
│ Instance │                          │  System   │
│          │──── Write Tuple ────▶    │          │
│          │◄─── Change Event ───     │          │
│          │                          │          │
│  Conflict│     Last-Write-Wins      │          │
│  Resolver│     + Source Priority     │          │
└──────────┘                          └──────────┘
```

Conflict resolution strategy:
1. **Last-write-wins** by default (timestamp-based)
2. **Source priority** — configurable: e.g., "LDAP wins for group memberships"
3. **Manual resolution queue** for conflicts that can't be auto-resolved

---

## 10. Storage Layer

### 10.1 Database Schema (PostgreSQL)

```sql
-- ============================================================
-- TENANT MANAGEMENT
-- ============================================================

CREATE TABLE tenants (
    id              VARCHAR(128) PRIMARY KEY,          -- slug: "acme", "globex"
    display_name    VARCHAR(256) NOT NULL,
    parent_tenant_id VARCHAR(128) REFERENCES tenants(id),
    status          VARCHAR(32) NOT NULL DEFAULT 'active', -- active, suspended, deleted
    schema_mode     VARCHAR(32) NOT NULL DEFAULT 'own',    -- own, shared, inherited
    shared_schema_ref VARCHAR(128),
    config          JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_tenants_parent ON tenants(parent_tenant_id) WHERE parent_tenant_id IS NOT NULL;
CREATE INDEX idx_tenants_status ON tenants(status);

-- Cross-tenant access grants
CREATE TABLE cross_tenant_grants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_tenant_id  VARCHAR(128) NOT NULL REFERENCES tenants(id),
    to_tenant_id    VARCHAR(128) NOT NULL REFERENCES tenants(id),
    allowed_subject_types TEXT[] NOT NULL,                     -- {"user", "group"}
    allowed_relations     TEXT[] NOT NULL,                     -- {"member", "viewer"}
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (from_tenant_id, to_tenant_id)
);

-- ============================================================
-- CORE RELATION TUPLES (tenant-partitioned)
-- ============================================================

CREATE TABLE relation_tuples (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(128) NOT NULL REFERENCES tenants(id),
    object_type     VARCHAR(128) NOT NULL,
    object_id       VARCHAR(256) NOT NULL,
    relation        VARCHAR(128) NOT NULL,
    subject_type    VARCHAR(128) NOT NULL,
    subject_id      VARCHAR(256) NOT NULL,
    subject_relation VARCHAR(128),           -- for userset rewrites
    subject_tenant_id VARCHAR(128),          -- cross-tenant references

    -- Migration tracking
    source_system   VARCHAR(64),
    external_id     VARCHAR(512),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,             -- soft delete for sync

    UNIQUE (tenant_id, object_type, object_id, relation, subject_type, subject_id, subject_relation)
);

-- All indexes are tenant-prefixed for query isolation and performance
CREATE INDEX idx_tuples_lookup ON relation_tuples(tenant_id, object_type, object_id, relation)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_tuples_subject ON relation_tuples(tenant_id, subject_type, subject_id, subject_relation)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_tuples_source ON relation_tuples(tenant_id, source_system, external_id)
    WHERE source_system IS NOT NULL;

-- Optional: partition by tenant_id for large deployments
-- CREATE TABLE relation_tuples (...) PARTITION BY LIST (tenant_id);
-- CREATE TABLE relation_tuples_acme PARTITION OF relation_tuples FOR VALUES IN ('acme');

-- ============================================================
-- OBJECT & SUBJECT ATTRIBUTES (ABAC) — tenant-scoped
-- ============================================================

CREATE TABLE object_attributes (
    tenant_id       VARCHAR(128) NOT NULL REFERENCES tenants(id),
    object_type     VARCHAR(128) NOT NULL,
    object_id       VARCHAR(256) NOT NULL,
    attributes      JSONB NOT NULL DEFAULT '{}',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, object_type, object_id)
);

CREATE TABLE subject_attributes (
    tenant_id       VARCHAR(128) NOT NULL REFERENCES tenants(id),
    subject_type    VARCHAR(128) NOT NULL,
    subject_id      VARCHAR(256) NOT NULL,
    attributes      JSONB NOT NULL DEFAULT '{}',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, subject_type, subject_id)
);

-- ============================================================
-- CHANGELOG — tenant-scoped, append-only
-- ============================================================

CREATE TABLE changelog (
    sequence        BIGSERIAL PRIMARY KEY,
    tenant_id       VARCHAR(128) NOT NULL REFERENCES tenants(id),
    operation       VARCHAR(8) NOT NULL,      -- INSERT, DELETE, UPDATE
    object_type     VARCHAR(128) NOT NULL,
    object_id       VARCHAR(256) NOT NULL,
    relation        VARCHAR(128) NOT NULL,
    subject_type    VARCHAR(128) NOT NULL,
    subject_id      VARCHAR(256) NOT NULL,
    subject_relation VARCHAR(128),
    actor           VARCHAR(256),
    source          VARCHAR(32) NOT NULL,     -- api, migration, sync
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_changelog_tenant_seq ON changelog(tenant_id, sequence);
CREATE INDEX idx_changelog_tenant_time ON changelog(tenant_id, created_at);

-- ============================================================
-- SCHEMA VERSIONS — per-tenant or shared
-- ============================================================

CREATE TABLE schema_versions (
    tenant_id       VARCHAR(128),             -- NULL for shared/platform schemas
    version         VARCHAR(64) NOT NULL,
    schema_yaml     TEXT NOT NULL,
    schema_hash     VARCHAR(64) NOT NULL,
    compiled_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active       BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (COALESCE(tenant_id, '__shared__'), version)
);

CREATE INDEX idx_schema_active ON schema_versions(tenant_id) WHERE is_active = TRUE;

-- ============================================================
-- MIGRATION JOBS — tenant-scoped
-- ============================================================

CREATE TABLE migration_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(128) NOT NULL REFERENCES tenants(id),
    source_system   VARCHAR(64) NOT NULL,
    status          VARCHAR(32) NOT NULL,     -- pending, running, paused, completed, failed
    config          JSONB NOT NULL,
    progress        JSONB,                    -- { "total": 15420, "processed": 8000, "errors": 3 }
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    error_log       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_migration_tenant ON migration_jobs(tenant_id, status);
```

### 10.2 Storage Interface

All operations require a tenant context (extracted from `ctx`). The storage layer enforces tenant isolation — it is impossible to accidentally query across tenants.

```go
type TupleStore interface {
    // Tenant management
    CreateTenant(ctx context.Context, tenant *Tenant) error
    GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
    UpdateTenant(ctx context.Context, tenant *Tenant) error
    ListTenants(ctx context.Context, filter *TenantFilter) ([]*Tenant, error)

    // Core CRUD (tenant derived from ctx)
    WriteTuple(ctx context.Context, tuple *RelationTuple) error
    WriteTuples(ctx context.Context, tuples []*RelationTuple) error  // batch
    DeleteTuple(ctx context.Context, tuple *RelationTuple) error
    ReadTuples(ctx context.Context, filter *TupleFilter) ([]*RelationTuple, error)

    // Zanzibar lookups (all tenant-scoped via ctx)
    CheckDirect(ctx context.Context, object, relation, subject string) (bool, error)
    ListRelatedObjects(ctx context.Context, objectType, objectID, relation string) ([]*ObjectRef, error)
    ListSubjects(ctx context.Context, objectType, objectID, relation string) ([]*SubjectRef, error)
    Expand(ctx context.Context, objectType, objectID, relation string) (*SubjectTree, error)

    // Cross-tenant subject lookup (explicit tenant parameter)
    CheckDirectCrossTenant(ctx context.Context, targetTenantID, object, relation, subject string) (bool, error)

    // Attributes (tenant-scoped)
    GetObjectAttributes(ctx context.Context, objectType, objectID string) (map[string]any, error)
    SetObjectAttributes(ctx context.Context, objectType, objectID string, attrs map[string]any) error

    // Changelog (tenant-scoped)
    AppendChangelog(ctx context.Context, entry *ChangelogEntry) error
    ReadChangelog(ctx context.Context, sinceSeq uint64, limit int) ([]*ChangelogEntry, error)
    LatestSequence(ctx context.Context) (uint64, error)

    // Tenant data operations
    CountTuples(ctx context.Context) (int64, error)                    // for quota enforcement
    PurgeTenantData(ctx context.Context) error                          // hard delete all tenant data
    ExportTenantSnapshot(ctx context.Context, w io.Writer) error        // stream all tuples
}

// Internal helper: every query automatically adds WHERE tenant_id = ?
func tenantScope(ctx context.Context, query *sqlBuilder) *sqlBuilder {
    tc := TenantFromContext(ctx)
    return query.Where("tenant_id = ?", tc.TenantID)
}
```

---

## 11. Project Structure

```
zanguard/
├── cmd/
│   ├── server/             # Standalone server binary
│   │   └── main.go
│   └── zanguard-cli/       # CLI tool for migrations, schema management
│       └── main.go
├── pkg/
│   ├── api/
│   │   ├── authzen/        # AuthZen 1.0 handlers
│   │   ├── admin/          # Schema & relation management
│   │   ├── tenant/         # Tenant CRUD, lifecycle, config
│   │   ├── sync/           # Migration & sync endpoints
│   │   └── middleware/      # Auth, logging, rate limiting, tenant resolution
│   ├── engine/
│   │   ├── check.go        # Permission check (graph walk)
│   │   ├── expand.go       # Subject tree expansion
│   │   ├── condition.go    # ABAC condition evaluator
│   │   └── engine.go       # Engine orchestrator
│   ├── schema/
│   │   ├── parser.go       # YAML schema parser
│   │   ├── compiler.go     # Schema compilation
│   │   ├── validator.go    # Schema validation
│   │   └── types.go        # Type definitions
│   ├── storage/
│   │   ├── interface.go    # TupleStore interface
│   │   ├── postgres/       # PostgreSQL implementation
│   │   ├── sqlite/         # SQLite (embedded mode)
│   │   └── memory/         # In-memory (testing)
│   ├── cache/
│   │   ├── cache.go        # Cache interface & L1/L2/L3/L4
│   │   ├── ristretto.go    # Ristretto-based implementation
│   │   └── invalidation.go # Tenant-scoped cache invalidation logic
│   ├── tenant/
│   │   ├── manager.go      # Tenant CRUD, lifecycle operations
│   │   ├── resolver.go     # Tenant resolution middleware chain
│   │   ├── config.go       # Per-tenant config resolution (with inheritance)
│   │   ├── quota.go        # Quota enforcement (max tuples, rate limits)
│   │   ├── cross_tenant.go # Cross-tenant reference validation
│   │   └── types.go        # Tenant, TenantConfig, TenantStatus
│   ├── migration/
│   │   ├── pipeline.go     # ETL pipeline orchestrator
│   │   ├── format.go       # Universal migration format
│   │   ├── diff.go         # Source ↔ target comparison
│   │   └── adapters/
│   │       ├── adapter.go  # SourceAdapter interface
│   │       ├── spicedb.go
│   │       ├── casbin.go
│   │       ├── opa.go
│   │       ├── ldap.go
│   │       ├── keycloak.go
│   │       └── csv.go
│   ├── sync/
│   │   ├── changelog.go    # Changelog management
│   │   ├── watcher.go      # Real-time change notifications
│   │   ├── snapshot.go     # Point-in-time snapshots
│   │   └── conflict.go     # Conflict resolution
│   └── model/
│       ├── tuple.go        # RelationTuple, ObjectRef, etc.
│       ├── tenant.go       # Tenant, TenantContext, cross-tenant refs
│       ├── attributes.go   # Object/Subject attributes
│       └── changelog.go    # ChangelogEntry
├── configs/
│   ├── server.yaml         # Server configuration
│   └── examples/
│       ├── gdrive.zanguard.yaml     # Google Drive-like schema
│       ├── github.zanguard.yaml     # GitHub-like schema
│       └── multi-tenant.zanguard.yaml
├── migrations/             # Database migrations (golang-migrate)
│   ├── 001_initial.up.sql
│   └── 001_initial.down.sql
├── tests/
│   ├── integration/
│   ├── benchmarks/
│   └── migration/          # Migration adapter tests
├── docs/
│   ├── architecture.md
│   ├── schema-guide.md
│   ├── migration-guide.md
│   └── authzen-compliance.md
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yaml
└── Makefile
```

---

## 12. Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/expr-lang/expr` | ABAC condition evaluation (safe, fast) |
| `github.com/dgraph-io/ristretto` | High-performance concurrent cache |
| `github.com/jackc/pgx/v5` | PostgreSQL driver |
| `github.com/grpc-ecosystem/grpc-gateway` | gRPC + REST dual serving |
| `google.golang.org/grpc` | gRPC server |
| `github.com/golang-migrate/migrate` | Database migrations |
| `github.com/go-chi/chi` | HTTP router (lightweight) |
| `gopkg.in/yaml.v3` | Schema parsing |
| `go.uber.org/zap` | Structured logging |
| `github.com/prometheus/client_golang` | Metrics |
| `go.opentelemetry.io/otel` | Distributed tracing |
| `mattn/go-sqlite3` | SQLite backend |

---

## 13. Implementation Plan

### Phase 1: Core Engine + Tenancy Foundation (Weeks 1–3)

| Week | Deliverable |
|------|-------------|
| 1 | Project scaffold, data model, **tenant model & resolver middleware**, storage interface, PostgreSQL + in-memory backends, DB migrations with tenant tables |
| 2 | Schema parser/compiler (YAML → compiled graph), **per-tenant & shared schema modes**, relation tuple CRUD API (tenant-scoped) |
| 3 | Check engine (direct lookup, userset rewrite, arrow traversal, cycle detection), **tenant-scoped query enforcement**, basic unit tests |

**Milestone:** `Check(tenant="acme", "document:readme", "view", "user:thilina")` works end-to-end with tenant isolation.

### Phase 2: ABAC, Caching & Tenant Config (Weeks 4–5)

| Week | Deliverable |
|------|-------------|
| 4 | Attribute storage, condition evaluator (expr-lang integration), hybrid ReBAC+ABAC checks, **tenant quota enforcement** |
| 5 | Ristretto cache layer (L1/L2/L3/L4), **tenant-partitioned cache**, cache invalidation per tenant, consistency tokens (zookies with tenant_id) |

**Milestone:** ABAC conditions work, cache is tenant-isolated, quotas enforced.

### Phase 3: AuthZen 1.0 API + Tenant Admin (Week 6)

| Week | Deliverable |
|------|-------------|
| 6 | AuthZen `/access/v1/evaluation` and `/evaluations` endpoints with tenant resolution, **tenant CRUD API** (`/admin/v1/tenants`), tenant lifecycle (suspend/activate/delete), OpenAPI spec, compliance test suite |

**Milestone:** Passes AuthZen 1.0 conformance; full tenant management API operational.

### Phase 4: Migration Engine (Weeks 7–9)

| Week | Deliverable |
|------|-------------|
| 7 | Universal migration format (**tenant-targeted + multi-tenant bundles**), ETL pipeline, CSV/JSON adapter, CLI import/export commands |
| 8 | SpiceDB adapter, Casbin adapter, **per-tenant migration job tracking**, progress reporting |
| 9 | LDAP adapter, Keycloak adapter (**multi-realm → multi-tenant mapping**), diff/validation tool, migration documentation |

**Milestone:** One-command migration from SpiceDB / Casbin / LDAP into specific tenants with full validation.

### Phase 5: Sync, Cross-Tenant & Changelog (Weeks 10–11)

| Week | Deliverable |
|------|-------------|
| 10 | Append-only changelog (tenant-scoped), sync endpoints (changelog streaming, snapshot export), WebSocket watcher with **tenant filtering** |
| 11 | Bidirectional sync protocol, conflict resolution, **cross-tenant reference support**, inherited schema mode, incremental sync from SpiceDB/Keycloak |

**Milestone:** Real-time bidirectional sync with SpiceDB; cross-tenant subject references working.

### Phase 6: Hardening & Release (Weeks 12–14)

| Week | Deliverable |
|------|-------------|
| 12 | Integration test suite (**multi-tenant isolation tests**, cross-tenant security boundary tests), benchmark suite (target: <5ms p99 check latency) |
| 13 | **Tenant data purge pipeline**, **PostgreSQL table partitioning by tenant** (optional), load testing with 100+ tenants, SQLite backend |
| 14 | Docker/Helm packaging, documentation site, example schemas (single-tenant, multi-tenant SaaS, hierarchical org), getting-started guide |

**Milestone:** Production-ready v0.1.0 release with full multi-tenancy support.

---

## 14. Performance Targets

| Metric | Target |
|--------|--------|
| Check latency (cached) | < 0.5ms p99 |
| Check latency (uncached, 3-hop) | < 5ms p99 |
| Batch evaluation throughput | > 10,000 checks/sec per instance |
| Migration import throughput | > 50,000 tuples/sec per tenant |
| Changelog write throughput | > 20,000 entries/sec |
| Cache hit ratio (steady state) | > 85% |
| Memory per 1M cached entries | < 500MB |
| Tenant resolution overhead | < 0.1ms (cached tenant config) |
| Cross-tenant check overhead | < 2ms additional vs same-tenant |
| Concurrent tenants per instance | 500+ (with partitioned cache) |
| Tenant data purge (1M tuples) | < 60s |

---

## 15. Configuration Example

```yaml
# server.yaml
server:
  http_port: 8080
  grpc_port: 8081
  metrics_port: 9090

storage:
  backend: postgres         # postgres | sqlite | memory
  postgres:
    dsn: "postgres://zanguard:secret@localhost:5432/zanguard?sslmode=disable"
    max_connections: 50
    max_idle: 10
    partition_by_tenant: false  # enable for large-scale deployments (100+ tenants)

cache:
  max_entries: 100000
  max_entries_per_tenant: 0    # 0 = no per-tenant limit (capped by global max)
  default_ttl: 60s
  schema_ttl: 300s
  negative_cache_ttl: 30s
  eviction_policy: lfu      # lru | lfu | arc

schema:
  path: ./schemas/           # directory of .zanguard.yaml files
  auto_reload: true          # watch for changes

# Multi-tenancy configuration
tenancy:
  enabled: true              # false = single-tenant mode (uses default_tenant)
  default_tenant: "default"  # fallback if no tenant resolved

  # Tenant resolution chain (evaluated in order)
  resolution:
    - type: header
      header_name: "X-Tenant-ID"
    - type: jwt_claim
      claim_path: "org_id"
    - type: path_prefix
      prefix: "/t/"
    - type: api_key           # key prefix → tenant mapping

  # Default config for newly created tenants
  default_tenant_config:
    max_tuples: 1000000
    max_requests_per_sec: 1000
    retention_days: 30
    sync_enabled: true

  # Cross-tenant access
  cross_tenant:
    enabled: false
    mode: "explicit"         # explicit | hierarchical | disabled

  # Tenant lifecycle
  lifecycle:
    purge_after_deletion_days: 90   # hard-delete data N days after soft-delete
    suspended_allows_reads: true     # suspended tenants can still serve checks

sync:
  changelog:
    retention: 720h          # 30 days
    batch_size: 1000
  watchers:
    max_connections: 100
    max_connections_per_tenant: 10

migration:
  batch_size: 5000
  concurrency: 4
  error_threshold: 0.01     # fail if >1% errors
  adapters:
    spicedb:
      endpoint: "localhost:50051"
      insecure: true
    ldap:
      url: "ldap://ldap.example.com:389"
      bind_dn: "cn=admin,dc=example,dc=com"
```

---

## 16. Why This Design for WSO2 Identity Server Customers

1. **Tenant-native, not tenant-bolted** — WSO2 IS is fundamentally multi-tenant. ZanGuard mirrors this with tenant_id baked into every tuple, every cache key, every DB index. Keycloak realms, WSO2 IS tenants, and LDAP directory partitions all map cleanly to ZanGuard tenants.
2. **Three schema modes** — SaaS platforms get `shared` schemas (all tenants same model). Enterprises get `own` schemas. Holding companies get `inherited` schemas (parent + extensions). This maps directly to how WSO2 IS customers deploy.
3. **Migration adapters speak their language** — LDAP groups, Keycloak realms, RBAC tables — these are the exact sources WSO2 IS customers are coming from. Multi-realm Keycloak exports map to multi-tenant bundles.
4. **Changelog-first** — customers can run ZanGuard alongside their existing system, sync incrementally per-tenant, verify parity with the diff tool, then cut over tenant by tenant.
5. **Tenant lifecycle management** — suspend/activate/purge operations mirror what platform operators need. Quota enforcement prevents noisy-neighbor issues.
6. **Simple schema** — YAML policy definitions don't require learning Rego or SpiceDB's schema language. A CIAM developer can read and modify them.
7. **AuthZen compliance** — positions this as a standards-based solution, important for enterprise procurement.
8. **Cross-tenant federation** — controlled cross-tenant subject references support the hierarchical organization patterns common in B2B SaaS (parent org → subsidiary → department).
9. **Embeddable** — can run as a sidecar next to WSO2 IS or as a standalone microservice, fitting into existing deployment patterns. Single-tenant embedded mode (SQLite) works for edge deployments.
