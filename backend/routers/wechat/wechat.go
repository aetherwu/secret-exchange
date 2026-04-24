package wechat

import (
	"log"
	"net/http"

	"dagong.in/delphi-web-server/routers"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/medivhzhan/weapp"
)

type decryptShareInfoRequest struct {
	Data string `binding:"required"`
	Iv   string `binding:"required"`
}

// DecryptShareInfo 解密 群id
func DecryptShareInfo(c *gin.Context) {

	var params decryptShareInfoRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	gid, err := weapp.DecryptShareInfo(auth.SessionKey, params.Data, params.Iv)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.AuthError())
		return
	}

	c.JSON(http.StatusOK, gin.H{"gid": gid})
}
