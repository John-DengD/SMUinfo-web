package report

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the report routes onto the /api group.
//
//	POST /api/reports  (auth required)
func Register(api *gin.RouterGroup, svc *Service) {
	reports := api.Group("/reports")
	reports.Use(httpx.RequireAuth())
	{
		reports.POST("", func(c *gin.Context) {
			var req CreateReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			err := svc.Create(c.Request.Context(), httpx.RequireUserID(c), req)
			var be httpx.BizError
			if errors.As(err, &be) {
				httpx.Abort(c, be)
				return
			}
			if err != nil {
				panic(err)
			}
			c.JSON(200, httpx.OK(nil))
		})
	}
}
