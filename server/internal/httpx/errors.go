package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type BizError struct {
	Code int
	Msg  string
}

func (e BizError) Error() string { return e.Msg }

func NewBiz(code int, msg string) BizError { return BizError{code, msg} }

func Biz(msg string) BizError { return BizError{400, msg} }

// Abort 用于 handler 内主动返回业务错误
func Abort(c *gin.Context, e BizError) {
	c.JSON(http.StatusOK, Fail(e.Code, e.Msg))
	c.Abort()
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				if be, ok := r.(BizError); ok {
					c.JSON(http.StatusOK, Fail(be.Code, be.Msg))
				} else {
					c.JSON(http.StatusOK, Fail(500, "服务器错误"))
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}
