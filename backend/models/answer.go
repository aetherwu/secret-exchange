package models

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

type Answer struct {
	ID               bson.ObjectId          `json:"id" bson:"_id,omitempty"`
	OpenID           string                 `json:"-"`                 // 创建者
	QuestionID       bson.ObjectId          `json:"questionID"`        // 问题 ID
	Content          string                 `json:"content"`           // 答案内容
	Length           int                    `json:"length" beson:"-"`  //答案内容字数
	ExchangeCount    int                    `json:"exchangeCount"`     // 交换次数
	BeExchangedCount int                    `json:"beExchangedCount"`  // 被交换次数
	AllExchangeCount int                    `json:"allExchangeCount"`  // 交换次数 + 被交换次数
	Status           int                    `json:"status"`            // -1表示被删除
	Privacy          int                    `json:"privacy"`           // 隐私等级
	Comments         []AnswerCommentDisplay `json:"comments" bson:"-"` // 评论
	IsExchanged      bool                   `json:"isExchanged"`

	Likes     int                      `json:"likes"`              // 点赞数
	LikeUsers map[int][]AnswerLikeUser `json:"likeUsers" bson:"-"` // 点赞的用户
	IsLike    bool                     `json:"isLike" bson:"-"`    // 是否点过赞

	WeChatShareImage       string `json:"weChatShareImage"` // 微信分享图
	WeChatShareImageStatus int    `json:"weChatShareImageStatus"`
	QRCodeImage            string `json:"qrCodeImage"` // 小程序码分享图
	QRCodeImageStatus      int    `json:"qrCodeImageStatus"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// 隐私等级
const (
	AnswerPrivacyFree   = -1     // 公开展示 -1
	AnswerPrivacyOpen   = 0      // 公开交换  0
	AnswerPrivacyFriend = 1 << 0 // 好友可交换  1
	AnswerPrivacyMyself = 1 << 1 // 仅限主动交换  2
)

type Answers []Answer

func (answers Answers) Len() int {
	return len(answers)
}

func (answers Answers) Less(i, j int) bool {
	return answers[i].CreatedAt.After(answers[j].CreatedAt)
}

func (answers Answers) Swap(i, j int) {
	answers[i], answers[j] = answers[j], answers[i]
}

func (answers Answers) MapToObjectID(f func(Answer) bson.ObjectId) []bson.ObjectId {
	vsm := make([]bson.ObjectId, len(answers))
	for i, v := range answers {
		vsm[i] = f(v)
	}
	return vsm
}

// 微信分享
const (
	WeChatShareImageStatusNone       = 1
	WeChatShareImageStatusGenerating = 2
	WeChatShareImageStatusGenerated  = 3
)

type AnswerDisplay struct {
	Answer
	User LeanUser `json:"user"`
}

type QA struct {
	Question Question `json:"question"`
	Answer   Answer   `json:"answer"`
}

type QADisplay struct {
	Question Question      `json:"question"`
	Answer   AnswerDisplay `json:"answer"`
}

type QADisplays []QADisplay

func (p QADisplays) Len() int {
	return len(p)
}

func (p QADisplays) Less(i, j int) bool {
	return p[i].Answer.CreatedAt.After(p[j].Answer.CreatedAt)
}

func (p QADisplays) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type AnswerExchange struct {
	ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Question    bson.ObjectId // 交换的问题
	Source      bson.ObjectId // 被交换的答案
	SourceUser  string        // 被交换的答案创建者
	Swapper     bson.ObjectId // 主动交换的答案
	SwapperUser string        // 主动交换的答案创建者
	CreatedAt   time.Time
}

// AnswerExchangeUserList 某用户所有交换过的用户
type AnswerExchangeUserList struct {
	ID        bson.ObjectId `json:"id" bson:"_id,omitempty"`
	OpenID    string
	Exchanges []AnswerExchangeUser
}

type AnswerExchangeUser struct {
	OpenID     string        // 用户
	QuestionID bson.ObjectId // 最后交换的问题
	AnswerID   bson.ObjectId //最后交换的答案
	Count      int           //交换过的次数
	UpdatedAt  time.Time
}

// AnswerExchangeAtUser 某用户对某用户的所有交换答案
type AnswerExchangeAtUser struct {
	ID        bson.ObjectId                `json:"id" bson:"_id,omitempty"`
	OpenID    string                       // 我
	User      string                       // 对方
	Answers   []AnswerExchangeAtUserDetail // 所有交换过的答案
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AnswerExchangeAtUserDetail struct {
	QuestionID bson.ObjectId // 问题
	AnswerID   bson.ObjectId // 答案
	CreatedAt  time.Time
}

// UserAnswerCounters 用户回答和交换计数器
type UserAnswerCounters struct {
	ID               bson.ObjectId `json:"id" bson:"_id,omitempty"`
	OpenID           string
	AnswerCount      int `json:"answerCount"`      // 答案数
	ExchangeCount    int `json:"exchangeCount"`    // 主动交换次数
	BeExchangedCount int `json:"beExchangedCount"` // 被动交换此时
	AllExchangeCount int `json:"allExchangeCount"` // 所有交换次数
}
