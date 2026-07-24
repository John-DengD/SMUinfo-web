package announcement

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the announcement routes onto the /api group.
//
//	GET /api/announcements/active  (public)
func Register(api *gin.RouterGroup, svc *Service) {
	api.GET("/announcements/active", func(c *gin.Context) {
		item, err := svc.Active(c.Request.Context())
		var be httpx.BizError
		if errors.As(err, &be) {
			httpx.Abort(c, be)
			return
		}
		if err != nil {
			panic(err)
		}
		c.JSON(200, httpx.OK(item))
	})
}
