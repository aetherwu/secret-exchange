package profile

import (
	"log"
	"net/http"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/draw"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type getQRCodeRequest struct {
	ID string `binging:"required"`
}

type qrCodeResponse struct {
	QRCode string `json:"qrCode"`
	Owner  bool   `json:"owner"`
}

// GetQRCode 获取个人主页小程序码
func GetQRCode(c *gin.Context) {

	var params getQRCodeRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	user, err := routerUtils.FindUser(cols, params.ID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	avatar := user.Avatar
	if avatar == "" {
		avatar = user.WeChatAvatar
	}

	resp := qrCodeResponse{QRCode: routerUtils.QiniuKeyToURL(user.QRCodeImage)}
	resp.Owner = auth.OpenID == user.OpenID

	if user.QRCodeImageStatus > models.WeChatShareImageStatusNone {
		c.JSON(http.StatusOK, resp)
		return
	}

	totalQuestionCount, _ := cols.Questions().Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).Count()
	image, err := draw.GenProfileQRCodeWeChatShareImage(user.Nickname, user.OpenID, avatar, totalQuestionCount)
	if err != nil {
		log.Println(err)
	} else {
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
		} else {
			cols.Users().Update(bson.M{"openid": user.OpenID}, bson.M{"$set": bson.M{
				"qrcodeimage":       key,
				"qrcodeimagestatus": models.WeChatShareImageStatusGenerated,
			}})
			resp.QRCode = routerUtils.QiniuKeyToURL(key)
		}
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateQRCode 强制更新个人主页小程序码
func UpdateQRCode(c *gin.Context) {

	var params getQRCodeRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	user, err := routerUtils.FindUser(cols, params.ID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if auth.OpenID != user.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.PermissionError())
		return
	}

	avatar := user.Avatar
	if avatar == "" {
		avatar = user.WeChatAvatar
	}

	resp := qrCodeResponse{QRCode: routerUtils.QiniuKeyToURL(user.QRCodeImage)}

	totalQuestionCount, _ := cols.Questions().Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).Count()
	image, err := draw.GenProfileQRCodeWeChatShareImage(user.Nickname, user.OpenID, avatar, totalQuestionCount)
	if err != nil {
		log.Println(err)
	} else {
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
		} else {
			cols.Users().Update(bson.M{"openid": user.OpenID}, bson.M{"$set": bson.M{
				"qrcodeimage":       key,
				"qrcodeimagestatus": models.WeChatShareImageStatusGenerated,
			}})
			resp.QRCode = routerUtils.QiniuKeyToURL(key)
		}
	}

	c.JSON(http.StatusOK, resp)
}
