package lostfound

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the lost-and-found routes onto the /api group.
//
//	GET    /api/lost-found        (public)  list with type + keyword + pagination
//	GET    /api/lost-found/:id    (public)  detail (increments view_count)
//	POST   /api/lost-found        (auth)    create with images (transactional)
//	DELETE /api/lost-found/:id    (auth)    close (owner or admin) → status CLOSED
func Register(api *gin.RouterGroup, svc *Service) {
	api.GET("/lost-found", func(c *gin.Context) {
		q := parseListQuery(c)
		page, err := svc.List(c.Request.Context(), q)
		if dispatch(c, err) {
			return
		}
		c.JSON(200, httpx.OK(page))
	})

	api.GET("/lost-found/:id", func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			return
		}
		item, err := svc.Detail(c.Request.Context(), id)
		if dispatch(c, err) {
			return
		}
		c.JSON(200, httpx.OK(item))
	})

	auth := api.Group("")
	auth.Use(httpx.RequireAuth())
	{
		auth.POST("/lost-found", func(c *gin.Context) {
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

		auth.DELETE("/lost-found/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			uid := httpx.UserID(c)
			if uid == 0 {
				httpx.Abort(c, httpx.NewBiz(401, "未登录"))
				return
			}
			err := svc.Close(c.Request.Context(), id, uid, isAdmin(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})
	}
}

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

func isAdmin(c *gin.Context) bool {
	return strings.EqualFold(httpx.Role(c), "ADMIN")
}

func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Abort(c, httpx.Biz("参数错误"))
		return 0, false
	}
	return id, true
}

func parseListQuery(c *gin.Context) ListQuery {
	return ListQuery{
		Type:    strPtr(c.Query("type")),
		Keyword: strPtr(c.Query("keyword")),
		Page:    int32Ptr(c.Query("page")),
		Size:    int32Ptr(c.Query("size")),
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func int32Ptr(s string) *int32 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil
	}
	x := int32(v)
	return &x
}
