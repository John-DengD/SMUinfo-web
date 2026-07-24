package favorite

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the favorite routes onto the /api group.
// All routes require authentication.
//
//	GET    /api/favorites              list current user's favorited products (product items shape)
//	POST   /api/favorites/:productId   add favorite (idempotent)
//	DELETE /api/favorites/:productId   remove favorite
func Register(api *gin.RouterGroup, svc *Service) {
	fav := api.Group("/favorites")
	fav.Use(httpx.RequireAuth())
	{
		fav.GET("", func(c *gin.Context) {
			page, err := svc.MyFavorites(c.Request.Context(), httpx.RequireUserID(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(page))
		})

		fav.POST("/:productId", func(c *gin.Context) {
			pid, ok := pathID(c)
			if !ok {
				return
			}
			err := svc.Add(c.Request.Context(), httpx.RequireUserID(c), pid)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})

		fav.DELETE("/:productId", func(c *gin.Context) {
			pid, ok := pathID(c)
			if !ok {
				return
			}
			err := svc.Remove(c.Request.Context(), httpx.RequireUserID(c), pid)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(nil))
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
	id, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil {
		httpx.Abort(c, httpx.Biz("参数错误"))
		return 0, false
	}
	return id, true
}
