package main

import (
	"log/slog"
	"os"

	"zanguard/pkg/api"
	"zanguard/pkg/engine"
	"zanguard/pkg/storage/memory"
	"zanguard/pkg/tenant"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	addr := os.Getenv("ZANGUARD_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	store := memory.New()
	mgr := tenant.NewManager(store)
	eng := engine.New(store, engine.DefaultConfig())

	srv := api.NewServer(store, eng, mgr, log)

	if err := srv.Start(addr); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}
