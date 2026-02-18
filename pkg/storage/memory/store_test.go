package memory

import (
	"context"
	"sync"
	"testing"

	"zanguard/pkg/model"
	"zanguard/pkg/storage"
)

func tenantCtx(tenantID string) context.Context {
	tc := &model.TenantContext{TenantID: tenantID}
	return model.WithTenantContext(context.Background(), tc)
}

func setupStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	s := New()
	ctx := context.Background()

	// Create and activate tenant
	err := s.CreateTenant(ctx, &model.Tenant{
		ID:     "acme",
		Status: model.TenantActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	return s, tenantCtx("acme")
}

func TestCreateAndGetTenant(t *testing.T) {
	s := New()
	ctx := context.Background()

	err := s.CreateTenant(ctx, &model.Tenant{
		ID:          "test-tenant",
		DisplayName: "Test",
		Status:      model.TenantActive,
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	got, err := s.GetTenant(ctx, "test-tenant")
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if got.ID != "test-tenant" {
		t.Errorf("got ID %q, want %q", got.ID, "test-tenant")
	}
}

func TestGetTenantNotFound(t *testing.T) {
	s := New()
	_, err := s.GetTenant(context.Background(), "missing")
	if err != storage.ErrTenantNotFound {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}

func TestWriteAndCheckDirect(t *testing.T) {
	s, ctx := setupStore(t)

	err := s.WriteTuple(ctx, &model.RelationTuple{
		ObjectType:  "document",
		ObjectID:    "readme",
		Relation:    "viewer",
		SubjectType: "user",
		SubjectID:   "thilina",
	})
	if err != nil {
		t.Fatalf("WriteTuple: %v", err)
	}

	ok, err := s.CheckDirect(ctx, "document", "readme", "viewer", "user", "thilina")
	if err != nil {
		t.Fatalf("CheckDirect: %v", err)
	}
	if !ok {
		t.Error("expected direct check to return true")
	}
}

func TestDuplicateTuple(t *testing.T) {
	s, ctx := setupStore(t)

	tuple := &model.RelationTuple{
		ObjectType:  "document",
		ObjectID:    "readme",
		Relation:    "viewer",
		SubjectType: "user",
		SubjectID:   "alice",
	}
	if err := s.WriteTuple(ctx, tuple); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteTuple(ctx, tuple); err != storage.ErrDuplicateTuple {
		t.Errorf("expected ErrDuplicateTuple, got %v", err)
	}
}

func TestTenantIsolation(t *testing.T) {
	s := New()
	ctx := context.Background()

	for _, id := range []string{"tenant-a", "tenant-b"} {
		if err := s.CreateTenant(ctx, &model.Tenant{ID: id, Status: model.TenantActive}); err != nil {
			t.Fatal(err)
		}
	}

	ctxA := tenantCtx("tenant-a")
	ctxB := tenantCtx("tenant-b")

	// Write tuple to tenant A
	_ = s.WriteTuple(ctxA, &model.RelationTuple{
		ObjectType:  "doc", ObjectID: "1", Relation: "viewer",
		SubjectType: "user", SubjectID: "alice",
	})

	// Should not be visible from tenant B
	ok, _ := s.CheckDirect(ctxB, "doc", "1", "viewer", "user", "alice")
	if ok {
		t.Error("tenant isolation breach: tenant-b can see tenant-a's tuple")
	}
}

func TestPurgeTenantData(t *testing.T) {
	s, ctx := setupStore(t)

	for i := 0; i < 5; i++ {
		_ = s.WriteTuple(ctx, &model.RelationTuple{
			ObjectType:  "doc",
			ObjectID:    string(rune('a' + i)),
			Relation:    "viewer",
			SubjectType: "user",
			SubjectID:   "alice",
		})
	}

	count, _ := s.CountTuples(ctx)
	if count != 5 {
		t.Fatalf("expected 5 tuples, got %d", count)
	}

	if err := s.PurgeTenantData(ctx); err != nil {
		t.Fatal(err)
	}

	count, _ = s.CountTuples(ctx)
	if count != 0 {
		t.Errorf("expected 0 tuples after purge, got %d", count)
	}
}

func TestChangelogSequence(t *testing.T) {
	s, ctx := setupStore(t)

	for i := 0; i < 3; i++ {
		_ = s.AppendChangelog(ctx, &model.ChangelogEntry{
			Operation: model.ChangeOpInsert,
		})
	}

	seq, err := s.LatestSequence(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 3 {
		t.Errorf("expected sequence 3, got %d", seq)
	}

	entries, err := s.ReadChangelog(ctx, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestConcurrentWrites(t *testing.T) {
	s, ctx := setupStore(t)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = s.WriteTuple(ctx, &model.RelationTuple{
				ObjectType:  "doc",
				ObjectID:    string(rune('a' + n%26)),
				Relation:    "viewer",
				SubjectType: "user",
				SubjectID:   string(rune('A' + n%26)),
			})
		}(i)
	}
	wg.Wait()
}

func TestDeleteTuple(t *testing.T) {
	s, ctx := setupStore(t)

	tuple := &model.RelationTuple{
		ObjectType:  "document", ObjectID: "doc1",
		Relation: "viewer", SubjectType: "user", SubjectID: "bob",
	}
	_ = s.WriteTuple(ctx, tuple)

	if err := s.DeleteTuple(ctx, tuple); err != nil {
		t.Fatalf("DeleteTuple: %v", err)
	}

	ok, _ := s.CheckDirect(ctx, "document", "doc1", "viewer", "user", "bob")
	if ok {
		t.Error("expected tuple to be deleted")
	}
}

func TestSuspendedTenantNoWrites(t *testing.T) {
	s := New()
	ctx := context.Background()
	_ = s.CreateTenant(ctx, &model.Tenant{ID: "suspended-tenant", Status: model.TenantSuspended})
	tCtx := tenantCtx("suspended-tenant")

	err := s.WriteTuple(tCtx, &model.RelationTuple{
		ObjectType: "doc", ObjectID: "x", Relation: "viewer", SubjectType: "user", SubjectID: "y",
	})
	if err != storage.ErrTenantSuspended {
		t.Errorf("expected ErrTenantSuspended, got %v", err)
	}
}
