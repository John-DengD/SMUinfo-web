// Package app assembles the HTTP router shared by the production binary and
// the end-to-end test suite. NewRouter performs exactly the same middleware and
// route wiring that used to live inline in cmd/smudeal/main.go.
package app

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/John-DengD/smu-deal/server/internal/admin"
	"github.com/John-DengD/smu-deal/server/internal/announcement"
	"github.com/John-DengD/smu-deal/server/internal/auth"
	"github.com/John-DengD/smu-deal/server/internal/category"
	"github.com/John-DengD/smu-deal/server/internal/config"
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

// NewRouter builds the gin engine with all middleware, the static uploads
// handler, the health check, and every domain's routes registered under /api.
func NewRouter(cfg config.Config, pool *pgxpool.Pool) *gin.Engine {
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
	productSvc := product.NewService(q, pool)
	product.Register(api, productSvc)
	report.Register(api, report.NewService(q))
	feedback.Register(api, feedback.NewService(q))
	announcement.Register(api, announcement.NewService(q))
	favorite.Register(api, favorite.NewService(q))
	order.Register(api, order.NewService(q, pool))
	message.Register(api, message.NewService(q))
	lostfound.Register(api, lostfound.NewService(q, pool))
	transit.Register(api, transit.NewService(q))
	upload.Register(api, upload.NewService(cfg.UploadDir, cfg.URLPrefix, cfg.MaxFileSize))
	admin.Register(api, admin.NewService(q, productSvc, pool))

	return r
}
