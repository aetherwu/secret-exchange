package profile

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"dagong.in/delphi-web-server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type profileListUser struct {
	models.LeanUser
	IsFollow       bool      `json:"isFollow"`
	ExchangeCount  int       `json:"exchangeCount"`
	LastAnsweredAt time.Time `json:"lastAnsweredAt"`
}

type RecentListResponse struct {
	Users []profileListUser `json:"users"`
}

type answerExchangeUserListGroup struct {
	ID        bson.ObjectId `json:"id" bson:"_id,omitempty"`
	OpenID    string
	Exchanges models.AnswerExchangeUser
}

// RecentList 个人主页  tab -  最近交换联系人
// pages/profile-list
// Requets: /profile/list
func RecentList(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	col := cols.AnswerExchangeUserList()
	exchanges := []answerExchangeUserListGroup{}
	if err := col.Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID}},
		bson.M{"$unwind": "$exchanges"},
		bson.M{"$sort": bson.M{"exchanges.updatedat": -1}},
	}).All(&exchanges); err != nil {
		log.Println(err)
	}

	userIDs := make([]string, len(exchanges))
	for i, v := range exchanges {
		userIDs[i] = v.Exchanges.OpenID
	}

	users, err := routerUtils.FindUsersAsMap(cols, userIDs)
	if err != nil {
		log.Println(err)
	}

	follow := models.Follows{}
	followCol := cols.Follows()
	followCol.Find(bson.M{"openid": auth.OpenID}).One(&follow)
	followMap := map[string]bool{}
	for _, v := range follow.Follows {
		followMap[v.OpenID] = true
	}

	respUsers := []profileListUser{}
	for _, v := range exchanges {
		openid := v.Exchanges.OpenID
		if u, ok := users[openid]; ok {
			r := profileListUser{
				LeanUser:       u.ToLean(),
				ExchangeCount:  v.Exchanges.Count,
				LastAnsweredAt: v.Exchanges.UpdatedAt,
			}
			if f, ok := followMap[openid]; ok {
				r.IsFollow = f
			}
			respUsers = append(respUsers, r)
		}
	}

	c.JSON(http.StatusOK, RecentListResponse{Users: respUsers})
}

// ProfileListMost 个人主页 交换最多
// pages/most
func ProfileListMost(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	col := cols.AnswerExchangeUserList()
	exchanges := []answerExchangeUserListGroup{}
	if err := col.Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID}},
		bson.M{"$unwind": "$exchanges"},
		bson.M{"$sort": bson.M{"exchanges.count": -1}},
	}).All(&exchanges); err != nil {
		log.Println(err)
	}

	userIDs := make([]string, len(exchanges))
	for i, v := range exchanges {
		userIDs[i] = v.Exchanges.OpenID
	}

	users, err := routerUtils.FindUsersAsMap(cols, userIDs)
	if err != nil {
		log.Println(err)
	}

	follow := models.Follows{}
	followCol := cols.Follows()
	followCol.Find(bson.M{"openid": auth.OpenID}).One(&follow)
	followMap := map[string]bool{}
	for _, v := range follow.Follows {
		followMap[v.OpenID] = true
	}

	respUsers := []profileListUser{}
	for _, v := range exchanges {
		openid := v.Exchanges.OpenID
		if u, ok := users[openid]; ok {
			r := profileListUser{
				LeanUser:       u.ToLean(),
				ExchangeCount:  v.Exchanges.Count,
				LastAnsweredAt: v.Exchanges.UpdatedAt,
			}
			if f, ok := followMap[openid]; ok {
				r.IsFollow = f
			}
			respUsers = append(respUsers, r)
		}
	}

	c.JSON(http.StatusOK, RecentListResponse{Users: respUsers})
}

type profileResponse struct {
	User                   models.LeanUser `json:"user"`
	Owner                  models.LeanUser `json:"owner"`
	ExchangeCount          int             `json:"exchangeCount"`
	LatelyExchanges        []models.QA     `json:"latelyExchanges"`
	FreeAnswers            []models.QA     `json:"freeAnswers"`
	TotalQuestionCount     int             `json:"totalQuestionCount"`
	WeChatShareImageStatus int             `json:"weChatShareImageStatus"`
	WeChatShareImage       string          `json:"weChatShareImage"`
	IsFollowing            int             `json:"isFollowing"`
}

type answerExchangeAtUserGroup struct {
	ID        bson.ObjectId `json:"id" bson:"_id,omitempty"`
	OpenID    string
	User      string
	Answers   models.AnswerExchangeAtUserDetail
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Profile - browse other's profile
// /profile
// pages/profile
func Profile(c *gin.Context) {

	// bind request
	var params routers.OpenIDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	// get authed user
	auth := routers.GetContextAuth(c)

	// get db & collection
	session, cols := db.Default()
	defer session.Close()

	// check auth user exsiting
	user, err := routerUtils.FindUser(cols, params.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// get mutual exchanged answers between browser and target user
	answerExchangeAtUserCol := cols.AnswerExchangeAtUser()
	exchanges := []answerExchangeAtUserGroup{}

	if err := answerExchangeAtUserCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID, "user": params.OpenID}},
		bson.M{"$unwind": "$answers"},
		bson.M{"$sort": bson.M{"answers.createdat": -1}},
	}).All(&exchanges); err != nil {
		log.Println(err)
	}

	// get availavble answers (all answers - except complete private answers - deleted)
	answerCol := cols.Answers()
	freeAnswers := []models.Answer{}
	answerCol.Find(bson.M{
		"openid":  params.OpenID,
		"privacy": bson.M{"$lt": models.AnswerPrivacyFriend},
		"status":  bson.M{"$ne": models.StatusDelete},
	}).Sort("-createdat").All(&freeAnswers)

	// struct response
	resp := profileResponse{
		Owner:           auth.ToLean(),
		User:            user.ToLean(),
		LatelyExchanges: []models.QA{},
		FreeAnswers:     []models.QA{},
	}

	// restruct question IDs and answer IDs
	questionIDs := make([]bson.ObjectId, len(exchanges))
	answerIDs := make([]bson.ObjectId, len(exchanges))
	for i, v := range exchanges {
		questionIDs[i] = v.Answers.QuestionID
		answerIDs[i] = v.Answers.AnswerID
	}

	for _, v := range freeAnswers {
		questionIDs = append(questionIDs, v.QuestionID)
		answerIDs = append(answerIDs, v.ID)
	}

	// remove dulpliate IDs
	questionIDs = utils.ObjectIDRemoveRep(questionIDs)
	answerIDs = utils.ObjectIDRemoveRep(answerIDs)

	// get question data via question IDs
	questionCol := cols.Questions()
	questions := []models.Question{}
	if err := questionCol.Find(bson.M{"_id": bson.M{"$in": questionIDs}}).All(&questions); err != nil {
		log.Println(err)
	}

	// remap question arary/set
	questionMap := map[bson.ObjectId]models.Question{}
	for _, v := range questions {
		questionMap[v.ID] = v
	}

	// get answer data via answer IDs
	answers := []models.Answer{}
	if err := answerCol.Find(bson.M{"_id": bson.M{"$in": answerIDs}}).All(&answers); err != nil {
		log.Println(err)
	}

	// remap answer array/set
	answerMap := map[bson.ObjectId]models.Answer{}
	for _, v := range answers {
		v.Length = utf8.RuneCountInString(v.Content)
		answerMap[v.ID] = v
	}

	// recompile question and answer to pairing
	// start with mutually exchanged list
	for _, v := range exchanges {
		// loop through question array
		if q, ok := questionMap[v.Answers.QuestionID]; ok {
			// loop through answer array
			if a, ok := answerMap[v.Answers.AnswerID]; ok {
				q.CreatedAt = v.Answers.CreatedAt
				qa := models.QA{Question: q, Answer: a}
				resp.LatelyExchanges = append(resp.LatelyExchanges, qa)
			}
		}
	}

	// recompile question and answer to pairing
	for _, v := range freeAnswers {
		// loop through question array
		if q, ok := questionMap[v.QuestionID]; ok {
			// loop through answer array
			if a, ok := answerMap[v.ID]; ok {

				// check and add exchanged info
				isExchanged := routerUtils.IsExchanged(cols, q.ID, auth.OpenID, a.OpenID)
				a.IsExchanged = isExchanged != nil
				if !a.IsExchanged {
					a.Content = ""
				}

				qa := models.QA{Question: q, Answer: a}
				resp.FreeAnswers = append(resp.FreeAnswers, qa)
			}
		}
	}

	// return statistics
	exchangeCount := bson.M{}
	err = answerExchangeAtUserCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID, "user": params.OpenID}},
		bson.M{"$project": bson.M{"count": bson.M{"$size": "$answers"}}},
	}).One(&exchangeCount)
	if err != nil {
		// log.Println(err)
		resp.ExchangeCount = 0
	} else {
		resp.ExchangeCount = exchangeCount["count"].(int)
	}

	totalQuestionCount, _ := cols.Questions().Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).Count()
	resp.TotalQuestionCount = totalQuestionCount

	if user.WeChatShareImage != "" {
		resp.WeChatShareImage = routerUtils.QiniuKeyToURL(user.WeChatShareImage)
	}
	resp.WeChatShareImageStatus = user.WeChatShareImageStatus

	// is following?
	col := cols.Follows()
	var follow *models.Follows

	col.Find(bson.M{
		"openid": auth.OpenID,
		"follows": bson.M{
			"$elemMatch": bson.M{
				"openid": params.OpenID,
			},
		},
	}).One(&follow)
	fmt.Println(follow)
	if follow != nil {
		resp.IsFollowing = 1
	} else {
		resp.IsFollowing = 0
	}

	// send response
	c.JSON(http.StatusOK, resp)
}
