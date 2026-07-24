package httpx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthParse(j JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if strings.HasPrefix(h, "Bearer ") {
			if uid, role, err := j.Parse(h[7:]); err == nil {
				c.Set(CtxUserID, uid)
				c.Set(CtxRole, role)
			}
		}
		c.Next()
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if UserID(c) == 0 {
			c.JSON(http.StatusUnauthorized, Fail(401, "登录已过期，请重新登录"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if UserID(c) == 0 {
			c.JSON(http.StatusUnauthorized, Fail(401, "登录已过期，请重新登录"))
			c.Abort()
			return
		}
		if !strings.EqualFold(Role(c), "ADMIN") {
			c.JSON(http.StatusForbidden, Fail(403, "无权访问"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// CORS 反射 Origin 命中 origins 则回写头；OPTIONS 直接 204。
func CORS(origins []string) gin.HandlerFunc {
	set := map[string]bool{}
	for _, o := range origins {
		set[strings.TrimSpace(o)] = true
	}
	return func(c *gin.Context) {
		o := c.GetHeader("Origin")
		c.Header("Vary", "Origin")
		if set[o] {
			c.Header("Access-Control-Allow-Origin", o)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
