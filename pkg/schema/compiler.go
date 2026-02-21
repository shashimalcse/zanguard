package schema

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

// Compile compiles a RawSchema (plus original YAML bytes for hashing) into a CompiledSchema.
func Compile(raw *RawSchema, originalYAML []byte) (*CompiledSchema, error) {
	cs := &CompiledSchema{
		Version:    raw.Version,
		Hash:       hashYAML(originalYAML),
		CompiledAt: time.Now().UTC(),
		Types:      make(map[string]*TypeDef),
	}

	// First pass: build TypeDefs without cross-references
	for typeName, rawType := range raw.Types {
		td := &TypeDef{
			Name:        typeName,
			Attributes:  rawType.Attributes,
			Relations:   make(map[string]*RelationDef),
			Permissions: make(map[string]*PermissionDef),
		}
		if td.Attributes == nil {
			td.Attributes = make(map[string]string)
		}

		for relName, rawRel := range rawType.Relations {
			rd := &RelationDef{Name: relName}
			for _, typeRef := range rawRel.Types {
				rd.AllowedTypes = append(rd.AllowedTypes, parseAllowedTypeRef(typeRef))
			}
			td.Relations[relName] = rd
		}

		cs.Types[typeName] = td
	}

	// Second pass: compile permissions (needs all types defined for validation)
	for typeName, rawType := range raw.Types {
		td := cs.Types[typeName]
		for permName, rawPerm := range rawType.Permissions {
			pd, err := compilePermission(permName, rawPerm)
			if err != nil {
				return nil, fmt.Errorf("type %q permission %q: %w", typeName, permName, err)
			}
			td.Permissions[permName] = pd
		}
	}

	return cs, nil
}

// hashYAML returns SHA-256 hex of the YAML content.
func hashYAML(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// parseAllowedTypeRef parses "group#member" or "user" into AllowedTypeRef.
func parseAllowedTypeRef(s string) *AllowedTypeRef {
	parts := strings.SplitN(s, "#", 2)
	ref := &AllowedTypeRef{Type: parts[0]}
	if len(parts) == 2 {
		ref.Relation = parts[1]
	}
	return ref
}

// compilePermission compiles a RawPermission into a PermissionDef.
func compilePermission(name string, raw *RawPermission) (*PermissionDef, error) {
	pd := &PermissionDef{Name: name}

	// Compile condition if present
	if raw.Condition != "" {
		cond, err := compileCondition(raw.Condition)
		if err != nil {
			return nil, fmt.Errorf("compile condition: %w", err)
		}
		pd.Condition = cond
	}

	switch {
	case raw.Resolve != "":
		pd.Operation = PermOpDirect
		pd.Children = []*PermissionRef{{RelationRef: raw.Resolve}}

	case len(raw.Union) > 0:
		pd.Operation = PermOpUnion
		for _, ref := range raw.Union {
			pr, err := parsePermissionRef(string(ref))
			if err != nil {
				return nil, err
			}
			pd.Children = append(pd.Children, pr)
		}

	case len(raw.Intersect) > 0:
		pd.Operation = PermOpIntersect
		for _, ref := range raw.Intersect {
			pr, err := parsePermissionRef(string(ref))
			if err != nil {
				return nil, err
			}
			pd.Children = append(pd.Children, pr)
		}

	case len(raw.Exclusion) > 0:
		pd.Operation = PermOpExclusion
		for _, ref := range raw.Exclusion {
			pr, err := parsePermissionRef(string(ref))
			if err != nil {
				return nil, err
			}
			pd.Children = append(pd.Children, pr)
		}

	default:
		return nil, fmt.Errorf("permission %q has no operation (resolve/union/intersect/exclusion)", name)
	}

	return pd, nil
}

// parsePermissionRef parses "parent->view", "viewer", or "condition: ..." into a PermissionRef.
func parsePermissionRef(s string) (*PermissionRef, error) {
	// Arrow: "parent->view"
	if idx := strings.Index(s, "->"); idx >= 0 {
		relation := strings.TrimSpace(s[:idx])
		permission := strings.TrimSpace(s[idx+2:])
		if relation == "" || permission == "" {
			return nil, fmt.Errorf("invalid arrow ref %q", s)
		}
		return &PermissionRef{ArrowRef: relation, ArrowPermission: permission}, nil
	}

	// Condition inline: "condition: <expr>"
	if strings.HasPrefix(s, "condition:") {
		expr := strings.TrimSpace(strings.TrimPrefix(s, "condition:"))
		return &PermissionRef{ConditionExpr: expr}, nil
	}

	// Direct relation reference
	return &PermissionRef{RelationRef: strings.TrimSpace(s)}, nil
}

// CompileConditionExpr is the exported version for use in the engine.
func CompileConditionExpr(raw string) (*ConditionExpr, error) {
	return compileCondition(raw)
}

// compileCondition compiles an ABAC condition expression using expr-lang.
func compileCondition(raw string) (*ConditionExpr, error) {
	// Clean up multi-line YAML block scalar
	cleaned := strings.TrimSpace(raw)

	program, err := expr.Compile(cleaned,
		expr.Env(map[string]any{
			"object":  map[string]any{},
			"subject": map[string]any{},
			"request": map[string]any{},
			"tenant":  map[string]any{},
		}),
		expr.AsBool(),
		expr.AllowUndefinedVariables(),
	)
	if err != nil {
		return nil, fmt.Errorf("compile expression %q: %w", cleaned, err)
	}

	return &ConditionExpr{
		Raw:      raw,
		Compiled: program,
	}, nil
}
