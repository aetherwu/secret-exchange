package models

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

// User 用户
type User struct {

	// System UID
	ID     bson.ObjectId `bson:"_id,omitempty" json:"-"`
	OpenID string        `json:"openid" form:"openid" bson:"openid"`
	Code   string        `json:"-"`

	// Web auth
	Email        string `json:"email" bson:"email"`
	PasswordHash string `json:"-" bson:"passwordhash"`

	// Private auth
	SessionKey  string `json:"-"`
	AccessToken string `json:"-"`

	// Sharing related
	WeChatShareImageStatus int    `json:"weChatShareImageStatus"`
	WeChatShareImage       string `json:"weChatShareImage" form:"weChatShareImage" bson:"wechatshareimage"` // 个人主页分享图
	QRCodeImage            string `json:"QRCodeImage" form:"QRCodeImage" bson:"qrcodeimage"`                // 个人主页小程序码
	QRCodeImageStatus      int    `json:"-"`

	// Activity record
	CreatedAt time.Time `json:"-" bson:"createdat"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updateat"`

	// Public Profile info
	Avatar       string `json:"avatar"`                                               // 自定义头像
	WeChatAvatar string `json:"weChatAvatar" form:"weChatAvatar" bson:"wechatavatar"` // 微信头像
	Nickname     string `json:"nickname" form:"nickname" bson:"nickname"`             // 昵称

	// Public intro
	SelfIntro    string `json:"selfIntro" form:"selfIntro" bson:"selfintro"`
	Profession   string `json:"profession" form:"profession" bson:"profession"`
	Verification string `json:"verification" form:"verification" bson:"verification"`
}

// LeanUser user info without privacy info
type LeanUser struct {
	OpenID       string `json:"openID"`
	Nickname     string `json:"nickname"`
	Avatar       string `json:"avatar"`
	SelfIntro    string `json:"selfIntro"`
	Profession   string `json:"profession"`
	Verification string `json:"verification"`
}

// ToLean 脱敏
func (user *User) ToLean() LeanUser {
	avatar := user.Avatar
	if avatar == "" {
		avatar = user.WeChatAvatar
	}
	return LeanUser{
		OpenID:       user.OpenID,
		Nickname:     user.Nickname,
		Avatar:       avatar,
		SelfIntro:    user.SelfIntro,
		Profession:   user.Profession,
		Verification: user.Verification,
	}
}

// Users array
type Users []User

// MapToOpenID : users list with opendID only
func (users Users) MapToOpenID() []string {
	results := []string{}
	for _, v := range users {
		results = append(results, v.OpenID)
	}
	return results
}
