package schema

import (
	"time"

	"github.com/expr-lang/expr/vm"
)

// PermOp defines the logical operation used to combine permission children.
type PermOp int

const (
	PermOpUnion     PermOp = iota // union: first match wins
	PermOpIntersect               // intersect: all must match
	PermOpExclusion               // exclusion: base AND NOT excluded
	PermOpDirect                  // direct: resolve: <relation> sugar
)

// RawSchema is the deserialized YAML schema before compilation.
type RawSchema struct {
	Version string                `yaml:"version"`
	Types   map[string]*RawType   `yaml:"types"`
}

// RawType is a type definition in the raw schema.
type RawType struct {
	Attributes  map[string]string     `yaml:"attributes,omitempty"`
	Relations   map[string]*RawRelation   `yaml:"relations,omitempty"`
	Permissions map[string]*RawPermission `yaml:"permissions,omitempty"`
}

// RawRelation is a relation definition in the raw schema.
type RawRelation struct {
	Types []string `yaml:"types"`
}

// RawPermission is a permission definition in the raw schema.
// Supports: resolve, union, intersect, exclusion (with optional condition).
type RawPermission struct {
	Resolve   string   `yaml:"resolve,omitempty"`
	Union     []string `yaml:"union,omitempty"`
	Intersect []string `yaml:"intersect,omitempty"`
	Exclusion []string `yaml:"exclusion,omitempty"`
	Condition string   `yaml:"condition,omitempty"`
}

// CompiledSchema is the compiled, ready-to-use schema for a tenant.
type CompiledSchema struct {
	Types      map[string]*TypeDef
	Version    string
	Hash       string    // SHA-256 of source YAML for cache invalidation
	CompiledAt time.Time
}

// GetPermission returns the PermissionDef for the given type and permission name.
func (cs *CompiledSchema) GetPermission(objectType, permission string) (*PermissionDef, error) {
	td, ok := cs.Types[objectType]
	if !ok {
		return nil, &ValidationError{Message: "unknown type: " + objectType}
	}
	pd, ok := td.Permissions[permission]
	if !ok {
		return nil, &ValidationError{Message: "unknown permission " + permission + " on type " + objectType}
	}
	return pd, nil
}

// GetRelation returns the RelationDef for the given type and relation name.
func (cs *CompiledSchema) GetRelation(objectType, relation string) (*RelationDef, error) {
	td, ok := cs.Types[objectType]
	if !ok {
		return nil, &ValidationError{Message: "unknown type: " + objectType}
	}
	rd, ok := td.Relations[relation]
	if !ok {
		return nil, &ValidationError{Message: "unknown relation " + relation + " on type " + objectType}
	}
	return rd, nil
}

// TypeDef is the compiled representation of a type.
type TypeDef struct {
	Name        string
	Attributes  map[string]string // attribute name → type string
	Relations   map[string]*RelationDef
	Permissions map[string]*PermissionDef
}

// RelationDef defines the allowed subject types for a relation.
type RelationDef struct {
	Name         string
	AllowedTypes []*AllowedTypeRef
}

// AllowedTypeRef is a type reference allowed in a relation, optionally with a userset relation.
type AllowedTypeRef struct {
	Type     string // e.g., "user"
	Relation string // optional: e.g., "member" for "group#member"
}

// PermissionDef is the compiled permission definition.
type PermissionDef struct {
	Name      string
	Operation PermOp
	Children  []*PermissionRef
	Condition *ConditionExpr
}

// PermissionRef is a reference within a permission definition.
type PermissionRef struct {
	// For direct relation reference: "viewer"
	RelationRef string
	// For arrow: "parent->view" → ArrowRef="parent", ArrowPermission="view"
	ArrowRef        string
	ArrowPermission string
	// For nested condition-only items in intersect/union lists
	ConditionExpr string
}

// Kind returns the kind of permission reference: "relation", "arrow", or "condition".
func (pr *PermissionRef) Kind() string {
	if pr.ArrowRef != "" {
		return "arrow"
	}
	if pr.ConditionExpr != "" {
		return "condition"
	}
	return "relation"
}

// ConditionExpr holds a parsed ABAC condition expression.
type ConditionExpr struct {
	Raw      string
	Compiled *vm.Program
}

// ValidationError represents a schema validation problem.
type ValidationError struct {
	Message string
	Field   string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}
