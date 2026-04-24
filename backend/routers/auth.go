package routers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dagong.in/delphi-web-server/config"
	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"github.com/medivhzhan/weapp"
	"golang.org/x/crypto/bcrypt"
)

type registRequest struct {
	Code          string `binding:"required"`
	RawData       string `binding:"required"`
	EncryptedData string `binding:"required"`
	Signature     string `binding:"required"`
	Iv            string `binding:"required"`
}

type registResponse struct {
	OpenID      string `json:"openid"`
	AccessToken string `json:"accessToken"`
}

// Regist handles WeChat login (preserved from archive).
func Regist(c *gin.Context) {
	var params registRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, ParamsError())
		return
	}

	resp, err := weapp.Login(config.WxAppID, config.WxAppSecret, params.Code)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, LoginError())
		return
	}

	userInfo, err := weapp.DecryptUserInfo(params.RawData, params.EncryptedData, params.Signature, params.Iv, resp.SessionKey)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, LoginError())
		return
	}

	session, cols := db.Default()
	defer session.Close()
	col := cols.Users()

	query := bson.M{"openid": resp.OpenID}
	var users []models.User
	if err := col.Find(query).All(&users); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, LoginError())
		return
	}

	accessToken := genAccessToken(resp.OpenID)
	now := time.Now()

	if len(users) == 0 {
		user := models.User{
			Avatar:                 userInfo.Avatar,
			WeChatAvatar:           userInfo.Avatar,
			Nickname:               userInfo.Nickname,
			OpenID:                 resp.OpenID,
			Code:                   params.Code,
			SessionKey:             resp.SessionKey,
			AccessToken:            accessToken,
			WeChatShareImage:       "",
			WeChatShareImageStatus: models.WeChatShareImageStatusNone,
			CreatedAt:              now,
			UpdatedAt:              now,
		}
		if err := col.Insert(user); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, LoginError())
			return
		}
	} else {
		doc := bson.M{"$set": bson.M{
			"wechatavatar": userInfo.Avatar,
			"nickname":     userInfo.Nickname,
			"sessionkey":   resp.SessionKey,
			"accesstoken":  accessToken,
			"updatedat":    now,
		}}
		if err := col.Update(query, doc); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, LoginError())
			return
		}
	}

	c.JSON(http.StatusOK, registResponse{OpenID: resp.OpenID, AccessToken: accessToken})
}

// --- Web auth: email + password ---

type registerLocalRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
}

type loginLocalRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterLocal handles POST /auth/register {email, password, nickname}
func RegisterLocal(c *gin.Context) {
	var params registerLocalRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, ParamsError())
		return
	}

	params.Email = strings.TrimSpace(strings.ToLower(params.Email))
	params.Nickname = strings.TrimSpace(params.Nickname)

	if len(params.Password) < 6 {
		c.AbortWithStatusJSON(http.StatusOK, NewError(1001, "Password must be at least 6 characters"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), 12)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, DefaultError())
		return
	}

	openID := "u_" + uuid.New().String()
	accessToken := genAccessToken(openID)
	now := time.Now()

	session, cols := db.Default()
	defer session.Close()
	col := cols.Users()

	user := models.User{
		Email:        params.Email,
		PasswordHash: string(hash),
		Nickname:     params.Nickname,
		OpenID:       openID,
		AccessToken:  accessToken,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := col.Insert(user); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.AbortWithStatusJSON(http.StatusOK, NewError(1111, "Email already registered"))
			return
		}
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, DefaultError())
		return
	}

	c.JSON(http.StatusOK, registResponse{OpenID: openID, AccessToken: accessToken})
}

// LoginLocal handles POST /auth/login {email, password}
func LoginLocal(c *gin.Context) {
	var params loginLocalRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, ParamsError())
		return
	}

	params.Email = strings.TrimSpace(strings.ToLower(params.Email))

	session, cols := db.Default()
	defer session.Close()
	col := cols.Users()

	var user models.User
	if err := col.Find(bson.M{"email": params.Email}).One(&user); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewError(1005, "Invalid email or password"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(params.Password)); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, NewError(1005, "Invalid email or password"))
		return
	}

	accessToken := genAccessToken(user.OpenID)
	now := time.Now()

	col.Update(bson.M{"openid": user.OpenID}, bson.M{"$set": bson.M{
		"accesstoken": accessToken,
		"updatedat":   now,
	}})

	c.JSON(http.StatusOK, registResponse{OpenID: user.OpenID, AccessToken: accessToken})
}

func genAccessToken(openID string) string {
	u := utils.UUID()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	return utils.MD5(u + timestamp + openID + utils.UUID())
}

// Auth verifies openID + accessToken from the database.
func Auth(openID string, accessToken string) *models.User {
	if openID == "" || accessToken == "" {
		return nil
	}
	session, cols := db.Default()
	defer session.Close()

	col := cols.Users()
	query := bson.M{"openid": openID, "accesstoken": accessToken}
	var user *models.User
	col.Find(query).One(&user)
	return user
}
