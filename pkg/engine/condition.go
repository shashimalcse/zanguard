package engine

import (
	"context"
	"fmt"

	"github.com/expr-lang/expr"

	"zanguard/pkg/schema"
	"zanguard/pkg/storage"
)

// evaluateCondition loads object/subject attributes and runs the ABAC condition.
func evaluateCondition(ctx context.Context, store storage.TupleStore, req *CheckRequest, cond *schema.ConditionExpr) (bool, error) {
	if cond == nil || cond.Compiled == nil {
		return true, nil
	}

	objAttrs, err := store.GetObjectAttributes(ctx, req.ObjectType, req.ObjectID)
	if err != nil {
		objAttrs = map[string]any{}
	}

	subAttrs, err := store.GetSubjectAttributes(ctx, req.SubjectType, req.SubjectID)
	if err != nil {
		subAttrs = map[string]any{}
	}

	tc := ctx.Value(struct{}{}) // not used directly
	_ = tc

	env := map[string]any{
		"object":  objAttrs,
		"subject": subAttrs,
		"request": req.Context,
	}

	result, err := expr.Run(cond.Compiled, env)
	if err != nil {
		return false, fmt.Errorf("evaluate condition %q: %w", cond.Raw, err)
	}

	allowed, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("condition %q did not return bool, got %T", cond.Raw, result)
	}
	return allowed, nil
}
