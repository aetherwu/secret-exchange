package answer

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
	ID     string `binding:"required"`
	Answer string `binding:"required"`
}

// Exchange 创建一个新的答案并交换
func Exchange(c *gin.Context) {

	var params createRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	// 查询问题和答案是否存在
	question, answer, err := routerUtils.FindQA(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 不存在
	if question == nil || answer == nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 不能和自己交换答案
	if answer.OpenID == auth.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "不能和自己交换答案"))
		return
	}

	// 查询是否已经交换过答案
	if routerUtils.IsExchanged(cols, question.ID, answer.OpenID, auth.OpenID) != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "已经交换过答案"))
		return
	}

	// 检查 Privacy 设置 ?

	now := time.Now()
	a := models.Answer{
		ID:               bson.NewObjectId(),
		OpenID:           auth.OpenID,
		QuestionID:       question.ID,
		Content:          params.Answer,
		ExchangeCount:    1,
		AllExchangeCount: 1,
		Status:           models.StatusNormal,
		Privacy:          models.AnswerPrivacyOpen,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	qa := models.QADisplay{Question: *question, Answer: models.AnswerDisplay{
		Answer: a,
		User:   auth.ToLean(),
	}}

	// 上传分享图片
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

	answerCol := cols.Answers()
	if err := answerCol.Insert(&a); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// 更新问题答案数
	questionCol := cols.Questions()
	if err := questionCol.Update(bson.M{"_id": question.ID}, bson.M{"$inc": bson.M{"answercount": 1}}); err != nil {
		log.Println(err)
	}

	// 更新问题被交换次数
	if err := answerCol.Update(bson.M{"_id": answer.ID}, bson.M{"$inc": bson.M{"beexchangedcount": 1, "allexchangecount": 1}}); err != nil {
		log.Println(err)
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

	// 更新 activitys

	//
	activitysCol := cols.Activitys()

	// current user
	// my created answer
	// 向自己的动态插入自己的回答，仅自己可见，无需隐私控制
	activity := models.Feed{
		OpenID:       auth.OpenID,
		Type:         0,
		Question:     question.ID,
		Answer:       a.ID,
		AnswerOpenID: a.OpenID,
		CreatedAt:    now,
	}
	activitysCol.Insert(activity)

	// should  ignore personal question (models.AnswerPrivacyFriend<2)

	// original answer auther
	// collected answer
	// 新的答案
	// 向被交换的用户动态里插入交换动态
	activity = models.Feed{
		OpenID:       answer.OpenID,
		Type:         1,
		Question:     question.ID,
		Answer:       a.ID,
		AnswerOpenID: a.OpenID,
		CreatedAt:    now,
	}

	/*
		// 会导致两条“回答-交换“冗余，所以这里不再插入”交换“信息。
		// new anwser
		// my collected answer
		activity2 := models.Feed{
			OpenID:       a.OpenID,
			Type:         1,
			Question:     question.ID,
			Answer:       answer.ID,
			AnswerOpenID: answer.OpenID,
			CreatedAt:    now,
		}
		activitysCol.Insert(activity, activity2)
	*/

	// 查询粉丝
	followsCol := cols.Follows()
	follow := models.Follows{}
	followsCol.Find(bson.M{"openid": auth.OpenID}).One(&follow)

	// 向粉丝们广播新的交换信息
	if question.Level < 2 {
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

		// 给粉丝的feed插入数据
		if len(activitys) > 0 {
			activitysCol.Insert(activitys...)
		}
	}

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

	c.JSON(http.StatusOK, gin.H{"id": a.ID.Hex()})
}

// ExchangeWithAnswer 交换已经创建好的答案
func ExchangeWithAnswer(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	// 判断 问题和答案是否存在
	question, answer, err := routerUtils.FindQA(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if question == nil || answer == nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if answer.OpenID == auth.OpenID {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "不能和自己交换答案"))
		return
	}

	exchanged := routerUtils.IsExchanged(cols, question.ID, answer.OpenID, auth.OpenID)
	if exchanged != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "已经交换过答案"))
		return
	}

	// 我回答过的答案
	answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, auth.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "我还没有回答过"))
		return
	}

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
	// 仅向 follower 广播新的回答，所以这里不做广播

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

	// 发送模板消息
	msg := wechat.ExchangedNotify{
		Content: fmt.Sprintf("点击查看：%s 回答了 问题 %s", auth.Nickname, question.Content),
		Date:    time.Now().Format("2006/01/02 - 15:04:05"),
	}
	if err := wechat.SendNotify(answer.OpenID, "pages/answer?id="+answered.ID.Hex(), msg); err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, gin.H{"id": answer.ID.Hex()})
}
