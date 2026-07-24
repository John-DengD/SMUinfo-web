package httpx

import "github.com/gin-gonic/gin"

const CtxUserID = "uid"
const CtxRole = "role"

func UserID(c *gin.Context) int64 {
	v, _ := c.Get(CtxUserID)
	id, _ := v.(int64)
	return id
}

func Role(c *gin.Context) string {
	v, _ := c.Get(CtxRole)
	s, _ := v.(string)
	return s
}

func RequireUserID(c *gin.Context) int64 {
	id := UserID(c)
	if id == 0 {
		panic(NewBiz(401, "未登录"))
	}
	return id
}
