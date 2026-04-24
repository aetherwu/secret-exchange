package dashboard

import (
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/routers/routerUtils"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

// EditAnswer 编辑答案
func EditAnswer(c *gin.Context) {

	var params editRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	col := cols.Answers()

	set := bson.M{"updatedat": time.Now(), "content": params.Content}
	err := col.Update(bson.M{"_id": bson.ObjectIdHex(params.ID)}, bson.M{"$set": set})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	c.JSON(http.StatusOK, routers.OK())
}

// DeleteAnswer 删除答案
func DeleteAnswer(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	answerID := bson.ObjectIdHex(params.ID)

	answerCol := cols.Answers()
	answer := models.Answer{}
	err := answerCol.Find(bson.M{"_id": answerID}).One(&answer)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// questionCol := cols.Questions()
	// questionCol.Update(bson.M{"_id": answer.QuestionID}, bson.M{"$inc": bson.M{"answercount": -1}})

	// userAnswerCountersCol := cols.UserAnswerCounters()
	// userAnswerCountersCol.Update(bson.M{"openid": answer.OpenID}, bson.M{"$inc": bson.M{"answercount": -1}})

	// answerExchangeAtUserCol := cols.AnswerExchangeAtUser()
	// or := bson.M{"$or": []bson.M{bson.M{"openid": answer.OpenID}, bson.M{"user": answer.OpenID}}}
	// answerExchangeAtUserCol.Update(or, bson.M{"$pull": bson.M{"answers": bson.M{"answerid": answer.ID}}})

	set := bson.M{"status": models.StatusDelete}
	err = answerCol.Update(bson.M{"_id": answerID}, bson.M{"$set": set})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	exchangeCol := cols.AnswerExchange()
	err = exchangeCol.Remove(bson.M{"source": params.ID})
	if err != nil {
		log.Println(err)
	}

	err = exchangeCol.Remove(bson.M{"swapper": params.ID})
	if err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, routers.OK())
}

// Answers 答案列表
func Answers(c *gin.Context) {

	var params queryRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	answersCol := cols.Answers()

	limit := 20

	sort := "createdat"
	if params.OrderBy == "allExchangeCount" {
		sort = "allexchangecount"
	}

	sortDirection := -1
	if params.OrderDirection == 1 {
		sortDirection = 1
	}

	skip := (params.Page - 1) * limit
	if skip < 0 {
		skip = 0
	}

	total, err := answersCol.Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).Count()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answers := []models.Answer{}
	err = answersCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"status": bson.M{"$ne": models.StatusDelete}}},
		bson.M{"$sort": bson.M{sort: sortDirection}},
		bson.M{"$skip": skip},
		bson.M{"$limit": limit},
	}).All(&answers)

	if err != nil {
		log.Println(err)
	}

	questionsIDs := make([]bson.ObjectId, len(answers))
	userIDs := make([]string, len(answers))
	for i, v := range answers {
		questionsIDs[i] = v.QuestionID
		userIDs[i] = v.OpenID
	}

	questionsIDs = utils.ObjectIDRemoveRep(questionsIDs)
	userIDs = utils.StringRemoveRep(userIDs)

	questionCol := cols.Questions()
	questions := []models.Question{}
	questionCol.Find(bson.M{"_id": bson.M{"$in": questionsIDs}}).All(&questions)

	questionMap := map[bson.ObjectId]models.Question{}
	for _, v := range questions {
		questionMap[v.ID] = v
	}

	users, err := routerUtils.FindUsersAsMap(cols, userIDs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	results := []models.QADisplay{}
	for _, v := range answers {
		if q, ok := questionMap[v.QuestionID]; ok {
			if u, ok := users[v.OpenID]; ok {
				results = append(results, models.QADisplay{
					Question: q,
					Answer:   models.AnswerDisplay{Answer: v, User: u.ToLean()},
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "answers": results})
}
