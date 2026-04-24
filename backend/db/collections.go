package db

import "github.com/globalsign/mgo"

type Collections struct {
	db *mgo.Database
}

// Users 用户
func (cols Collections) Users() *mgo.Collection {
	return cols.db.C("users")
}

// Questions 问题
func (cols Collections) Questions() *mgo.Collection {
	return cols.db.C("questions")
}

// QuestionLimit 每天交换限次
func (cols Collections) QuestionLimit() *mgo.Collection {
	return cols.db.C("questionlimit")
}

// Answers 答案
func (cols Collections) Answers() *mgo.Collection {
	return cols.db.C("answers")
}

// AnswerExchange 交换记录
func (cols Collections) AnswerExchange() *mgo.Collection {
	return cols.db.C("answerexchange")
}

// AnswerExchangeUserList 某用户所有交换过的用户
func (cols Collections) AnswerExchangeUserList() *mgo.Collection {
	return cols.db.C("answerexchangeuserlist")
}

// AnswerExchangeAtUser 某用户对某用户的所有交换的答案
func (cols Collections) AnswerExchangeAtUser() *mgo.Collection {
	return cols.db.C("answerexchangeatuser")
}

// AnswerCommentMap 答案评论
func (cols Collections) AnswerCommentMap() *mgo.Collection {
	return cols.db.C("answercommentmap")
}

// AnswerLikes 答案点赞
func (cols Collections) AnswerLikes() *mgo.Collection {
	return cols.db.C("answerlikes")
}

// UserAnswerCounters 某个用户的交换和答案计数器
func (cols Collections) UserAnswerCounters() *mgo.Collection {
	return cols.db.C("useranswercounters")
}

// WeChat 微信ssk
func (cols Collections) WeChat() *mgo.Collection {
	return cols.db.C("wechat")
}

// FormIDs 模板 id
func (cols Collections) FormIDs() *mgo.Collection {
	return cols.db.C("formids")
}

// Invites 回答邀请
func (cols Collections) Invites() *mgo.Collection {
	return cols.db.C("invites")
}

// InvitePermits 回答邀请
func (cols Collections) InvitePermits() *mgo.Collection {
	return cols.db.C("invitepermits")
}

// InviteBadgeNumber 邀请提醒小红点
func (cols Collections) InviteBadgeNumber() *mgo.Collection {
	return cols.db.C("invitebadgenumber")
}

// Version 数据库版本
func (cols Collections) Version() *mgo.Collection {
	return cols.db.C("version")
}

// Groups 群答案
func (cols Collections) Groups() *mgo.Collection {
	return cols.db.C("groups")
}

// ShakeQuestions 摇到过的问题
func (cols Collections) ShakedQuestions() *mgo.Collection {
	return cols.db.C("shakedquestions")
}

// SkippedQuestions 首页跳过的问题
func (cols Collections) SkippedQuestions() *mgo.Collection {
	return cols.db.C("skippedquestions")
}

// Follows 加星
func (cols Collections) Follows() *mgo.Collection {
	return cols.db.C("follows")
}

// Activitys 动态
func (cols Collections) Activitys() *mgo.Collection {
	return cols.db.C("activitys")
}

type version struct {
	V int
}
