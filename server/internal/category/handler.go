package category

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the category routes onto the /api group.
//
//	GET /api/categories  (public)
func Register(api *gin.RouterGroup, svc *Service) {
	api.GET("/categories", func(c *gin.Context) {
		items, err := svc.List(c.Request.Context())
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
