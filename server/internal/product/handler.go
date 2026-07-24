package product

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires the product routes onto the /api group.
//
//	GET    /api/products               (public)  list + search + pagination
//	GET    /api/products/:id           (public)  detail (increments view_count)
//	GET    /api/products/:id/comments  (public)  list comments
//	POST   /api/products               (auth)    create
//	PUT    /api/products/:id           (auth)    update (owner or admin)
//	DELETE /api/products/:id           (auth)    soft delete -> status OFFLINE
//	POST   /api/products/:id/comments  (auth)    create comment
func Register(api *gin.RouterGroup, svc *Service) {
	// Public reads. AuthParse still runs globally, so an optional logged-in user
	// is available for the `favorited` flag.
	api.GET("/products", func(c *gin.Context) {
		q := parseListQuery(c)
		var uid *int64
		if id := httpx.UserID(c); id != 0 {
			uid = &id
		}
		page, err := svc.List(c.Request.Context(), q, uid)
		if dispatch(c, err) {
			return
		}
		c.JSON(200, httpx.OK(page))
	})

	api.GET("/products/:id", func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			return
		}
		var uid *int64
		if u := httpx.UserID(c); u != 0 {
			uid = &u
		}
		item, err := svc.Detail(c.Request.Context(), id, uid)
		if dispatch(c, err) {
			return
		}
		c.JSON(200, httpx.OK(item))
	})

	api.GET("/products/:id/comments", func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			return
		}
		items, err := svc.ListComments(c.Request.Context(), id)
		if dispatch(c, err) {
			return
		}
		c.JSON(200, httpx.OK(items))
	})

	auth := api.Group("")
	auth.Use(httpx.RequireAuth())
	{
		auth.POST("/products", func(c *gin.Context) {
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

		auth.PUT("/products/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			var req UpdateReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			item, err := svc.Update(c.Request.Context(), id, httpx.RequireUserID(c), isAdmin(c), req)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		auth.DELETE("/products/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			err := svc.ChangeStatus(c.Request.Context(), id, httpx.RequireUserID(c), isAdmin(c), "OFFLINE")
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})

		auth.POST("/products/:id/comments", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			var req CommentCreateReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			item, err := svc.CreateComment(c.Request.Context(), id, httpx.RequireUserID(c), req)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})
	}
}

// dispatch handles BizError -> Abort. Returns true if the request was handled
// (either an error was written or a panic re-raised for the recovery middleware).
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
	q := ListQuery{
		Keyword:        strPtr(c.Query("keyword")),
		CategoryID:     int64Ptr(c.Query("categoryId")),
		MinPrice:       numericFrom(c.Query("minPrice")),
		MaxPrice:       numericFrom(c.Query("maxPrice")),
		ConditionLevel: strPtr(c.Query("conditionLevel")),
		Campus:         strPtr(c.Query("campus")),
		SortBy:         strPtr(c.Query("sortBy")),
		Status:         strPtr(c.Query("status")),
		SellerID:       int64Ptr(c.Query("sellerId")),
		Page:           int32Ptr(c.Query("page")),
		Size:           int32Ptr(c.Query("size")),
	}
	if v := c.Query("includeAllStatus"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			q.IncludeAllStatus = &b
		}
	}
	return q
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func int64Ptr(s string) *int64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
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

func numericFrom(s string) pgtype.Numeric {
	var n pgtype.Numeric
	if s == "" {
		return n
	}
	if err := n.Scan(s); err != nil {
		return pgtype.Numeric{}
	}
	return n
}
