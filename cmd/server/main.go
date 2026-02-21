package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"zanguard/pkg/api"
	"zanguard/pkg/engine"
	"zanguard/pkg/storage/postgres"
	"zanguard/pkg/tenant"
)

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
	eng := engine.New(store, engine.DefaultConfig())

	srv := api.NewServer(store, eng, mgr, log)

	if err := srv.Start(addr); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}
