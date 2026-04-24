package models

import "github.com/globalsign/mgo/bson"

// AnswerLike 答案点赞
type AnswerLike struct {
	Answer bson.ObjectId    // 答案
	Likes  []AnswerLikeType // 所有点赞
}

type AnswerLikeType struct {
	OpenID string // 点赞的用户
	Type   int    // 点赞类型
}

// AnswerLikeUser 返回客户端用
type AnswerLikeUser struct {
	Nickname string `json:"nickname"`
	Type     int    `json:"type"`
}
