package admin

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
	"github.com/John-DengD/smu-deal/server/internal/product"
)

// Register wires the admin backoffice routes onto the /api group. All routes
// live under /api/admin and require an ADMIN role (RequireAdmin -> 403 无权访问
// for non-admins, 401 for unauthenticated). Mirrors Java AdminController.
//
//	GET    /api/admin/users                 list users (keyword, page, size)
//	PUT    /api/admin/users/:id/status      enable/disable a user
//	GET    /api/admin/products              list products (all statuses)
//	PUT    /api/admin/products/:id/status   force status change
//	GET    /api/admin/categories            list active categories
//	POST   /api/admin/categories            create category
//	PUT    /api/admin/categories/:id        update category
//	DELETE /api/admin/categories/:id        delete category
//	GET    /api/admin/reports               list reports (optional status)
//	PUT    /api/admin/reports/:id           handle a report
//	GET    /api/admin/feedback              list feedback (optional status)
//	PUT    /api/admin/feedback/:id          reply/resolve feedback
//	GET    /api/admin/announcements         list all announcements
//	POST   /api/admin/announcements         create announcement
//	PUT    /api/admin/announcements/:id     update announcement
//	DELETE /api/admin/announcements/:id     delete announcement
func Register(api *gin.RouterGroup, svc *Service) {
	g := api.Group("/admin")
	g.Use(httpx.RequireAdmin())
	{
		// --- users ---
		g.GET("/users", func(c *gin.Context) {
			page, err := svc.ListUsers(c.Request.Context(),
				strPtr(c.Query("keyword")), int32Ptr(c.Query("page")), int32Ptr(c.Query("size")))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(page))
		})

		g.PUT("/users/:id/status", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			var body map[string]string
			if err := c.ShouldBindJSON(&body); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			if dispatch(c, svc.ChangeUserStatus(c.Request.Context(), id, body["status"])) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})

		// --- products ---
		g.GET("/products", func(c *gin.Context) {
			page, err := svc.ListProducts(c.Request.Context(), parseListQuery(c), httpx.RequireUserID(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(page))
		})

		g.PUT("/products/:id/status", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			var body map[string]string
			if err := c.ShouldBindJSON(&body); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			if dispatch(c, svc.ChangeProductStatus(c.Request.Context(), id, httpx.RequireUserID(c), body["status"])) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})

		// --- categories ---
		g.GET("/categories", func(c *gin.Context) {
			items, err := svc.ListCategories(c.Request.Context())
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(items))
		})

		g.POST("/categories", func(c *gin.Context) {
			var req CategoryReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			item, err := svc.CreateCategory(c.Request.Context(), req)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		g.PUT("/categories/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			var req CategoryReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			item, err := svc.UpdateCategory(c.Request.Context(), id, req)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		g.DELETE("/categories/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			if dispatch(c, svc.DeleteCategory(c.Request.Context(), id)) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})

		// --- reports ---
		g.GET("/reports", func(c *gin.Context) {
			items, err := svc.ListReports(c.Request.Context(), strPtr(c.Query("status")))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(items))
		})

		g.PUT("/reports/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			var req ReportHandleReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			if dispatch(c, svc.HandleReport(c.Request.Context(), id, req)) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})

		// --- feedback ---
		g.GET("/feedback", func(c *gin.Context) {
			items, err := svc.ListFeedback(c.Request.Context(), strPtr(c.Query("status")))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(items))
		})

		g.PUT("/feedback/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			var req FeedbackReplyReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			if dispatch(c, svc.ReplyFeedback(c.Request.Context(), id, req)) {
				return
			}
			c.JSON(200, httpx.OK(nil))
		})

		// --- announcements ---
		g.GET("/announcements", func(c *gin.Context) {
			items, err := svc.ListAnnouncements(c.Request.Context())
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(items))
		})

		g.POST("/announcements", func(c *gin.Context) {
			var req AnnouncementSaveReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			item, err := svc.CreateAnnouncement(c.Request.Context(), httpx.RequireUserID(c), req)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		g.PUT("/announcements/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			var req AnnouncementSaveReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			item, err := svc.UpdateAnnouncement(c.Request.Context(), id, req)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		g.DELETE("/announcements/:id", func(c *gin.Context) {
			id, ok := pathID(c)
			if !ok {
				return
			}
			if dispatch(c, svc.DeleteAnnouncement(c.Request.Context(), id)) {
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Abort(c, httpx.Biz("参数错误"))
		return 0, false
	}
	return id, true
}

// parseListQuery mirrors product.parseListQuery for the admin product list.
// includeAllStatus is forced true in the service, so it is not read here.
func parseListQuery(c *gin.Context) product.ListQuery {
	return product.ListQuery{
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
