package user

import (
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/draw"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type userInfoResponse struct {
	Avatar                 string `json:"avatar"`
	Nickname               string `json:"nickname"`
	AnswerCount            int    `json:"answerCount"`
	ExchangeCount          int    `json:"exchangeCount"`
	FormIDCount            int    `json:"formIDCount"`
	TotalQuestionCount     int    `json:"totalQuestionCount"`
	FriendsCount           int    `json:"friendsCount"`
	WeChatShareImageStatus int    `json:"weChatShareImageStatus"`
	WeChatShareImage       string `json:"weChatShareImage"`
	SelfIntro              string `json:"selfIntro" form:"selfIntro" bson:"selfIntro"`
	Profession             string `json:"profession" form:"profession" bson:"profession"`
	Verification           string `json:"verification" form:"verification" bson:"verification"`
}

// UserInfo 获取 user info
func Profile(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	// 收集到的答案总数
	userAnswerCountersCol := cols.UserAnswerCounters()
	counters := models.UserAnswerCounters{}
	userAnswerCountersCol.Find(bson.M{"openid": auth.OpenID}).One(&counters)

	// 提醒宝石数
	formIDCol := cols.FormIDs()
	formIDCount, err := formIDCol.Find(bson.M{"user": auth.OpenID}).Count()
	if err != nil {
		log.Println(err)
	}

	// 已答问题数
	totalQuestionCount, err := cols.Questions().Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).Count()
	if err != nil {
		log.Println(err)
	}

	// 交换过的好友数
	exchangeFriends := bson.M{}
	cols.AnswerExchangeUserList().Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID}},
		bson.M{"$project": bson.M{
			"count": bson.M{"$size": "$exchanges"},
		}},
	}).One(&exchangeFriends)

	exchangeFriendsCount := 0
	if count, ok := exchangeFriends["count"].(int); ok {
		exchangeFriendsCount = count
	}

	// 头像
	avatar := auth.Avatar
	if avatar == "" {
		avatar = auth.WeChatAvatar
	}

	// 小程序分享截图
	weChatShareImage := auth.WeChatShareImage
	if weChatShareImage != "" {
		weChatShareImage = routerUtils.QiniuKeyToURL(weChatShareImage)
	}

	resp := userInfoResponse{
		Avatar:                 avatar,
		Nickname:               auth.Nickname,
		AnswerCount:            counters.AnswerCount,
		ExchangeCount:          counters.AllExchangeCount,
		FormIDCount:            formIDCount,
		TotalQuestionCount:     totalQuestionCount,
		FriendsCount:           exchangeFriendsCount,
		WeChatShareImage:       weChatShareImage,
		WeChatShareImageStatus: auth.WeChatShareImageStatus,
		SelfIntro:              auth.SelfIntro,
		Profession:             auth.Profession,
		Verification:           auth.Verification,
	}

	c.JSON(http.StatusOK, resp)
}

type updateRequest struct {
	Nickname string `binding:"required"`
	Avatar   string `binding:"required"`
}

// UpdateUserInfo 更新 nickname;avatar; from WeChat Client by Auth.OPENID
func UpdateUserInfo(c *gin.Context) {

	var params updateRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	col := cols.Users()

	set := bson.M{"nickname": auth.Nickname, "avatar": auth.Avatar}
	if params.Nickname != auth.Nickname {
		set["nickname"] = params.Nickname
	}

	upgradeAvatar := false
	if params.Avatar != auth.WeChatAvatar {
		image, format, err := routers.DownloadImage(params.Avatar)
		if err != nil {
			log.Println(err)
		} else {
			avatarKey, err := routers.UploadWXAvatar(image, format)
			if err != nil {
				log.Println(err)
			} else {
				upgradeAvatar = true
				set["avatar"] = routerUtils.QiniuKeyToURL(avatarKey)
				set["wechatavatar"] = params.Avatar
			}
		}
	}

	if upgradeAvatar {
		set["wechatshareimagestatus"] = models.WeChatShareImageStatusNone
	}

	set["updatedat"] = time.Now()
	if err := col.Update(bson.M{"openid": auth.OpenID}, bson.M{"$set": set}); err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, gin.H{"avatar": set["avatar"], "nickname": set["nickname"]})
}

type profileWeChatShareImageResponse struct {
	WeChatShareImage string `json:"weChatShareImage"`
}

// UpdateProfileWeChatShareImage 更新个人主页分享图片
func UpdateProfileWeChatShareImage(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	col := cols.Users()

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

	resp := profileWeChatShareImageResponse{WeChatShareImage: routerUtils.QiniuKeyToURL(user.WeChatShareImage)}

	if user.WeChatShareImageStatus > models.WeChatShareImageStatusNone {
		c.JSON(http.StatusOK, resp)
		return
	}

	image, err := draw.GenProfileWeChatShareImage(avatar, user.SelfIntro, user.Nickname)
	if err != nil {
		log.Println(err)
	} else {
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
		} else {
			col.Update(bson.M{"openid": user.OpenID}, bson.M{"$set": bson.M{
				"wechatshareimage":       key,
				"wechatshareimagestatus": models.WeChatShareImageStatusGenerated,
			}})
			resp.WeChatShareImage = routerUtils.QiniuKeyToURL(key)
		}
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateUserIntro intro
func UpdateUserIntro(c *gin.Context) {

	session, cols := db.Default()
	defer session.Close()

	user := models.User{}
	if err := c.Bind(&user); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	col := cols.Users()
	auth := routers.GetContextAuth(c)
	query := bson.M{"openid": auth.OpenID}

	doc := bson.M{
		"selfintro":              user.SelfIntro,
		"profession":             user.Profession,
		"verification":           user.Verification,
		"wechatshareimagestatus": models.WeChatShareImageStatusNone,
	}
	if err := col.Update(query, bson.M{"$set": doc}); err != nil {
		log.Println(err)
	}

	//update wechatShareImage

	c.JSON(http.StatusOK, gin.H{
		"openID":       user.OpenID,
		"selfIntro":    user.SelfIntro,
		"profession":   user.Profession,
		"verification": user.Verification,
	})

}
