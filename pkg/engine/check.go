package engine

import (
	"context"
	"fmt"

	"zanguard/pkg/schema"
)

// visitedSet tracks visited nodes to detect cycles.
type visitedSet struct {
	keys map[string]bool
}

func newVisitedSet() *visitedSet {
	return &visitedSet{keys: make(map[string]bool)}
}

func (v *visitedSet) has(key string) bool {
	return v.keys[key]
}

func (v *visitedSet) add(key string) {
	v.keys[key] = true
}

// visitKey creates a unique key for cycle detection.
func visitKey(objectType, objectID, permission, subjectType, subjectID string) string {
	return objectType + ":" + objectID + "#" + permission + "@" + subjectType + ":" + subjectID
}

// walkPermission dispatches on the PermOp of the permission definition.
func (e *Engine) walkPermission(
	ctx context.Context,
	req *CheckRequest,
	cs *schema.CompiledSchema,
	permDef *schema.PermissionDef,
	visited *visitedSet,
	depth int,
) (*CheckResult, error) {
	if depth > e.cfg.MaxCheckDepth {
		return deny(), fmt.Errorf("max check depth (%d) exceeded — possible cycle", e.cfg.MaxCheckDepth)
	}

	key := visitKey(req.ObjectType, req.ObjectID, permDef.Name, req.SubjectType, req.SubjectID)
	if visited.has(key) {
		return deny(), nil // cycle: deny
	}
	visited.add(key)

	switch permDef.Operation {
	case schema.PermOpUnion, schema.PermOpDirect:
		return e.walkUnion(ctx, req, cs, permDef, visited, depth)
	case schema.PermOpIntersect:
		return e.walkIntersect(ctx, req, cs, permDef, visited, depth)
	case schema.PermOpExclusion:
		return e.walkExclusion(ctx, req, cs, permDef, visited, depth)
	default:
		return deny(), fmt.Errorf("unknown permission operation: %d", permDef.Operation)
	}
}

// walkUnion returns allowed if any child allows.
func (e *Engine) walkUnion(
	ctx context.Context,
	req *CheckRequest,
	cs *schema.CompiledSchema,
	permDef *schema.PermissionDef,
	visited *visitedSet,
	depth int,
) (*CheckResult, error) {
	for _, child := range permDef.Children {
		result, err := e.walkChild(ctx, req, cs, child, permDef, visited, depth)
		if err != nil {
			return deny(), err
		}
		if result.Allowed {
			return result, nil // short-circuit
		}
	}
	return deny(), nil
}

// walkIntersect returns allowed only if all children allow.
func (e *Engine) walkIntersect(
	ctx context.Context,
	req *CheckRequest,
	cs *schema.CompiledSchema,
	permDef *schema.PermissionDef,
	visited *visitedSet,
	depth int,
) (*CheckResult, error) {
	if len(permDef.Children) == 0 {
		return deny(), nil
	}
	for _, child := range permDef.Children {
		result, err := e.walkChild(ctx, req, cs, child, permDef, visited, depth)
		if err != nil {
			return deny(), err
		}
		if !result.Allowed {
			return deny(), nil
		}
	}
	return allow(), nil
}

// walkExclusion returns allowed if first child allows AND no subsequent child allows.
func (e *Engine) walkExclusion(
	ctx context.Context,
	req *CheckRequest,
	cs *schema.CompiledSchema,
	permDef *schema.PermissionDef,
	visited *visitedSet,
	depth int,
) (*CheckResult, error) {
	if len(permDef.Children) == 0 {
		return deny(), nil
	}

	// Base (first child)
	base, err := e.walkChild(ctx, req, cs, permDef.Children[0], permDef, visited, depth)
	if err != nil {
		return deny(), err
	}
	if !base.Allowed {
		return deny(), nil
	}

	// Exclusions (remaining children)
	for _, child := range permDef.Children[1:] {
		excluded, err := e.walkChild(ctx, req, cs, child, permDef, visited, depth)
		if err != nil {
			return deny(), err
		}
		if excluded.Allowed {
			return deny(), nil // base excluded
		}
	}
	return allow(), nil
}

// walkChild dispatches on the kind of PermissionRef.
func (e *Engine) walkChild(
	ctx context.Context,
	req *CheckRequest,
	cs *schema.CompiledSchema,
	child *schema.PermissionRef,
	parentPermDef *schema.PermissionDef,
	visited *visitedSet,
	depth int,
) (*CheckResult, error) {
	switch child.Kind() {
	case "relation":
		return e.walkRelation(ctx, req, cs, child.RelationRef, visited, depth)
	case "arrow":
		return e.walkArrow(ctx, req, cs, child.ArrowRef, child.ArrowPermission, visited, depth)
	case "condition":
		// Inline condition in union/intersect list — evaluate directly
		cond, err := schema.CompileConditionExpr(child.ConditionExpr)
		if err != nil {
			return deny(), err
		}
		allowed, err := evaluateCondition(ctx, e.store, req, cond)
		if err != nil {
			return deny(), err
		}
		if allowed {
			return allow(), nil
		}
		return deny(), nil
	default:
		return deny(), fmt.Errorf("unknown permission ref kind: %s", child.Kind())
	}
}

// walkRelation checks direct tuple membership, then userset expansion.
func (e *Engine) walkRelation(
	ctx context.Context,
	req *CheckRequest,
	cs *schema.CompiledSchema,
	relation string,
	visited *visitedSet,
	depth int,
) (*CheckResult, error) {
	// 1. Direct lookup: is there a tuple <objectType>:<objectID>#<relation>@<subjectType>:<subjectID>?
	ok, err := e.store.CheckDirect(ctx, req.ObjectType, req.ObjectID, relation, req.SubjectType, req.SubjectID)
	if err != nil {
		return deny(), err
	}
	if ok {
		return allow(req.ObjectType + ":" + req.ObjectID + "#" + relation + "@" + req.SubjectType + ":" + req.SubjectID), nil
	}

	// 2. Userset expansion: find tuples with subject_relation set (group#member patterns)
	subjects, err := e.store.ListSubjects(ctx, req.ObjectType, req.ObjectID, relation)
	if err != nil {
		return deny(), err
	}

	for _, subj := range subjects {
		if subj.Relation == "" {
			continue // already checked direct
		}
		// Recurse: does subject have the relation to req subject?
		subReq := &CheckRequest{
			ObjectType:  subj.Type,
			ObjectID:    subj.ID,
			Permission:  subj.Relation,
			SubjectType: req.SubjectType,
			SubjectID:   req.SubjectID,
		}
		subKey := visitKey(subReq.ObjectType, subReq.ObjectID, subReq.Permission, subReq.SubjectType, subReq.SubjectID)
		if visited.has(subKey) {
			continue
		}

		// Look up the relation/permission definition for the userset
		subCS := cs
		subPermDef, err := subCS.GetPermission(subj.Type, subj.Relation)
		if err != nil {
			// Maybe it's a direct relation (not a permission), check as relation
			subOk, subErr := e.store.CheckDirect(ctx, subj.Type, subj.ID, subj.Relation, req.SubjectType, req.SubjectID)
			if subErr != nil {
				continue
			}
			if subOk {
				return allow(subj.Type + ":" + subj.ID + "#" + subj.Relation + "@" + req.SubjectType + ":" + req.SubjectID), nil
			}
			continue
		}

		subResult, subErr := e.walkPermission(ctx, subReq, subCS, subPermDef, visited, depth+1)
		if subErr != nil {
			return deny(), subErr
		}
		if subResult.Allowed {
			return subResult, nil
		}
	}

	return deny(), nil
}

// walkArrow traverses parent->permission arrows.
func (e *Engine) walkArrow(
	ctx context.Context,
	req *CheckRequest,
	cs *schema.CompiledSchema,
	relation string,
	permission string,
	visited *visitedSet,
	depth int,
) (*CheckResult, error) {
	// Find all objects connected via the relation
	targets, err := e.store.ListRelatedObjects(ctx, req.ObjectType, req.ObjectID, relation)
	if err != nil {
		return deny(), err
	}

	for _, target := range targets {
		subReq := &CheckRequest{
			ObjectType:  target.Type,
			ObjectID:    target.ID,
			Permission:  permission,
			SubjectType: req.SubjectType,
			SubjectID:   req.SubjectID,
			Context:     req.Context,
		}

		subKey := visitKey(subReq.ObjectType, subReq.ObjectID, subReq.Permission, subReq.SubjectType, subReq.SubjectID)
		if visited.has(subKey) {
			continue
		}

		permDef, err := cs.GetPermission(target.Type, permission)
		if err != nil {
			continue // target type may not have this permission
		}

		result, err := e.walkPermission(ctx, subReq, cs, permDef, visited, depth+1)
		if err != nil {
			return deny(), err
		}
		if result.Allowed {
			return result, nil
		}
	}
	return deny(), nil
}
