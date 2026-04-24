package routers

import (
	"net/http"
	"strings"

	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/utils"
	"github.com/gin-gonic/gin"
)

// AuthRequired 强制验证 中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 auth
		token := c.GetHeader("Authorization")
		// 获取 openID accessToken
		openID, accessToken := DecodeAuthToken(token)
		auth := Auth(openID, accessToken)
		// 没有找到用户或授权无效
		if auth == nil {
			c.AbortWithStatusJSON(http.StatusOK, AuthError())
		} else {
			// 保存 auth 到 context
			c.Set("_auth", *auth)
			c.Next()
		}
	}
}

// GetContextAuth 从 context 中获取 auth
func GetContextAuth(c *gin.Context) (u models.User) {
	if val, ok := c.Get("_auth"); ok && val != nil {
		u, _ = val.(models.User)
	}
	return
}

// 尝试验证
func TryGetAuth(c *gin.Context) *models.User {
	token := c.GetHeader("Authorization")
	openID, accessToken := DecodeAuthToken(token)
	return Auth(openID, accessToken)
}

// 解码 auth token
func DecodeAuthToken(token string) (string, string) {
	if token == "" {
		return "", ""
	}
	arr := strings.Split(utils.BASE64Decode(token), ";")
	if len(arr) != 3 {
		return "", ""
	}
	if arr[2] != "0x1072D" {
		return "", ""
	}
	return arr[0], arr[1]
}

func Test(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

type IDRequest struct {
	ID string `binding:"required"`
}

type OpenIDRequest struct {
	OpenID string `binding:"required"`
}
