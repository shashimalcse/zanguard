package model

import "time"

// ChangeOp represents the type of change in a changelog entry.
type ChangeOp string

const (
	ChangeOpInsert ChangeOp = "INSERT"
	ChangeOpDelete ChangeOp = "DELETE"
	ChangeOpUpdate ChangeOp = "UPDATE"
)

// ChangelogEntry records every mutation to the tuple store.
type ChangelogEntry struct {
	Sequence  uint64         `json:"seq"`
	TenantID  string         `json:"tenant_id"`
	Operation ChangeOp       `json:"op"`
	Tuple     RelationTuple  `json:"tuple"`
	Timestamp time.Time      `json:"ts"`
	Actor     string         `json:"actor"`
	Source    string         `json:"source"` // "api", "migration", "sync"
	Metadata  map[string]any `json:"meta,omitempty"`
}
