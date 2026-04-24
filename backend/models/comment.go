package models

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

// CommentImageInfo 图片
type CommentImageInfo struct {
	Key    string  `json:"key"`
	Size   int     `json:"size"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Src    string  `json:"src" bson:"-"`
}

// CommentVideoInfo 视频
type CommentVideoInfo struct {
	Key      string  `json:"key"`
	Size     int     `json:"size"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Duration float64 `json:"duration"`
	Src      string  `json:"src" bson:"-"`
}

// CommentAudioInfo 音频
type CommentAudioInfo struct {
	Key      string  `json:"key"`
	Size     int     `json:"size"`
	Duration float64 `json:"duration"`
	Src      string  `json:"src" bson:"-"`
}

type CommentMap struct {
	Answer   bson.ObjectId
	Comments []AnswerComment
}

type AnswerComment struct {
	OpenID    string             `json:"-"`
	Content   string             `json:"content"`
	Images    []CommentImageInfo `json:"images"`
	Audio     *CommentAudioInfo  `json:"audio"`
	Reply     string             `json:"-"`
	CreatedAt time.Time          `json:"createdAt"`
}

type CommentMapPipeGroup struct {
	Answer   bson.ObjectId
	Comments AnswerComment
}

func (comment *AnswerComment) IsReply() bool {
	return comment.Reply != ""
}

type AnswerCommentDisplay struct {
	AnswerComment
	LeanUser
	Reply *LeanUser `json:"reply"`
}
