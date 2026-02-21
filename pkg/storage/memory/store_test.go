package memory

import (
	"context"
	"sync"
	"testing"
	"time"

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
		ObjectType: "doc", ObjectID: "1", Relation: "viewer",
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
		ObjectType: "document", ObjectID: "doc1",
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

func TestWriteTuplesAtomic(t *testing.T) {
	s, ctx := setupStore(t)

	// Seed one tuple that will be duplicated in batch.
	if err := s.WriteTuple(ctx, &model.RelationTuple{
		ObjectType: "doc", ObjectID: "1", Relation: "viewer", SubjectType: "user", SubjectID: "alice",
	}); err != nil {
		t.Fatalf("seed WriteTuple: %v", err)
	}

	err := s.WriteTuples(ctx, []*model.RelationTuple{
		{ObjectType: "doc", ObjectID: "2", Relation: "viewer", SubjectType: "user", SubjectID: "bob"},
		{ObjectType: "doc", ObjectID: "1", Relation: "viewer", SubjectType: "user", SubjectID: "alice"}, // duplicate
	})
	if err != storage.ErrDuplicateTuple {
		t.Fatalf("expected ErrDuplicateTuple, got %v", err)
	}

	// Atomicity: first tuple in batch must not be persisted.
	ok, _ := s.CheckDirect(ctx, "doc", "2", "viewer", "user", "bob")
	if ok {
		t.Fatal("expected batch write to be atomic; found partially persisted tuple")
	}
}

func TestTupleMutationsAppendChangelog(t *testing.T) {
	s, ctx := setupStore(t)

	tuple := &model.RelationTuple{
		ObjectType: "doc", ObjectID: "1", Relation: "viewer", SubjectType: "user", SubjectID: "alice",
	}
	if err := s.WriteTuple(ctx, tuple); err != nil {
		t.Fatalf("WriteTuple: %v", err)
	}
	if err := s.DeleteTuple(ctx, tuple); err != nil {
		t.Fatalf("DeleteTuple: %v", err)
	}

	entries, err := s.ReadChangelog(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ReadChangelog: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 changelog entries, got %d", len(entries))
	}
	if entries[0].Operation != model.ChangeOpInsert {
		t.Fatalf("expected first changelog op INSERT, got %s", entries[0].Operation)
	}
	if entries[1].Operation != model.ChangeOpDelete {
		t.Fatalf("expected second changelog op DELETE, got %s", entries[1].Operation)
	}
}

func TestGetAttributesNotFound(t *testing.T) {
	s, ctx := setupStore(t)

	if _, err := s.GetObjectAttributes(ctx, "document", "missing"); err != storage.ErrNotFound {
		t.Fatalf("expected object attributes ErrNotFound, got %v", err)
	}
	if _, err := s.GetSubjectAttributes(ctx, "user", "missing"); err != storage.ErrNotFound {
		t.Fatalf("expected subject attributes ErrNotFound, got %v", err)
	}
}

func TestExpiredTuplesHiddenByDefault(t *testing.T) {
	s, ctx := setupStore(t)
	now := time.Now().UTC()

	if err := s.WriteTuple(ctx, &model.RelationTuple{
		ObjectType:  "doc",
		ObjectID:    "active",
		Relation:    "viewer",
		SubjectType: "user",
		SubjectID:   "alice",
		ExpiresAt:   ptrTime(now.Add(10 * time.Minute)),
	}); err != nil {
		t.Fatalf("WriteTuple active: %v", err)
	}
	if err := s.WriteTuple(ctx, &model.RelationTuple{
		ObjectType:  "doc",
		ObjectID:    "expired",
		Relation:    "viewer",
		SubjectType: "user",
		SubjectID:   "alice",
		ExpiresAt:   ptrTime(now.Add(-10 * time.Minute)),
	}); err != nil {
		t.Fatalf("WriteTuple expired: %v", err)
	}

	tuples, err := s.ReadTuples(ctx, &model.TupleFilter{})
	if err != nil {
		t.Fatalf("ReadTuples default: %v", err)
	}
	if len(tuples) != 1 || tuples[0].ObjectID != "active" {
		t.Fatalf("expected only active tuple, got %+v", tuples)
	}

	tuples, err = s.ReadTuples(ctx, &model.TupleFilter{IncludeExpired: true})
	if err != nil {
		t.Fatalf("ReadTuples include_expired: %v", err)
	}
	if len(tuples) != 2 {
		t.Fatalf("expected 2 tuples when include_expired=true, got %d", len(tuples))
	}
}

func TestRegrantExpiredTupleKey(t *testing.T) {
	s, ctx := setupStore(t)
	now := time.Now().UTC()

	tuple := &model.RelationTuple{
		ObjectType:  "tool",
		ObjectID:    "refund",
		Relation:    "executor",
		SubjectType: "agent",
		SubjectID:   "bot-1",
		ExpiresAt:   ptrTime(now.Add(-1 * time.Minute)),
	}
	if err := s.WriteTuple(ctx, tuple); err != nil {
		t.Fatalf("seed expired tuple: %v", err)
	}

	// Same identity should be accepted once previous tuple is expired.
	if err := s.WriteTuple(ctx, &model.RelationTuple{
		ObjectType:  tuple.ObjectType,
		ObjectID:    tuple.ObjectID,
		Relation:    tuple.Relation,
		SubjectType: tuple.SubjectType,
		SubjectID:   tuple.SubjectID,
		ExpiresAt:   ptrTime(now.Add(5 * time.Minute)),
	}); err != nil {
		t.Fatalf("regrant tuple: %v", err)
	}

	ok, err := s.CheckDirect(ctx, "tool", "refund", "executor", "agent", "bot-1")
	if err != nil {
		t.Fatalf("CheckDirect: %v", err)
	}
	if !ok {
		t.Fatal("expected regranted tuple to be active")
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
