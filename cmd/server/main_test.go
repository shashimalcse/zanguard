package main

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"zanguard/pkg/model"
	"zanguard/pkg/storage/memory"
	"zanguard/pkg/tenant"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEnsureBootstrapTenantCreatesAndActivates(t *testing.T) {
	store := memory.New()
	mgr := tenant.NewManager(store)

	if err := ensureBootstrapTenant(context.Background(), store, mgr, testLogger()); err != nil {
		t.Fatalf("ensureBootstrapTenant: %v", err)
	}

	got, err := store.GetTenant(context.Background(), bootstrapTenantID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if got.Status != model.TenantActive {
		t.Fatalf("expected status active, got %s", got.Status)
	}
}

func TestEnsureBootstrapTenantActivatesExistingTenant(t *testing.T) {
	store := memory.New()
	mgr := tenant.NewManager(store)
	ctx := context.Background()

	if err := store.CreateTenant(ctx, &model.Tenant{
		ID:         bootstrapTenantID,
		Status:     model.TenantPending,
		SchemaMode: model.SchemaOwn,
	}); err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	if err := ensureBootstrapTenant(ctx, store, mgr, testLogger()); err != nil {
		t.Fatalf("ensureBootstrapTenant: %v", err)
	}

	got, err := store.GetTenant(ctx, bootstrapTenantID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if got.Status != model.TenantActive {
		t.Fatalf("expected status active, got %s", got.Status)
	}
}

func TestEnsureBootstrapTenantRevivesDeletedTenant(t *testing.T) {
	store := memory.New()
	mgr := tenant.NewManager(store)
	ctx := context.Background()

	if err := store.CreateTenant(ctx, &model.Tenant{
		ID:         bootstrapTenantID,
		Status:     model.TenantDeleted,
		SchemaMode: model.SchemaOwn,
	}); err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	if err := ensureBootstrapTenant(ctx, store, mgr, testLogger()); err != nil {
		t.Fatalf("ensureBootstrapTenant: %v", err)
	}

	got, err := store.GetTenant(ctx, bootstrapTenantID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if got.Status != model.TenantActive {
		t.Fatalf("expected status active, got %s", got.Status)
	}
}
