package group

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/draw"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"dagong.in/delphi-web-server/routers/wechat"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type exchangeRequest struct {
	detailRequest
	Content string `binding:"required"`
}

// Exchange 群里答案交换
func Exchange(c *gin.Context) {

	var params exchangeRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	// 查询答案是否存在
	answer, err := routerUtils.FindAnswer(cols, bson.ObjectIdHex(params.Answer))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 是否是答案的创建者
	if answer.OpenID == auth.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "不能和自己交换答案"))
		return
	}

	// 查询问题是否存在
	question, err := routerUtils.FindQuestion(cols, answer.QuestionID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, auth.OpenID)
	if answered != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "已经回答过此问题"))
		return
	}

	now := time.Now()
	a := models.Answer{
		ID:               bson.NewObjectId(),
		OpenID:           auth.OpenID,
		QuestionID:       question.ID,
		Content:          params.Content,
		ExchangeCount:    1,
		AllExchangeCount: 1,
		Status:           models.StatusNormal,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	qa := models.QADisplay{Question: *question, Answer: models.AnswerDisplay{
		Answer: a,
		User:   auth.ToLean(),
	}}

	// 上传答案分享图片
	image, err := draw.GenWeChatShareImageOfDraw(qa)
	if err != nil {
		log.Println(err)
	} else {
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
		} else {
			a.WeChatShareImage = key
			a.WeChatShareImageStatus = models.WeChatShareImageStatusGenerated
		}
	}

	// 插入答案
	answerCol := cols.Answers()
	if err := answerCol.Insert(&a); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// 更新问题答案数
	questionCol := cols.Questions()
	if err := questionCol.Update(bson.M{"_id": question.ID}, bson.M{"$inc": bson.M{"answercount": 1, "allexchangecount": 1}}); err != nil {
		log.Println(err)
	}

	// 更新群数据
	groupCol := cols.Groups()
	push := bson.M{
		"answers": models.GroupAnswer{
			ID:        a.ID,
			OpenID:    a.OpenID,
			CreatedAt: now,
		},
		"addusers": a.OpenID,
	}
	err = groupCol.Update(bson.M{"groupid": params.Group, "questionid": question.ID}, bson.M{"$push": push})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// 更新 activitys
	activitysCol := cols.Activitys()
	activity := models.Feed{
		OpenID:       auth.OpenID,
		Type:         0,
		Question:     question.ID,
		Answer:       a.ID,
		AnswerOpenID: a.OpenID,
		CreatedAt:    now,
	}
	activity1 := models.Feed{
		OpenID:       answer.OpenID,
		Type:         1,
		Question:     question.ID,
		Answer:       a.ID,
		AnswerOpenID: a.OpenID,
		CreatedAt:    now,
	}
	activity2 := models.Feed{
		OpenID:       a.OpenID,
		Type:         1,
		Question:     question.ID,
		Answer:       answer.ID,
		AnswerOpenID: answer.OpenID,
		CreatedAt:    now,
	}
	activitysCol.Insert(activity, activity1, activity2)

	// 更新粉丝的 activitys
	followsCol := cols.Follows()
	follow := models.Follows{}
	followsCol.Find(bson.M{"openid": auth.OpenID}).One(&follow)

	var activitys []interface{}
	for _, v := range follow.Followers {
		activity := models.Feed{
			OpenID:       v.OpenID,
			Type:         2,
			Question:     question.ID,
			Answer:       a.ID,
			AnswerOpenID: a.OpenID,
			CreatedAt:    now,
		}
		activitys = append(activitys, activity)
	}

	if len(activitys) > 0 {
		activitysCol.Insert(activitys...)
	}

	// 更新交换统计数据
	answerExchangeUserListCol := cols.AnswerExchangeUserList()
	err = routerUtils.UpdateExchangeUserList(answerExchangeUserListCol, question.ID, *answer, a)
	if err != nil {
		log.Println(err)
	}

	err = routerUtils.UpdateExchangeUserList(answerExchangeUserListCol, question.ID, a, *answer)
	if err != nil {
		log.Println(err)
	}

	answerExchangeAtUserCol := cols.AnswerExchangeAtUser()
	err = routerUtils.UpdateExchangeAtUser(answerExchangeAtUserCol, question.ID, *answer, a)
	if err != nil {
		log.Println(err)
	}

	err = routerUtils.UpdateExchangeAtUser(answerExchangeAtUserCol, question.ID, a, *answer)
	if err != nil {
		log.Println(err)
	}

	userAnswerCountersCol := cols.UserAnswerCounters()
	userAnswerCountersCol.Upsert(bson.M{"openid": a.OpenID}, bson.M{"$inc": bson.M{
		"answercount":      1,
		"exchangecount":    1,
		"allexchangecount": 1,
	}})

	userAnswerCountersCol.Upsert(bson.M{"openid": answer.OpenID}, bson.M{"$inc": bson.M{
		"beexchangedcount": 1,
		"allexchangecount": 1,
	}})

	exchangeCol := cols.AnswerExchange()
	exchange := models.AnswerExchange{
		ID:          bson.NewObjectId(),
		Question:    question.ID,
		Source:      answer.ID,
		SourceUser:  answer.OpenID,
		Swapper:     a.ID,
		SwapperUser: a.OpenID,
		CreatedAt:   now,
	}

	// 更新交换数据
	if err := exchangeCol.Insert(&exchange); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// 发送模板消息
	msg := wechat.ExchangedNotify{
		Content: fmt.Sprintf("点击查看：%s 回答了 问题 %s", auth.Nickname, question.Content),
		Date:    time.Now().Format("2006/01/02 - 15:04:05"),
	}

	if err := wechat.SendNotify(answer.OpenID, "pages/answer?id="+a.ID.Hex(), msg); err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, routers.OK())
}

type exchangeWithAnsweredRequest struct {
	detailRequest
}

func ExchangeWithAnswered(c *gin.Context) {

	var params exchangeWithAnsweredRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	answer, err := routerUtils.FindAnswer(cols, bson.ObjectIdHex(params.Answer))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if answer.OpenID == auth.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "不能和自己交换答案"))
		return
	}

	question, err := routerUtils.FindQuestion(cols, answer.QuestionID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, auth.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	isJoinedQuestion := isJoinedQuestion(cols, params.Group, question.ID, auth.OpenID)
	if isJoinedQuestion != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "已经交换过答案"))
		return
	}

	groupCol := cols.Groups()
	push := bson.M{
		"answers": models.GroupAnswer{
			ID:        answered.ID,
			OpenID:    answered.OpenID,
			CreatedAt: time.Now(),
		},
		"addusers": answered.OpenID,
	}
	err = groupCol.Update(bson.M{"groupid": params.Group, "questionid": question.ID}, bson.M{"$push": push})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// 判断是否单独交换过
	if routerUtils.IsExchanged(cols, question.ID, answer.OpenID, auth.OpenID) == nil {
		// 更新交换次数
		answerCol := cols.Answers()
		answerCol.Update(bson.M{"_id": answer.ID}, bson.M{"$inc": bson.M{"beexchangedcount": 1, "allexchangecount": 1}})
		answerCol.Update(bson.M{"_id": answered.ID}, bson.M{"$inc": bson.M{"exchangecount": 1, "allexchangecount": 1}})

		exchangeCol := cols.AnswerExchange()
		now := time.Now()

		exchange := models.AnswerExchange{
			ID:          bson.NewObjectId(),
			Question:    question.ID,
			Source:      answer.ID,
			SourceUser:  answer.OpenID,
			Swapper:     answered.ID,
			SwapperUser: answered.OpenID,
			CreatedAt:   now,
		}

		// 插入交换数据
		if err := exchangeCol.Insert(&exchange); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}

		// 更新activitys
		activitysCol := cols.Activitys()
		activity := models.Feed{
			OpenID:       answer.OpenID,
			Type:         1,
			Question:     question.ID,
			Answer:       answered.ID,
			AnswerOpenID: answered.OpenID,
			CreatedAt:    now,
		}
		activity2 := models.Feed{
			OpenID:       answered.OpenID,
			Type:         1,
			Question:     question.ID,
			Answer:       answer.ID,
			AnswerOpenID: answer.OpenID,
			CreatedAt:    now,
		}
		activitysCol.Insert(activity, activity2)

		// 更新交换统计数据
		answerExchangeUserListCol := cols.AnswerExchangeUserList()
		err = routerUtils.UpdateExchangeUserList(answerExchangeUserListCol, question.ID, *answer, *answered)
		if err != nil {
			log.Println(err)
		}

		err = routerUtils.UpdateExchangeUserList(answerExchangeUserListCol, question.ID, *answered, *answer)
		if err != nil {
			log.Println(err)
		}

		answerExchangeAtUserCol := cols.AnswerExchangeAtUser()
		err = routerUtils.UpdateExchangeAtUser(answerExchangeAtUserCol, question.ID, *answer, *answered)
		if err != nil {
			log.Println(err)
		}

		err = routerUtils.UpdateExchangeAtUser(answerExchangeAtUserCol, question.ID, *answered, *answer)
		if err != nil {
			log.Println(err)
		}

		userAnswerCountersCol := cols.UserAnswerCounters()
		userAnswerCountersCol.Upsert(bson.M{"openid": answered.OpenID}, bson.M{"$inc": bson.M{
			"exchangecount":    1,
			"allexchangecount": 1,
		}})

		userAnswerCountersCol.Upsert(bson.M{"openid": answer.OpenID}, bson.M{"$inc": bson.M{
			"beexchangedcount": 1,
			"allexchangecount": 1,
		}})
	}

	// 发送模板消息
	msg := wechat.ExchangedNotify{
		Content: fmt.Sprintf("点击查看：%s 回答了 问题 %s", auth.Nickname, question.Content),
		Date:    time.Now().Format("2006/01/02 - 15:04:05"),
	}
	if err := wechat.SendNotify(answer.OpenID, "pages/answer?id="+answered.ID.Hex(), msg); err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, routers.OK())
}

// isJoinedQuestion 是否在群里交换过
func isJoinedQuestion(cols db.Collections, groupID string, question bson.ObjectId, openid string) *models.AnswerExchange {
	groupCol := cols.Groups()
	var exchanged *models.AnswerExchange
	groupCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"groupid": groupID, "questionid": question}},
		bson.M{"$unwind": "$answers"},
		bson.M{"$match": bson.M{"answers.openid": openid}},
	}).One(&exchanged)
	return exchanged
}
