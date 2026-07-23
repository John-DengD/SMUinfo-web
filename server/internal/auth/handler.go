package auth

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the auth routes onto the /api group.
//   POST /api/auth/register      (public)
//   POST /api/auth/login         (public)
//   GET  /api/users/me           (auth required)
//   PUT  /api/users/me           (auth required)
func Register(api *gin.RouterGroup, svc *Service) {
	api.POST("/auth/register", func(c *gin.Context) {
		var req RegisterReq
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Abort(c, httpx.Biz("参数错误"))
			return
		}
		info, err := svc.Register(c.Request.Context(), req)
		var be httpx.BizError
		if errors.As(err, &be) {
			httpx.Abort(c, be)
			return
		}
		if err != nil {
			panic(err)
		}
		c.JSON(200, httpx.OK(info))
	})

	api.POST("/auth/login", func(c *gin.Context) {
		var req LoginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Abort(c, httpx.Biz("参数错误"))
			return
		}
		resp, err := svc.Login(c.Request.Context(), req)
		var be httpx.BizError
		if errors.As(err, &be) {
			httpx.Abort(c, be)
			return
		}
		if err != nil {
			panic(err)
		}
		c.JSON(200, httpx.OK(resp))
	})

	users := api.Group("/users")
	users.Use(httpx.RequireAuth())
	{
		users.GET("/me", func(c *gin.Context) {
			info, err := svc.GetMe(c.Request.Context(), httpx.RequireUserID(c))
			var be httpx.BizError
			if errors.As(err, &be) {
				httpx.Abort(c, be)
				return
			}
			if err != nil {
				panic(err)
			}
			c.JSON(200, httpx.OK(info))
		})

		users.PUT("/me", func(c *gin.Context) {
			var req UserInfo
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			info, err := svc.UpdateMe(c.Request.Context(), httpx.RequireUserID(c), req)
			var be httpx.BizError
			if errors.As(err, &be) {
				httpx.Abort(c, be)
				return
			}
			if err != nil {
				panic(err)
			}
			c.JSON(200, httpx.OK(info))
		})
	}
}
