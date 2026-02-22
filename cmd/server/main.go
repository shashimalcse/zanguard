package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"zanguard/pkg/api"
	"zanguard/pkg/engine"
	"zanguard/pkg/model"
	"zanguard/pkg/storage"
	"zanguard/pkg/storage/postgres"
	"zanguard/pkg/tenant"
)

const bootstrapTenantID = "super"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	addr := os.Getenv("ZANGUARD_ADDR")
	if addr == "" {
		addr = ":1997"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	maxConns := int32(10)
	if raw := os.Getenv("ZANGUARD_DB_MAX_CONNS"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n <= 0 {
			log.Error("invalid ZANGUARD_DB_MAX_CONNS", "value", raw)
			os.Exit(1)
		}
		maxConns = int32(n)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := postgres.New(ctx, dsn, postgres.WithMaxConns(maxConns))
	if err != nil {
		log.Error("failed to initialize postgres store", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	mgr := tenant.NewManager(store)
	if err := ensureBootstrapTenant(ctx, store, mgr, log); err != nil {
		log.Error("failed to ensure bootstrap tenant", "tenant_id", bootstrapTenantID, "err", err)
		os.Exit(1)
	}

	eng := engine.New(store, engine.DefaultConfig())

	srv := api.NewServer(store, eng, mgr, log)

	if err := srv.Start(addr); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}

func ensureBootstrapTenant(ctx context.Context, store storage.TupleStore, mgr *tenant.Manager, log *slog.Logger) error {
	t, err := mgr.Get(ctx, bootstrapTenantID)
	if err != nil {
		if errors.Is(err, storage.ErrTenantNotFound) {
			if _, err := mgr.Create(ctx, bootstrapTenantID, "Super Tenant", model.SchemaOwn); err != nil {
				return fmt.Errorf("create bootstrap tenant: %w", err)
			}
			if err := mgr.Activate(ctx, bootstrapTenantID); err != nil {
				return fmt.Errorf("activate bootstrap tenant: %w", err)
			}
			log.Info("bootstrap tenant created and activated", "tenant_id", bootstrapTenantID)
			return nil
		}
		return fmt.Errorf("get bootstrap tenant: %w", err)
	}

	if t.Status == model.TenantActive {
		return nil
	}

	prevStatus := t.Status
	t.Status = model.TenantActive
	if err := store.UpdateTenant(ctx, t); err != nil {
		return fmt.Errorf("set bootstrap tenant active from %s: %w", prevStatus, err)
	}
	log.Info("bootstrap tenant set active", "tenant_id", bootstrapTenantID, "previous_status", string(prevStatus))
	return nil
}
