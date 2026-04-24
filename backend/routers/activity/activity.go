package activity

import (
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"dagong.in/delphi-web-server/routers/routerUtils"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type listRequest struct {
	LastID string
}

type listResponse struct {
	models.QADisplay
	ID          string    `json:"id"`
	Type        int       `json:"type"`
	IsExchanged bool      `json:"isExchanged"`
	CreatedAt   time.Time `json:"createdAt"`
}

// List Activitys
func List(c *gin.Context) {

	var params listRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	activityCol := cols.Activitys()

	feeds := []models.Feed{}
	query := bson.M{"openid": auth.OpenID}

	// 判断是否需要翻页
	if params.LastID != "" {
		last := models.Feed{}
		lastID := bson.ObjectIdHex(params.LastID)
		err := activityCol.Find(bson.M{"_id": lastID}).One(&last)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
			return
		}

		query["createdat"] = bson.M{"$lte": last.CreatedAt}
		query["_id"] = bson.M{"$ne": lastID}
	}
	// 查询 activitys
	activityCol.Find(query).Sort("-createdat").Limit(20).All(&feeds)

	questionIDs := []bson.ObjectId{}
	answerIDs := []bson.ObjectId{}

	for _, v := range feeds {
		questionIDs = append(questionIDs, v.Question)
		answerIDs = append(answerIDs, v.Answer)
	}

	// 去重
	questionIDs = utils.ObjectIDRemoveRep(questionIDs)
	answerIDs = utils.ObjectIDRemoveRep(answerIDs)

	// 查找问题
	questions := []models.Question{}
	questionCol := cols.Questions()
	questionCol.Find(bson.M{"_id": bson.M{"$in": questionIDs}}).All(&questions)
	questionMap := map[bson.ObjectId]models.Question{}
	for _, v := range questions {
		questionMap[v.ID] = v
	}

	// 查找答案
	answers := []models.Answer{}
	answerCol := cols.Answers()
	answerCol.Find(bson.M{"_id": bson.M{"$in": answerIDs}}).All(&answers)
	answerMap := map[bson.ObjectId]models.Answer{}
	for _, v := range answers {
		v.Length = utf8.RuneCountInString(v.Content)
		answerMap[v.ID] = v
	}

	openIDs := []string{}
	for _, v := range answers {
		openIDs = append(openIDs, v.OpenID)
	}
	// 查找用户
	users, _ := routerUtils.FindUsersAsMap(cols, openIDs)

	results := []listResponse{}

	for _, v := range feeds {
		if q, ok := questionMap[v.Question]; ok {
			if a, ok := answerMap[v.Answer]; ok {
				if u, ok := users[a.OpenID]; ok {
					var isExchanged *models.AnswerExchange
					// 判断是否交换过
					if v.Type == 2 {
						isExchanged = routerUtils.IsExchanged(cols, v.Question, v.OpenID, v.AnswerOpenID)
						if isExchanged == nil {
							a.Content = ""
						}
					}
					r := listResponse{
						QADisplay: models.QADisplay{
							Question: q,
							Answer: models.AnswerDisplay{
								Answer: a,
								User:   u.ToLean(),
							},
						},
						ID:        v.ID.Hex(),
						Type:      v.Type,
						CreatedAt: v.CreatedAt,
					}
					if v.Type == 2 {
						r.IsExchanged = isExchanged != nil
					}
					results = append(results, r)
				}
			}
		}
	}

	c.JSON(http.StatusOK, results)
}
