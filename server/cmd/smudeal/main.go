package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/auth"
	"github.com/John-DengD/smu-deal/server/internal/config"
	"github.com/John-DengD/smu-deal/server/internal/db"
	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
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
	jwt := httpx.NewJWT(cfg.JWTSecret, cfg.JWTExpireHours)
	q := gen.New(pool)

	r := gin.New()
	r.Use(httpx.CORS(cfg.AllowedOrigins), httpx.AuthParse(jwt), httpx.Recovery())
	r.Static(cfg.URLPrefix, cfg.UploadDir)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, httpx.OK("ok")) })

	api := r.Group("/api")
	auth.Register(api, auth.NewService(q, jwt))

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
