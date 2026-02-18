package engine

import (
	"context"
	"testing"

	"zanguard/pkg/model"
	"zanguard/pkg/schema"
	"zanguard/pkg/storage/memory"
	"zanguard/pkg/tenant"
)

const conditionSchema = `
version: "1.0"
types:
  user:
    attributes:
      clearance_level: int
  document:
    attributes:
      classification: string
    relations:
      owner:
        types: [user]
      viewer:
        types: [user]
    permissions:
      view:
        union:
          - viewer
          - owner
        condition: "object.classification != \"restricted\" || subject.clearance_level >= 4"
      delete:
        resolve: owner
        condition: "subject.clearance_level >= 3"
`

func setupConditionEngine(t *testing.T) (*Engine, *memory.Store, context.Context) {
	t.Helper()
	store := memory.New()
	ctx := context.Background()

	mgr := tenant.NewManager(store)
	_, _ = mgr.Create(ctx, "cond-tenant", "Cond Test", model.SchemaOwn)
	_ = mgr.Activate(ctx, "cond-tenant")
	tenantCtx, _ := tenant.BuildContext(ctx, store, "cond-tenant")

	data := []byte(conditionSchema)
	raw, err := schema.Parse(data)
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	cs, err := schema.Compile(raw, data)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	eng := New(store, DefaultConfig())
	eng.LoadSchema("cond-tenant", cs)

	return eng, store, tenantCtx
}

func TestConditionAllow(t *testing.T) {
	eng, store, ctx := setupConditionEngine(t)

	// user:alice is a viewer
	_ = store.WriteTuple(ctx, &model.RelationTuple{
		ObjectType: "document", ObjectID: "public-doc",
		Relation: "viewer", SubjectType: "user", SubjectID: "alice",
	})

	// Set document classification to "internal" (not restricted)
	_ = store.SetObjectAttributes(ctx, "document", "public-doc", map[string]any{
		"classification": "internal",
	})

	result, err := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "public-doc",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "alice",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true for non-restricted document")
	}
}

func TestConditionDeny(t *testing.T) {
	eng, store, ctx := setupConditionEngine(t)

	// user:bob is a viewer but document is restricted and bob has low clearance
	_ = store.WriteTuple(ctx, &model.RelationTuple{
		ObjectType: "document", ObjectID: "secret-doc",
		Relation: "viewer", SubjectType: "user", SubjectID: "bob",
	})
	_ = store.SetObjectAttributes(ctx, "document", "secret-doc", map[string]any{
		"classification": "restricted",
	})
	_ = store.SetSubjectAttributes(ctx, "user", "bob", map[string]any{
		"clearance_level": 2,
	})

	result, err := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "secret-doc",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "bob",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Allowed {
		t.Error("expected allowed=false: restricted doc and low clearance")
	}
}

func TestConditionHighClearanceOverride(t *testing.T) {
	eng, store, ctx := setupConditionEngine(t)

	// charlie has clearance_level 4 — can view restricted
	_ = store.WriteTuple(ctx, &model.RelationTuple{
		ObjectType: "document", ObjectID: "top-secret",
		Relation: "viewer", SubjectType: "user", SubjectID: "charlie",
	})
	_ = store.SetObjectAttributes(ctx, "document", "top-secret", map[string]any{
		"classification": "restricted",
	})
	_ = store.SetSubjectAttributes(ctx, "user", "charlie", map[string]any{
		"clearance_level": 4,
	})

	result, err := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "top-secret",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "charlie",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true: high clearance overrides restriction")
	}
}
