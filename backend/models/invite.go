package models

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

type Invite struct {
	ID         bson.ObjectId `json:"id" bson:"_id,omitempty"`
	QuestionID bson.ObjectId
	AnswerID   bson.ObjectId
	Owner      string
	OpenID     string
	CreatedAt  time.Time
}

type InvitePermit struct {
	ID           bson.ObjectId `json:"id" bson:"_id,omitempty"`
	InviteID     bson.ObjectId
	QuestionID   bson.ObjectId
	AnswerID     bson.ObjectId
	Owner        string
	InviteUserID string
	OpenID       string
	Status       int
	CreatedAt    time.Time
}

const (
	InvitePermitStatusWait   = 0
	InvitePermitStatusPass   = 1
	InvitePermitStatusReject = 2
)

type InviteBadgeNumber struct {
	ID     bson.ObjectId `json:"id" bson:"_id,omitempty"`
	OpenID string
	Number int
}
