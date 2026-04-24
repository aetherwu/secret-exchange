package models

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

// Group 群
type Group struct {
	ID         bson.ObjectId `json:"id" bson:"_id,omitempty"`
	QuestionID bson.ObjectId `json:"-"`       // 问题 id
	GroupID    string        `json:"groupID"` // 群 id
	OpenID     string        `json:"-"`       // 答案创建者
	AddUsers   []string      `json:"-"`       // 参加回答的人
	Answers    []GroupAnswer `json:"answers"` // 群里的答案
	CreatedAt  time.Time     `json:"createdAt"`
}

type GroupAnswer struct {
	ID        bson.ObjectId `bson:"_id"`
	OpenID    string
	CreatedAt time.Time
}
