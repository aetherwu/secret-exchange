package wechat

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"dagong.in/delphi-web-server/config"
	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/medivhzhan/weapp/message/template"
	"github.com/medivhzhan/weapp/token"
)

// GetFormID 获取最快过期的 form id
func GetFormID(col *mgo.Collection, openID string) (string, error) {

	var f models.FormID
	if err := col.Find(bson.M{"user": openID}).Sort("createdat").One(&f); err != nil {
		return "", err
	}

	return f.FormID, nil
}

type addFormIDRequest struct {
	FormID string `binding:"required"`
}

// AddFormID 添加 Form id
func AddFormID(c *gin.Context) {

	var params addFormIDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	col := cols.FormIDs()

	f := models.FormID{User: auth.OpenID, FormID: params.FormID, CreatedAt: time.Now()}

	if err := col.Insert(f); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	c.JSON(http.StatusOK, routers.OK())
}

// FormIDCount 剩余可用 Form id 数量
func FormIDCount(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	col := cols.FormIDs()

	count, err := col.Find(bson.M{"user": auth.OpenID}).Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

type weChatAccessToken struct {
	Token     string
	CreatedAt time.Time
}

// GetWeChatAccessToken 获取微信 access_token
func GetWeChatAccessToken() (string, error) {

	// Get db connection
	session, cols := db.Default()
	defer session.Close()

	// Get collection
	col := cols.WeChat()

	var results []weChatAccessToken

	// Try find token in db
	if err := col.Find(nil).All(&results); err != nil {
		return "", err
	}

	// Fetch new token and save it to db
	if len(results) == 0 {

		// Fetch new token
		tok, _, err := token.AccessToken(config.WxAppID, config.WxAppSecret)
		if err != nil {
			return "", err
		}

		// Save it to db
		if err := col.Insert(&weChatAccessToken{Token: tok, CreatedAt: time.Now()}); err != nil {
			return "", err
		}

		// Return refreshed token
		return tok, nil
	}

	// Return exsiting token
	return results[0].Token, nil
}

// ForceGetWeChatAccessToken 强制获取微信 access_token
func ForceGetWeChatAccessToken() (string, error) {
	session, cols := db.Default()
	defer session.Close()

	col := cols.WeChat()

	// Clear token in db
	col.RemoveAll(nil)

	// Forced fetch new
	tok, _, err := token.AccessToken(config.WxAppID, config.WxAppSecret)
	if err != nil {
		return "", err
	}

	// Save it to db
	if err := col.Insert(&weChatAccessToken{Token: tok, CreatedAt: time.Now()}); err != nil {
		return "", err
	}

	// Return new token
	return tok, nil
}

const (
	exchangedTemplateID               = "9e-KRr_wKRVlHAGfZnEYj78Nn0cfUMAWxPEKjBadE_A"
	likeTemplateID                    = "08GsuBtP0zTMZCn-PK7a6RsCE3kcBat6XVkfxv1b3Uo"
	commentTemplateID                 = "z0CZ2bDD9-WQ4qV54p8Fn9faiPiDQAuuDJR7XbZyTqk"
	invitePermitPassOwnerTemplateID   = "He-JxgjJVTnmdt07r49RW_a5OxCMzOvBX_jb5KF6Ta0"
	invitePermitPassInviterTemplateID = "rckRgLKcZu9zwoUj7FVBosaWtI1cy3ls_xD-1SAjm-U"
)

// ExchangedNotify 收到答案通知
type ExchangedNotify struct {
	Content string `json:"keyword1"`
	Date    string `json:"keyword2"`
}

// InvitePermitPassInviterNotify 邀请通过通知被邀请人
type InvitePermitPassInviterNotify struct {
	Nickname string `json:"keyword1"`
	Text     string `json:"keyword2"`
}

// InvitePermitPassOwnerNotify 邀请通过通知邀请人
type InvitePermitPassOwnerNotify struct {
	Question string `json:"keyword1"`
	Text     string `json:"keyword2"`
}

// LikeNotify 点赞通知
type LikeNotify struct {
	Question string `json:"keyword1"`
	Answer   string `json:"keyword2"`
	Nickname string `json:"keyword3"`
}

// CommentNotify 评论通知
type CommentNotify struct {
	Type     string `json:"keyword1"`
	Nickname string `json:"keyword2"`
	Content  string `json:"keyword3"`
}

// SendNotify 发送模板消息
func SendNotify(openID string, page string, msg interface{}) error {
	templateID := ""
	switch msg.(type) {
	case ExchangedNotify:
		templateID = exchangedTemplateID
	case InvitePermitPassInviterNotify:
		templateID = invitePermitPassInviterTemplateID
	case InvitePermitPassOwnerNotify:
		templateID = invitePermitPassOwnerTemplateID
	case LikeNotify:
		templateID = likeTemplateID
	case CommentNotify:
		templateID = commentTemplateID
	default:
		return errors.New("Msg struct is not define")
	}
	return sendNotify(openID, templateID, page, msg)
}

// sendNotify 发送模板消息
func sendNotify(openID string, templateID string, page string, msg interface{}) error {

	token, err := GetWeChatAccessToken()
	if err != nil {
		return err
	}

	session, cols := db.Default()
	defer session.Close()

	col := cols.FormIDs()

	formID, err := GetFormID(col, openID)
	if err != nil {
		return err
	}

	var inInterface template.Message
	inrec, _ := json.Marshal(msg)
	json.Unmarshal(inrec, &inInterface)

	err = template.Send(openID, templateID, page, formID, inInterface, "", token)
	if err != nil {
		if strings.Contains(err.Error(), "token") {
			token, err = ForceGetWeChatAccessToken()
			if err != nil {
				return err
			}
			err = template.Send(openID, templateID, page, formID, inInterface, "", token)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if err := col.Remove(bson.M{"formid": formID}); err != nil {
		return err
	}

	return nil
}
