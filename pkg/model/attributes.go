package model

import "time"

// ObjectAttributes holds ABAC attributes for an object.
type ObjectAttributes struct {
	TenantID   string         `json:"tenant_id"`
	ObjectType string         `json:"object_type"`
	ObjectID   string         `json:"object_id"`
	Attributes map[string]any `json:"attributes"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// SubjectAttributes holds ABAC attributes for a subject.
type SubjectAttributes struct {
	TenantID    string         `json:"tenant_id"`
	SubjectType string         `json:"subject_type"`
	SubjectID   string         `json:"subject_id"`
	Attributes  map[string]any `json:"attributes"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
