package models

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

// Feed feed
type Feed struct {
	ID           bson.ObjectId `json:"id" bson:"_id,omitempty"`
	OpenID       string        // 所有者
	Type         int           // 类型 0：我创建的答案 1：我交换到的答案 2：加星用户创建的答案
	Question     bson.ObjectId // 问题
	Answer       bson.ObjectId // 答案
	AnswerOpenID string        // 答案的创建者
	CreatedAt    time.Time
}
