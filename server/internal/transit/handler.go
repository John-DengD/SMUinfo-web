package transit

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the transit routes onto the /api group.
//
//	GET /api/transit/next  (public)  next upcoming shuttle departures
func Register(api *gin.RouterGroup, svc *Service) {
	api.GET("/transit/next", func(c *gin.Context) {
		resp, err := svc.Next(c.Request.Context(),
			c.Query("line"),
			c.Query("station"),
			c.Query("direction"))
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
}
