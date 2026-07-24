package message

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Register wires message routes onto the /api group. All routes require auth.
//
//	POST /api/messages                        send a message
//	GET  /api/messages                        conversation list for current user
//	GET  /api/messages/conversation/{userId}  full thread with a peer (marks read)
//	GET  /api/messages/unread-count           total unread count
func Register(api *gin.RouterGroup, svc *Service) {
	auth := api.Group("")
	auth.Use(httpx.RequireAuth())
	{
		auth.POST("/messages", func(c *gin.Context) {
			var req SendReq
			if err := c.ShouldBindJSON(&req); err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			item, err := svc.Send(c.Request.Context(), httpx.RequireUserID(c), req)
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(item))
		})

		auth.GET("/messages", func(c *gin.Context) {
			convs, err := svc.Conversations(c.Request.Context(), httpx.RequireUserID(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(convs))
		})

		// Must be registered before the catch-all parameter route; Gin matches
		// static path segments before param segments so ordering is safe here.
		auth.GET("/messages/unread-count", func(c *gin.Context) {
			count, err := svc.UnreadCount(c.Request.Context(), httpx.RequireUserID(c))
			if dispatch(c, err) {
				return
			}
			c.JSON(200, httpx.OK(UnreadCountResp{Count: count}))
		})

		auth.GET("/messages/conversation/:userId", func(c *gin.Context) {
			peerID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
			if err != nil {
				httpx.Abort(c, httpx.Biz("参数错误"))
				return
			}
			items, err2 := svc.Conversation(c.Request.Context(), httpx.RequireUserID(c), peerID)
			if dispatch(c, err2) {
				return
			}
			c.JSON(200, httpx.OK(items))
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
