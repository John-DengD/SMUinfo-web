package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/announcement"
	"github.com/John-DengD/smu-deal/server/internal/auth"
	"github.com/John-DengD/smu-deal/server/internal/category"
	"github.com/John-DengD/smu-deal/server/internal/config"
	"github.com/John-DengD/smu-deal/server/internal/db"
	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/favorite"
	"github.com/John-DengD/smu-deal/server/internal/feedback"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
	"github.com/John-DengD/smu-deal/server/internal/lostfound"
	"github.com/John-DengD/smu-deal/server/internal/message"
	"github.com/John-DengD/smu-deal/server/internal/order"
	"github.com/John-DengD/smu-deal/server/internal/product"
	"github.com/John-DengD/smu-deal/server/internal/report"
	"github.com/John-DengD/smu-deal/server/internal/transit"
	"github.com/John-DengD/smu-deal/server/internal/upload"
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
	r.MaxMultipartMemory = cfg.MaxFileSize + 512
	r.Use(httpx.CORS(cfg.AllowedOrigins), httpx.AuthParse(jwt), httpx.Recovery())
	r.Static(cfg.URLPrefix, cfg.UploadDir)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, httpx.OK("ok")) })

	api := r.Group("/api")
	auth.Register(api, auth.NewService(q, jwt))
	category.Register(api, category.NewService(q))
	product.Register(api, product.NewService(q, pool))
	report.Register(api, report.NewService(q))
	feedback.Register(api, feedback.NewService(q))
	announcement.Register(api, announcement.NewService(q))
	favorite.Register(api, favorite.NewService(q))
	order.Register(api, order.NewService(q, pool))
	message.Register(api, message.NewService(q))
	lostfound.Register(api, lostfound.NewService(q, pool))
	transit.Register(api, transit.NewService(q))
	upload.Register(api, upload.NewService(cfg.UploadDir, cfg.URLPrefix, cfg.MaxFileSize))

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
