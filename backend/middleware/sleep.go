package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Sleep 延迟返回 中间件
func Sleep(milliseconds time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		time.Sleep(milliseconds * time.Millisecond)
		c.Next()
	}
}
