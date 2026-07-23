package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/John-DengD/smu-deal/server/internal/app"
	"github.com/John-DengD/smu-deal/server/internal/config"
	"github.com/John-DengD/smu-deal/server/internal/db"
)

func main() {
	cfg := config.Load()

	mdir := envOr("MIGRATIONS_DIR", "internal/db/migrations")
	if err := db.RunMigrations(cfg.DBURL, mdir); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(context.Background(), cfg.DBURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	_ = os.MkdirAll(cfg.UploadDir, 0o755)

	r := app.NewRouter(cfg, pool)

	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("run", "err", err)
		os.Exit(1)
	}
}

func envOr(v, def string) string {
	if x := os.Getenv(v); x != "" {
		return x
	}
	return def
}
