package engine

import (
	"context"
	"testing"

	"zanguard/pkg/model"
	"zanguard/pkg/schema"
	"zanguard/pkg/storage/memory"
	"zanguard/pkg/tenant"
)

const testSchema = `
version: "1.0"
types:
  user: {}
  group:
    relations:
      member:
        types: [user]
    permissions:
      has_member:
        resolve: member
  folder:
    relations:
      owner:
        types: [user]
      viewer:
        types: [user, group#member]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - owner
          - parent->view
      edit:
        resolve: owner
  document:
    relations:
      owner:
        types: [user]
      editor:
        types: [user, group#member]
      viewer:
        types: [user, group#member]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - editor
          - owner
          - parent->view
      edit:
        union:
          - editor
          - owner
      delete:
        resolve: owner
`

func setupEngine(t *testing.T) (*Engine, context.Context) {
	t.Helper()
	store := memory.New()
	ctx := context.Background()

	mgr := tenant.NewManager(store)
	_, _ = mgr.Create(ctx, "test-tenant", "Test", model.SchemaOwn)
	_ = mgr.Activate(ctx, "test-tenant")

	tenantCtx, _ := tenant.BuildContext(ctx, store, "test-tenant")

	data := []byte(testSchema)
	raw, err := schema.Parse(data)
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	cs, err := schema.Compile(raw, data)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	eng := New(store, DefaultConfig())
	eng.LoadSchema("test-tenant", cs)

	return eng, tenantCtx
}

func writeTuple(t *testing.T, ctx context.Context, store *memory.Store, objectType, objectID, relation, subjectType, subjectID string) {
	t.Helper()
	err := store.WriteTuple(ctx, &model.RelationTuple{
		ObjectType:  objectType,
		ObjectID:    objectID,
		Relation:    relation,
		SubjectType: subjectType,
		SubjectID:   subjectID,
	})
	if err != nil {
		t.Fatalf("WriteTuple: %v", err)
	}
}

func TestCheckDirect(t *testing.T) {
	eng, ctx := setupEngine(t)
	store := eng.store.(*memory.Store)

	writeTuple(t, ctx, store, "document", "readme", "viewer", "user", "alice")

	result, err := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "readme",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "alice",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true for direct viewer")
	}
}

func TestCheckDeny(t *testing.T) {
	eng, ctx := setupEngine(t)

	result, err := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "readme",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "nobody",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Allowed {
		t.Error("expected allowed=false for unknown user")
	}
}

func TestCheckUserset(t *testing.T) {
	eng, ctx := setupEngine(t)
	store := eng.store.(*memory.Store)

	// group:engineering#member → user:bob
	err := store.WriteTuple(ctx, &model.RelationTuple{
		ObjectType:  "group",
		ObjectID:    "engineering",
		Relation:    "member",
		SubjectType: "user",
		SubjectID:   "bob",
	})
	if err != nil {
		t.Fatal(err)
	}

	// document:spec#viewer → group:engineering#member
	err = store.WriteTuple(ctx, &model.RelationTuple{
		ObjectType:      "document",
		ObjectID:        "spec",
		Relation:        "viewer",
		SubjectType:     "group",
		SubjectID:       "engineering",
		SubjectRelation: "member",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "spec",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "bob",
	})
	if err != nil {
		t.Fatalf("Check userset: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true via group membership")
	}
}

func TestCheckArrow(t *testing.T) {
	eng, ctx := setupEngine(t)
	store := eng.store.(*memory.Store)

	// user:carol owns folder:root
	writeTuple(t, ctx, store, "folder", "root", "owner", "user", "carol")

	// document:design#parent → folder:root
	writeTuple(t, ctx, store, "document", "design", "parent", "folder", "root")

	// carol should be able to view document:design via folder:root ownership
	result, err := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "design",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "carol",
	})
	if err != nil {
		t.Fatalf("Check arrow: %v", err)
	}
	if !result.Allowed {
		t.Error("expected allowed=true via parent->view arrow")
	}
}

func TestCheckOwnerCanEdit(t *testing.T) {
	eng, ctx := setupEngine(t)
	store := eng.store.(*memory.Store)

	writeTuple(t, ctx, store, "document", "report", "owner", "user", "dave")

	result, _ := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "report",
		Permission:  "edit",
		SubjectType: "user",
		SubjectID:   "dave",
	})
	if !result.Allowed {
		t.Error("owner should be able to edit")
	}

	result2, _ := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "report",
		Permission:  "delete",
		SubjectType: "user",
		SubjectID:   "dave",
	})
	if !result2.Allowed {
		t.Error("owner should be able to delete")
	}
}

func TestTenantIsolationEngine(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	mgr := tenant.NewManager(store)

	for _, id := range []string{"tenant-x", "tenant-y"} {
		_, _ = mgr.Create(ctx, id, id, model.SchemaOwn)
		_ = mgr.Activate(ctx, id)
	}

	data := []byte(testSchema)
	raw, _ := schema.Parse(data)
	cs, _ := schema.Compile(raw, data)

	eng := New(store, DefaultConfig())
	eng.LoadSchema("tenant-x", cs)
	eng.LoadSchema("tenant-y", cs)

	ctxX, _ := tenant.BuildContext(ctx, store, "tenant-x")
	ctxY, _ := tenant.BuildContext(ctx, store, "tenant-y")

	// Write tuple only in tenant-x
	_ = store.WriteTuple(ctxX, &model.RelationTuple{
		ObjectType: "document", ObjectID: "secret",
		Relation: "viewer", SubjectType: "user", SubjectID: "eve",
	})

	// Check in tenant-x: should be allowed
	r1, _ := eng.Check(ctxX, &CheckRequest{
		ObjectType: "document", ObjectID: "secret",
		Permission: "view", SubjectType: "user", SubjectID: "eve",
	})
	if !r1.Allowed {
		t.Error("expected allowed in tenant-x")
	}

	// Check in tenant-y: should be denied (isolation)
	r2, _ := eng.Check(ctxY, &CheckRequest{
		ObjectType: "document", ObjectID: "secret",
		Permission: "view", SubjectType: "user", SubjectID: "eve",
	})
	if r2.Allowed {
		t.Error("tenant isolation breach: tenant-y saw tenant-x data")
	}
}

func TestCycleDetection(t *testing.T) {
	// Create a cycle: folder A's parent is folder B, folder B's parent is folder A
	eng, ctx := setupEngine(t)
	store := eng.store.(*memory.Store)

	writeTuple(t, ctx, store, "folder", "a", "parent", "folder", "b")
	writeTuple(t, ctx, store, "folder", "b", "parent", "folder", "a")
	writeTuple(t, ctx, store, "document", "doc", "parent", "folder", "a")

	// Should not hang or panic — cycle detection should break it
	result, err := eng.Check(ctx, &CheckRequest{
		ObjectType:  "document",
		ObjectID:    "doc",
		Permission:  "view",
		SubjectType: "user",
		SubjectID:   "nobody",
	})
	if err != nil {
		t.Logf("cycle detected with error: %v (acceptable)", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}
