// TypeScript types mirroring Go models from pkg/model/ and pkg/api/types.go

export type TenantStatus = "pending" | "active" | "suspended" | "deleted";
export type SchemaMode = "own" | "shared" | "inherited";

export interface TenantConfig {
  max_tuples: number;
  max_requests_per_sec: number;
  cache_ttl_override?: number | null;
  allowed_object_types?: string[];
  retention_days: number;
  sync_enabled: boolean;
  webhook_url?: string;
  metadata?: Record<string, unknown>;
}

export interface Tenant {
  id: string;
  display_name: string;
  parent_tenant_id?: string;
  status: TenantStatus;
  schema_mode: SchemaMode;
  shared_schema_ref?: string;
  config: TenantConfig;
  created_at: string;
  updated_at: string;
}

export interface RelationTuple {
  tenant_id: string;
  object_type: string;
  object_id: string;
  relation: string;
  subject_type: string;
  subject_id: string;
  subject_relation?: string;
  subject_tenant_id?: string;
  attributes?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  source_system?: string;
  external_id?: string;
}

export interface SchemaResponse {
  tenant_id: string;
  hash: string;
  version: string;
  source: string;
  compiled_at: string;
}

export interface SubjectRef {
  Type: string;
  ID: string;
  Relation?: string;
}

export interface SubjectTree {
  Subject: SubjectRef;
  Children: SubjectTree[] | null;
}

export type ChangeOp = "INSERT" | "DELETE" | "UPDATE";

export interface ChangelogEntry {
  seq: number;
  tenant_id: string;
  op: ChangeOp;
  tuple: RelationTuple;
  ts: string;
  actor: string;
  source: string;
  meta?: Record<string, unknown>;
}

// Request types

export interface TupleRequest {
  object_type: string;
  object_id: string;
  relation: string;
  subject_type: string;
  subject_id: string;
  subject_relation?: string;
  attributes?: Record<string, unknown>;
}

export interface BatchTuplesRequest {
  tuples: TupleRequest[];
}

export interface ExpandRequest {
  object_type: string;
  object_id: string;
  relation: string;
}

export interface TupleFilter {
  object_type?: string;
  object_id?: string;
  relation?: string;
  subject_type?: string;
  subject_id?: string;
  subject_relation?: string;
}

// Response types

export interface ListTenantsResponse {
  tenants: Tenant[];
  count: number;
}

export interface TuplesResponse {
  tuples: RelationTuple[];
  count: number;
}

export interface ChangelogResponse {
  entries: ChangelogEntry[];
  count: number;
  latest_sequence: number;
}

export interface AttributesResponse {
  attributes: Record<string, unknown>;
}

export interface SchemaValidationError {
  error: string;
  details: string[];
}
