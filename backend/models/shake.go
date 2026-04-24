package models

import (
	"github.com/globalsign/mgo/bson"
)

// ShakedQuestions a 对 b 用户 摇到过的问题
type ShakedQuestions struct {
	OpenID    string
	User      string
	Questions []bson.ObjectId
}
