package middleware

import (
	"dagong.in/delphi-web-server/config"
	"github.com/gin-gonic/gin"
)

func BaseAuth() gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{
		config.DashboardUser: config.DashboardPassword,
	})
}
