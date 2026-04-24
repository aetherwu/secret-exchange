package answer

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

// GetQRCode 获取答案的小程序码
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

	// 查询答案和问题是否存在
	question, answer, err := routerUtils.FindQA(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 获取答案的创建者
	user, err := routerUtils.FindUser(cols, answer.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	resp := qrCodeResponse{QRCode: routerUtils.QiniuKeyToURL(answer.QRCodeImage)}
	// 是否是答案的创建者
	resp.Owner = auth.OpenID == user.OpenID

	// 是否需要更新
	if answer.QRCodeImageStatus > models.WeChatShareImageStatusNone {
		c.JSON(http.StatusOK, resp)
		return
	}

	qa := models.QADisplay{
		Question: *question,
		Answer: models.AnswerDisplay{
			Answer: *answer,
			User:   user.ToLean(),
		},
	}

	// 绘制答案的小程序码
	image, err := draw.GenAnswerQRCodeWeChatShareImage(qa)
	if err != nil {
		log.Println(err)
	} else {
		// 上传图片
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
		} else {
			// 更新数据库
			cols.Answers().Update(bson.M{"_id": answer.ID}, bson.M{"$set": bson.M{
				"qrcodeimage":       key,
				"qrcodeimagestatus": models.WeChatShareImageStatusGenerated,
			}})
			resp.QRCode = routerUtils.QiniuKeyToURL(key)
		}
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateQRCode 强制刷新答案的小程序码
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

	// 查询答案和问题是否存在
	question, answer, err := routerUtils.FindQA(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 获取答案的创建者
	user, err := routerUtils.FindUser(cols, answer.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 判断是否是答案的创建者
	if user.OpenID != auth.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.PermissionError())
		return
	}

	resp := qrCodeResponse{QRCode: routerUtils.QiniuKeyToURL(answer.QRCodeImage)}

	qa := models.QADisplay{
		Question: *question,
		Answer: models.AnswerDisplay{
			Answer: *answer,
			User:   user.ToLean(),
		},
	}

	// 绘制答案的小程序码
	image, err := draw.GenAnswerQRCodeWeChatShareImage(qa)
	if err != nil {
		log.Println(err)
	} else {
		// 上传图片
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
		} else {
			// 更新数据库
			cols.Answers().Update(bson.M{"_id": answer.ID}, bson.M{"$set": bson.M{
				"qrcodeimage":       key,
				"qrcodeimagestatus": models.WeChatShareImageStatusGenerated,
			}})
			resp.QRCode = routerUtils.QiniuKeyToURL(key)
		}
	}

	c.JSON(http.StatusOK, resp)
}
