package upload

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the upload route onto the /api group.
// Requires authentication.
//
//	POST /api/upload/image   upload an image (multipart form field: "file")
//	                         returns {"url": "/uploads/yyyy-MM-dd/hexuuid.ext"}
func Register(api *gin.RouterGroup, svc *Service) {
	ul := api.Group("/upload")
	ul.Use(httpx.RequireAuth())
	{
		ul.POST("/image", func(c *gin.Context) {
			file, err := c.FormFile("file")
			if err != nil || file == nil {
				httpx.Abort(c, httpx.Biz("文件不能为空"))
				return
			}
			url, err := svc.Upload(file)
			var be httpx.BizError
			if errors.As(err, &be) {
				httpx.Abort(c, be)
				return
			}
			if err != nil {
				panic(err)
			}
			c.JSON(200, httpx.OK(map[string]string{"url": url}))
		})
	}
}
