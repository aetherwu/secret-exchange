package question

import (
	"log"
	"net/http"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

// Detail 问题详情
func Detail(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	result, err := routerUtils.FindQuestion(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	c.JSON(http.StatusOK, result)
}

// DetailWithAnswer 根据回答返回问题的详情
func DetailWithAnswer(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	answer, err := routerUtils.FindAnswer(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	question, err := routerUtils.FindQuestion(cols, answer.QuestionID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	c.JSON(http.StatusOK, question)
}

type randomResponse struct {
	Question             *models.Question `json:"question"`
	Limit                int              `json:"limit"`
	AnswerCount          int              `json:"answerCount"`
	QuestionCount        int              `json:"questionCount"`
	ExchangedAnswerCount int              `json:"exchangedAnswerCount"`
	ExchangedUserCount   int              `json:"exchangedUserCount"`
	FormIDCount          int              `json:"formIDCount"`
	TapEnabled           bool             `json:"tapEnabled"`
}

// Random 随机返回一个问题
func Random(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	limitCol := cols.QuestionLimit()
	questionLimit := models.QuestionLimit{}
	limitCol.Find(bson.M{"openid": auth.OpenID}).One(&questionLimit)

	// 检查今天答题是否超过5次
	limit := 5
	if questionLimit.Limit != 0 {
		limit = 5 - questionLimit.Limit
	}

	questionCol := cols.Questions()
	answerCol := cols.Answers()

	// 获取 formID 数
	formIDCount, _ := cols.FormIDs().Find(bson.M{"user": auth.OpenID}).Count()

	resp := randomResponse{Limit: limit, FormIDCount: formIDCount}

	// 获取好友数
	answerExchangeUserListCol := cols.AnswerExchangeUserList()
	answerExchangeUserList := bson.M{}
	answerExchangeUserListCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID}},
		bson.M{"$project": bson.M{"count": bson.M{"$size": "$exchanges"}}},
	}).One(&answerExchangeUserList)

	if count, ok := answerExchangeUserList["count"].(int); ok {
		resp.ExchangedUserCount = count
		resp.TapEnabled = count >= 5
	}

	// 今天还能答题
	if limit > 0 {
		// 获取所有问题
		questions := []models.Question{}
		if err := questionCol.Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).All(&questions); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
			return
		}

		quesitionIDs := []bson.ObjectId{}
		questionMap := map[bson.ObjectId]models.Question{}

		for _, v := range questions {
			quesitionIDs = append(quesitionIDs, v.ID)
			questionMap[v.ID] = v
		}

		// 获取我的所有答案
		answers := []models.Answer{}
		if err := answerCol.Find(bson.M{"openid": auth.OpenID}).All(&answers); err != nil {
			log.Println(err)
		}

		answeredQuestionIDs := []bson.ObjectId{}
		for _, v := range answers {
			answeredQuestionIDs = append(answeredQuestionIDs, v.QuestionID)
		}

		// 没有回答过的问题
		notAnsweredQuestionIDs := subtracting(quesitionIDs, answeredQuestionIDs)
		if len(notAnsweredQuestionIDs) == 0 {
			c.JSON(http.StatusOK, resp)
			return
		}

		notAnsweredQuestionIDs1 := []bson.ObjectId{}
		for _, v := range notAnsweredQuestionIDs {
			if q, ok := questionMap[v]; ok {
				notAnsweredQuestionIDs1 = append(notAnsweredQuestionIDs1, q.ID)
			}
		}

		// 获取跳过的问题
		skippedQuestionsCol := cols.SkippedQuestions()
		skippedQuestions := models.SkippedQuestions{}
		skippedQuestionsCol.Find(bson.M{"openid": auth.OpenID}).One(&skippedQuestions)

		// 没有答案并且没跳过的问题
		notAnsweredQuestionIDs = subtracting(notAnsweredQuestionIDs1, skippedQuestions.Questions)

		var r bson.ObjectId
		l := len(notAnsweredQuestionIDs)
		// 没有问题了
		if l == 0 {
			// 重置所有跳过的问题
			skippedQuestions.Questions = []bson.ObjectId{}
			skippedQuestionsCol.Upsert(bson.M{"openid": auth.OpenID}, bson.M{
				"$set": bson.M{"questions": skippedQuestions.Questions},
			})
			// 重新获取没有答案并且没跳过的问题
			notAnsweredQuestionIDs = subtracting(notAnsweredQuestionIDs1, skippedQuestions.Questions)
			l = len(notAnsweredQuestionIDs)
			// 题库里所有问题都答完了
			if l == 0 {
				c.JSON(http.StatusOK, resp)
				return
			} else if l == 1 {
				r = notAnsweredQuestionIDs[0]
			} else {
				next, err := routerUtils.GetQuestionsWeightRandomNext(questionCol, notAnsweredQuestionIDs)
				if err != nil {
					log.Println(err)
					c.JSON(http.StatusOK, resp)
					return
				}
				r = *next
			}

			if question, ok := questionMap[r]; ok {
				resp.Question = &question
				// 更新跳过的问题
				skippedQuestionsCol.Upsert(bson.M{"openid": auth.OpenID}, bson.M{"$push": bson.M{"questions": question.ID}})
			}
			// 返回
			c.JSON(http.StatusOK, resp)
			return
		} else if l == 1 {
			r = notAnsweredQuestionIDs[0]
		} else {
			next, err := routerUtils.GetQuestionsWeightRandomNext(questionCol, notAnsweredQuestionIDs)
			if err != nil {
				log.Println(err)
				c.JSON(http.StatusOK, resp)
				return
			}
			r = *next
		}

		if question, ok := questionMap[r]; ok {
			resp.Question = &question
			// 更新跳过的问题
			skippedQuestionsCol.Upsert(bson.M{"openid": auth.OpenID}, bson.M{"$push": bson.M{"questions": question.ID}})
		} else {
			// 返回
			c.JSON(http.StatusOK, resp)
			return
		}
	} else { // 今天的答题次数用完了
		// 查询题库里所有问题数
		questionCount, err := questionCol.Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).Count()
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}
		resp.QuestionCount = questionCount

		// 查询我答过的所有问题数
		answerCount, err := answerCol.Find(bson.M{"openid": auth.OpenID, "status": bson.M{"$ne": models.StatusDelete}}).Count()
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}
		resp.AnswerCount = answerCount

		// 查询交换次数
		exchangeCol := cols.AnswerExchange()
		or := []bson.M{bson.M{"sourceuser": auth.OpenID}, bson.M{"swapperuser": auth.OpenID}}
		exchangedAsnwerCount, err := exchangeCol.Find(bson.M{"$or": or}).Count()
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}
		resp.ExchangedAnswerCount = exchangedAsnwerCount
	}

	c.JSON(http.StatusOK, resp)
}

func subtracting(l, r []bson.ObjectId) []bson.ObjectId {
	if len(r) == 0 {
		return l
	}
	rs := []bson.ObjectId{}
	m := make(map[bson.ObjectId]bool)

	for _, v := range r {
		m[v] = true
	}

	for _, v := range l {
		if _, ok := m[v]; !ok {
			rs = append(rs, v)
		}
	}

	return rs
}
