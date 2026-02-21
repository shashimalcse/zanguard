package model

import (
	"fmt"
	"time"
)

// RelationTuple is the core storage unit — a single authorization fact.
// Format: <object_type>:<object_id>#<relation>@<subject_type>:<subject_id>[#<subject_relation>]
type RelationTuple struct {
	TenantID        string         `json:"tenant_id"`
	ObjectType      string         `json:"object_type"`
	ObjectID        string         `json:"object_id"`
	Relation        string         `json:"relation"`
	SubjectType     string         `json:"subject_type"`
	SubjectID       string         `json:"subject_id"`
	SubjectRelation string         `json:"subject_relation,omitempty"`
	SubjectTenantID string         `json:"subject_tenant_id,omitempty"`
	Attributes      map[string]any `json:"attributes,omitempty"`
	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	SourceSystem    string         `json:"source_system,omitempty"`
	ExternalID      string         `json:"external_id,omitempty"`
}

// TupleKey returns the canonical string representation of the tuple.
func (t *RelationTuple) TupleKey() string {
	if t.SubjectRelation != "" {
		return fmt.Sprintf("%s:%s#%s@%s:%s#%s", t.ObjectType, t.ObjectID, t.Relation, t.SubjectType, t.SubjectID, t.SubjectRelation)
	}
	return fmt.Sprintf("%s:%s#%s@%s:%s", t.ObjectType, t.ObjectID, t.Relation, t.SubjectType, t.SubjectID)
}

// ObjectRef is a reference to an object (type + id).
type ObjectRef struct {
	Type string
	ID   string
}

func (o *ObjectRef) String() string {
	return fmt.Sprintf("%s:%s", o.Type, o.ID)
}

// SubjectRef is a reference to a subject, optionally with a relation (for usersets).
type SubjectRef struct {
	Type     string
	ID       string
	Relation string // optional: for userset refs like group:eng#member
}

func (s *SubjectRef) String() string {
	if s.Relation != "" {
		return fmt.Sprintf("%s:%s#%s", s.Type, s.ID, s.Relation)
	}
	return fmt.Sprintf("%s:%s", s.Type, s.ID)
}

// SubjectTree is the expanded tree of subjects for a relation.
type SubjectTree struct {
	Subject  *SubjectRef
	Children []*SubjectTree
}

// TupleFilter is used to query tuples with optional field filters.
type TupleFilter struct {
	ObjectType      string
	ObjectID        string
	Relation        string
	SubjectType     string
	SubjectID       string
	SubjectRelation string
	IncludeExpired  bool
}

// TenantFilter is used to query tenants.
type TenantFilter struct {
	Status   TenantStatus
	ParentID string
	Limit    int
	Offset   int
}
