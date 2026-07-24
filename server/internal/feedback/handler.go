package feedback

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the feedback routes onto the /api group.
//
//	POST /api/feedback       (auth required)
//	GET  /api/feedback/mine  (auth required)
func Register(api *gin.RouterGroup, svc *Service) {
	fb := api.Group("/feedback")
	fb.Use(httpx.RequireAuth())
	{
		fb.POST("", func(c *gin.Context) {
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

		fb.GET("/mine", func(c *gin.Context) {
			items, err := svc.ListMine(c.Request.Context(), httpx.RequireUserID(c))
			var be httpx.BizError
			if errors.As(err, &be) {
				httpx.Abort(c, be)
				return
			}
			if err != nil {
				panic(err)
			}
			c.JSON(200, httpx.OK(items))
		})
	}
}
