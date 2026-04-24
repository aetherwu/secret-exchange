package profile

import (
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/routers/routerUtils"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

// Follow 加星 / 取消加星
func Follow(c *gin.Context) {

	var params routers.OpenIDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	if params.OpenID == auth.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

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

	q := bson.M{"openid": auth.OpenID}
	if follow != nil {
		// 取消加星
		col.Update(q, bson.M{"$pull": bson.M{"follows": bson.M{"openid": params.OpenID}}})
		q = bson.M{"openid": params.OpenID}
		col.Update(q, bson.M{"$pull": bson.M{"followers": bson.M{"openid": auth.OpenID}}})
		go func() {
			removeFollowersActivitys(auth.OpenID, params.OpenID)
		}()
	} else {
		// 加星
		f := models.Follow{OpenID: params.OpenID, CreatedAt: time.Now()}
		col.Upsert(q, bson.M{"$push": bson.M{"follows": f}})
		f = models.Follow{OpenID: auth.OpenID, CreatedAt: time.Now()}
		q = bson.M{"openid": params.OpenID}
		col.Upsert(q, bson.M{"$push": bson.M{"followers": f}})
		go func() {
			addFollowersActivitys(auth.OpenID, params.OpenID)
		}()
	}

	c.JSON(http.StatusOK, routers.OK())
}

// addFollowersActivitys 把加星用户的动态写入自己的动态
func addFollowersActivitys(owner, openID string) {
	session, cols := db.Default()
	defer session.Close()

	activityCol := cols.Activitys()
	answerCol := cols.Answers()
	answers := []models.Answer{}
	answerCol.Find(bson.M{"openid": openID, "status": bson.M{"$ne": models.StatusDelete}}).All(&answers)

	var feeds []interface{}
	for _, v := range answers {
		f := models.Feed{
			OpenID:       owner,
			Question:     v.QuestionID,
			Answer:       v.ID,
			AnswerOpenID: v.OpenID,
			Type:         2,
			CreatedAt:    v.CreatedAt,
		}
		feeds = append(feeds, f)
	}

	if len(feeds) > 0 {
		activityCol.Insert(feeds...)
	}
}

// removeFollowersActivitys 把加星用户的动态从自己的动态里删除
func removeFollowersActivitys(owner, openID string) {
	session, cols := db.Default()
	defer session.Close()

	activityCol := cols.Activitys()
	activityCol.RemoveAll(bson.M{"openid": owner, "type": 2, "answeropenid": openID})
}

// Follows 加星列表
func Follows(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	col := cols.Follows()

	follow := models.Follows{}
	col.Find(bson.M{"openid": auth.OpenID}).One(&follow)

	openids := make([]string, len(follow.Follows))
	for i, v := range follow.Follows {
		openids[i] = v.OpenID
	}

	users, err := routerUtils.FindUsersAsMap(cols, openids)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	answerExchangeUserListCol := cols.AnswerExchangeUserList()
	exchanges := []answerExchangeUserListGroup{}
	answerExchangeUserListCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID}},
		bson.M{"$unwind": "$exchanges"},
		bson.M{"$match": bson.M{"exchanges.openid": bson.M{"$in": openids}}},
	}).All(&exchanges)

	exchangeMap := map[string]models.AnswerExchangeUser{}
	for _, v := range exchanges {
		exchangeMap[v.Exchanges.OpenID] = v.Exchanges
	}

	respUsers := []profileListUser{}
	for _, v := range follow.Follows {
		openid := v.OpenID
		if u, ok := users[openid]; ok {
			if e, ok := exchangeMap[openid]; ok {
				r := profileListUser{
					LeanUser:       u.ToLean(),
					IsFollow:       true,
					ExchangeCount:  e.Count,
					LastAnsweredAt: e.UpdatedAt,
				}
				respUsers = append(respUsers, r)
			}
		}
	}

	c.JSON(http.StatusOK, RecentListResponse{Users: respUsers})
}
