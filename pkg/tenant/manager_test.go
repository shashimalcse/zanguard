package tenant

import (
	"context"
	"strings"
	"testing"

	"zanguard/pkg/model"
	"zanguard/pkg/storage/memory"
)

func setupManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager(memory.New())
}

func TestValidateTenantID(t *testing.T) {
	valid := []string{"acme", "my-org", "org-123", "a0", "abc-def-ghi"}
	for _, id := range valid {
		if err := ValidateTenantID(id); err != nil {
			t.Errorf("ValidateTenantID(%q) = %v, want nil", id, err)
		}
	}

	invalid := []string{"", "A", "ACME", "a", "-org", "org-", "has space", strings.Repeat("x", 129)}
	for _, id := range invalid {
		if err := ValidateTenantID(id); err == nil {
			t.Errorf("ValidateTenantID(%q) = nil, want error", id)
		}
	}
}

func TestCreateAndActivate(t *testing.T) {
	mgr := setupManager(t)
	ctx := context.Background()

	t0, err := mgr.Create(ctx, "my-tenant", "My Tenant", model.SchemaOwn)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if t0.Status != model.TenantPending {
		t.Errorf("expected pending, got %s", t0.Status)
	}

	if err := mgr.Activate(ctx, "my-tenant"); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	t1, _ := mgr.Get(ctx, "my-tenant")
	if t1.Status != model.TenantActive {
		t.Errorf("expected active, got %s", t1.Status)
	}
}

func TestStateMachine(t *testing.T) {
	mgr := setupManager(t)
	ctx := context.Background()

	_, _ = mgr.Create(ctx, "sm-test", "SM Test", model.SchemaOwn)
	_ = mgr.Activate(ctx, "sm-test")
	_ = mgr.Suspend(ctx, "sm-test")

	// Reactivate from suspended
	if err := mgr.Activate(ctx, "sm-test"); err != nil {
		t.Fatalf("Activate from suspended: %v", err)
	}

	// Delete
	if err := mgr.Delete(ctx, "sm-test"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Cannot activate deleted
	if err := mgr.Activate(ctx, "sm-test"); err == nil {
		t.Error("expected error activating deleted tenant")
	}
}

func TestBuildContext(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	_ = store.CreateTenant(ctx, &model.Tenant{ID: "ctx-test", Status: model.TenantActive})

	tenantCtx, err := BuildContext(ctx, store, "ctx-test")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	tc := model.TenantFromContext(tenantCtx)
	if tc == nil {
		t.Fatal("no tenant context found")
	}
	if tc.TenantID != "ctx-test" {
		t.Errorf("expected ctx-test, got %s", tc.TenantID)
	}
}
