package api

import "zanguard/pkg/model"

// ── Tenant management ────────────────────────────────────────────────────────

type CreateTenantRequest struct {
	ID              string           `json:"id"`
	DisplayName     string           `json:"display_name"`
	SchemaMode      model.SchemaMode `json:"schema_mode"`
	ParentTenantID  string           `json:"parent_tenant_id,omitempty"`
	SharedSchemaRef string           `json:"shared_schema_ref,omitempty"`
}

type ListTenantsResponse struct {
	Tenants []*model.Tenant `json:"tenants"`
	Count   int             `json:"count"`
}

// ── Schema ───────────────────────────────────────────────────────────────────

type SchemaResponse struct {
	TenantID   string `json:"tenant_id"`
	Hash       string `json:"hash"`
	Version    string `json:"version"`
	Source     string `json:"source"`
	CompiledAt string `json:"compiled_at"`
}

// ── Tuples ───────────────────────────────────────────────────────────────────

type TupleRequest struct {
	ObjectType      string         `json:"object_type"`
	ObjectID        string         `json:"object_id"`
	Relation        string         `json:"relation"`
	SubjectType     string         `json:"subject_type"`
	SubjectID       string         `json:"subject_id"`
	SubjectRelation string         `json:"subject_relation,omitempty"`
	Attributes      map[string]any `json:"attributes,omitempty"`
	TTLSeconds      *int64         `json:"ttl_seconds,omitempty"`
	ExpiresAt       string         `json:"expires_at,omitempty"`
}

type BatchTuplesRequest struct {
	Tuples []TupleRequest `json:"tuples"`
}

type TuplesResponse struct {
	Tuples []*model.RelationTuple `json:"tuples"`
	Count  int                    `json:"count"`
}

// ── Attributes ───────────────────────────────────────────────────────────────

type AttributesRequest struct {
	Attributes map[string]any `json:"attributes"`
}

type AttributesResponse struct {
	Attributes map[string]any `json:"attributes"`
}

type ListObjectAttributesResponse struct {
	Objects []*model.ObjectAttributes `json:"objects"`
	Count   int                       `json:"count"`
}

type ListSubjectAttributesResponse struct {
	Subjects []*model.SubjectAttributes `json:"subjects"`
	Count    int                        `json:"count"`
}

// ── Changelog ────────────────────────────────────────────────────────────────

type ChangelogResponse struct {
	Entries        []*model.ChangelogEntry `json:"entries"`
	Count          int                     `json:"count"`
	LatestSequence uint64                  `json:"latest_sequence"`
}

// ── Expand ───────────────────────────────────────────────────────────────────

type ExpandRequest struct {
	ObjectType string `json:"object_type"`
	ObjectID   string `json:"object_id"`
	Relation   string `json:"relation"`
}

// ── AuthZen 1.0 ──────────────────────────────────────────────────────────────

type AuthZenSubject struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties,omitempty"`
}

type AuthZenResource struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties,omitempty"`
}

type AuthZenAction struct {
	Name       string         `json:"name"`
	Properties map[string]any `json:"properties,omitempty"`
}

// AuthZenEvaluationRequest is the AuthZen 1.0 single-evaluation request body.
type AuthZenEvaluationRequest struct {
	Subject  AuthZenSubject  `json:"subject"`
	Resource AuthZenResource `json:"resource"`
	Action   AuthZenAction   `json:"action"`
	Context  map[string]any  `json:"context,omitempty"`
}

// AuthZenEvaluationResponse is the AuthZen 1.0 response body.
type AuthZenEvaluationResponse struct {
	Decision bool           `json:"decision"`
	Context  map[string]any `json:"context,omitempty"`
}

// AuthZenBatchItem is one entry in a batch evaluation request.
type AuthZenBatchItem struct {
	Resource AuthZenResource `json:"resource"`
	Action   AuthZenAction   `json:"action"`
	Context  map[string]any  `json:"context,omitempty"`
}

// AuthZenBatchRequest is the AuthZen 1.0 batch-evaluation request body.
type AuthZenBatchRequest struct {
	Subject     AuthZenSubject     `json:"subject"`
	Evaluations []AuthZenBatchItem `json:"evaluations"`
}

// AuthZenBatchResponse is the AuthZen 1.0 batch-evaluation response body.
type AuthZenBatchResponse struct {
	Evaluations []AuthZenEvaluationResponse `json:"evaluations"`
}
