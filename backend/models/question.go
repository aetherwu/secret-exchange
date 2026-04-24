package models

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

// Question 问题
type Question struct {
	ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Content     string        `json:"content"` // 问题内容
	AnswerCount int           `json:"-"`       // 答案数
	Status      int           `json:"status"`  //
	Level       int           `json:"-"`       // 等级
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

// 隐私等级
const (
	QuestionPrivacyFree      = -1     // 默认可公开展示 -1
	QuestionPrivacyOpen      = 0      // 默认可公开交换 0
	QuestionPrivacyFriend    = 1 << 0 // 默认好友可交换 1
	QuestionPrivacyProactive = 1 << 1 // 默认仅限主动交换 2
)

// 状态
const (
	StatusDelete = -1
	StatusNormal = 0
)

// QuestionLimit 每天首页回答次数上限
type QuestionLimit struct {
	OpenID    string
	Limit     int
	ExpiredAt time.Time
}

// SkippedQuestions 某个用户首页跳过的问题
type SkippedQuestions struct {
	OpenID    string
	Questions []bson.ObjectId
}
