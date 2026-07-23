package order

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the order routes onto the /api group. All routes require auth.
//
//	POST /api/orders             buyer creates a trade request
//	GET  /api/orders?role=...    list current user's orders (role=buyer|seller)
//	PUT  /api/orders/:id/confirm seller confirms (PENDING -> RESERVED)
//	PUT  /api/orders/:id/finish  buyer/seller completes (RESERVED -> COMPLETED)
//	PUT  /api/orders/:id/cancel  buyer/seller cancels (-> CANCELLED)
func Register(api *gin.RouterGroup, svc *Service) {
	auth := api.Group("")
	auth.Use(httpx.RequireAuth())
	{
		auth.POST("/orders", func(c *gin.Context) {
			var req CreateReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			item, err := svc.Create(c.Request.Context(), httpx.RequireUserID(c), req)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		auth.GET("/orders", func(c *gin.Context) {
			items, err := svc.MyOrders(c.Request.Context(), httpx.RequireUserID(c), c.Query("role"))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(items))
		})

		auth.PUT("/orders/:id/confirm", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			item, err := svc.Confirm(c.Request.Context(), id, httpx.RequireUserID(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		auth.PUT("/orders/:id/finish", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			item, err := svc.Finish(c.Request.Context(), id, httpx.RequireUserID(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		auth.PUT("/orders/:id/cancel", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			item, err := svc.Cancel(c.Request.Context(), id, httpx.RequireUserID(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})
	}
}

// dispatch handles BizError -> Abort. Returns true if the request was handled.
func dispatch(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var be httpx.BizError
	if errors.As(err, &be) {
		httpx.Abort(c, be)
		return true
	}
	panic(err)
}

func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Abort(c, httpx.Biz("参数错误"))
		return 0, false
	}
	return id, true
}
