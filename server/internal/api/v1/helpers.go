// API v1 共享工具函数
package v1

import "github.com/gin-gonic/gin"

func clientIP(c *gin.Context) string {
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return ip
	}
	return c.ClientIP()
}

func userStatus(s int16) string {
	if s == 2 {
		return "frozen"
	}
	return "active"
}
