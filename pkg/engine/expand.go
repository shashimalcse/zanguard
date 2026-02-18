package engine

import (
	"context"

	"zanguard/pkg/model"
)

// Expand returns the subject tree for a given object relation.
// This is a thin wrapper over the store's Expand method.
func (e *Engine) Expand(ctx context.Context, objectType, objectID, relation string) (*model.SubjectTree, error) {
	return e.store.Expand(ctx, objectType, objectID, relation)
}
