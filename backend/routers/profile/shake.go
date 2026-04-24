package profile

import (
	"log"
	"net/http"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"dagong.in/delphi-web-server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type shakeResponse struct {
	QA    *models.QA `json:"qa"`
	Limit int        `json:"limit"`
	Type  int        `json:"type"`
}

// Shake 点一点、摇一摇
func Shake(c *gin.Context) {

	var params routers.OpenIDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	user, err := routerUtils.FindUser(cols, params.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	isFriend := false

	answerExchangeAtUserCol := cols.AnswerExchangeAtUser()
	exchangeCount := bson.M{}
	err = answerExchangeAtUserCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID, "user": user.OpenID}},
		bson.M{"$project": bson.M{"count": bson.M{"$size": "$answers"}}},
	}).One(&exchangeCount)
	if err != nil {
		log.Println(err)
	} else {
		isFriend = exchangeCount["count"].(int) >= 5
	}

	// 获取问题
	resp, needResetShaked := getShake(cols, auth, params, isFriend)

	// 循环列表全部都跑完了 清空后重新循环
	if needResetShaked {
		resetShakedQuestions(cols, auth.OpenID, params.OpenID)
		resp, _ = getShake(cols, auth, params, isFriend)
	}

	c.JSON(http.StatusOK, resp)
}

func getShake(cols db.Collections, auth models.User, params routers.OpenIDRequest, isFriend bool) (shakeResponse, bool) {
	resp := shakeResponse{Limit: 6}

	lt := models.AnswerPrivacyMyself
	if !isFriend {
		lt = models.AnswerPrivacyFriend
	}

	answerCol := cols.Answers()
	answers := []models.Answer{}
	answerCol.Find(bson.M{
		"openid":  params.OpenID,
		"privacy": bson.M{"$lt": lt},
		"status":  bson.M{"$ne": models.StatusDelete},
	}).All(&answers)

	answerIDs := make([]bson.ObjectId, len(answers))
	questionIDs := make([]bson.ObjectId, len(answers))
	for i, v := range answers {
		answerIDs[i] = v.ID
		questionIDs[i] = v.QuestionID
	}

	scols := scols{}
	scols.cols = cols
	scols.Answer = answerCol
	scols.AnswerExchangeAtUser = cols.AnswerExchangeAtUser()
	scols.Question = cols.Questions()

	req := request{Auth: auth, OpenID: params.OpenID}

	shakedQuestions := getShakedQuestions(cols, auth.OpenID, params.OpenID)

	qa := getBothNotExchangedAndAnsweredQuestion(scols, req, answers, answerIDs, questionIDs, shakedQuestions)
	if qa != nil {
		setShakedQuestions(cols, auth.OpenID, params.OpenID, qa.Question.ID)
		resp.QA = qa
		resp.Type = 1
		return resp, false
	}

	qa = getHeAnsweredAndMyNotAnsweredQuestion(scols, req, answers, answerIDs, questionIDs, shakedQuestions)
	if qa != nil {
		setShakedQuestions(cols, auth.OpenID, params.OpenID, qa.Question.ID)
		resp.QA = qa
		resp.Type = 2
		return resp, false
	}

	answers = []models.Answer{}
	answerCol.Find(bson.M{
		"openid": auth.OpenID,
		"status": bson.M{"$ne": models.StatusDelete},
	}).All(&answers)

	answerIDs = make([]bson.ObjectId, len(answers))
	questionIDs = make([]bson.ObjectId, len(answers))
	for i, v := range answers {
		answerIDs[i] = v.ID
		questionIDs[i] = v.QuestionID
	}

	qa = getMyAnsweredAndHeNotAnsweredQuestion(scols, req, answers, answerIDs, questionIDs, shakedQuestions)
	if qa != nil {
		setShakedQuestions(cols, auth.OpenID, params.OpenID, qa.Question.ID)
		resp.QA = qa
		resp.Type = 3
		return resp, false
	}

	qa, needResetShaked := getBothNotAnsweredQuestion(scols, req, shakedQuestions)
	if qa != nil {
		setShakedQuestions(cols, auth.OpenID, params.OpenID, qa.Question.ID)
		resp.QA = qa
		resp.Type = 4
		return resp, false
	}

	return resp, needResetShaked
}

// setShakedQuestions 插入问题到循环列表
func setShakedQuestions(cols db.Collections, owner string, toUser string, question bson.ObjectId) error {
	col := cols.ShakedQuestions()
	update := bson.M{
		"$push": bson.M{"questions": question},
	}
	_, err := col.Upsert(bson.M{"openid": owner, "user": toUser}, update)
	return err
}

// getShakedQuestions 获取循环列表
func getShakedQuestions(cols db.Collections, owner string, toUser string) []bson.ObjectId {
	col := cols.ShakedQuestions()
	resp := models.ShakedQuestions{}
	col.Find(bson.M{"openid": owner, "user": toUser}).One(&resp)
	return resp.Questions
}

// resetShakedQuestions 清空循环列表
func resetShakedQuestions(cols db.Collections, owner string, toUser string) error {
	col := cols.ShakedQuestions()
	_, err := col.Upsert(bson.M{"openid": owner, "user": toUser}, bson.M{"$set": bson.M{"questions": []bson.ObjectId{}}})
	return err
}

type scols struct {
	cols                 db.Collections
	Answer               *mgo.Collection
	Question             *mgo.Collection
	AnswerExchangeAtUser *mgo.Collection
}

type request struct {
	Auth   models.User
	OpenID string
}

// getBothNotExchangedAndAnsweredQuestion 双方为交换并且已答
func getBothNotExchangedAndAnsweredQuestion(db scols, req request,
	answers []models.Answer,
	answerIDs, questionIDs, shakedQuestions []bson.ObjectId) *models.QA {

	// 已经交换过
	answerExchangeAtUser := models.AnswerExchangeAtUser{}
	db.AnswerExchangeAtUser.Find(bson.M{"openid": req.OpenID, "user": req.Auth.OpenID}).One(&answerExchangeAtUser)

	exchangedQuestionIDs := make([]bson.ObjectId, len(answerExchangeAtUser.Answers))
	for i, v := range answerExchangeAtUser.Answers {
		exchangedQuestionIDs[i] = v.QuestionID
	}

	// 没有交换过
	notExchangedQuestionIDs := subtracting(questionIDs, exchangedQuestionIDs)
	notExchangedQuestionIDs = utils.ObjectIDRemoveRep(notExchangedQuestionIDs)

	answereds := []models.Answer{}
	db.Answer.Find(bson.M{"openid": req.Auth.OpenID, "questionid": bson.M{"$in": notExchangedQuestionIDs}}).All(&answereds)

	answeredQuestionIDs := make([]bson.ObjectId, len(answereds))
	for i, v := range answereds {
		answeredQuestionIDs[i] = v.QuestionID
	}

	notShakedQuestionIDs := subtracting(answeredQuestionIDs, shakedQuestions)

	var r bson.ObjectId
	len := len(notShakedQuestionIDs)
	if len == 0 {
		return nil
	} else if len == 1 {
		r = notShakedQuestionIDs[0]
	} else {
		next, err := routerUtils.GetQuestionsWeightRandomNext(db.Question, notShakedQuestionIDs)
		if err != nil {
			return nil
		}
		r = *next
	}

	question := &models.Question{}
	db.Question.Find(bson.M{"_id": r}).One(&question)

	if question != nil {
		answer, err := routerUtils.FindAnswerWithQuesiton(db.cols, question.ID, req.OpenID)
		if err != nil {
			return nil
		}
		answer.Content = ""
		return &models.QA{Question: *question, Answer: *answer}
	}

	return nil
}

// getHeAnsweredAndMyNotAnsweredQuestion 他已答，我未答
func getHeAnsweredAndMyNotAnsweredQuestion(db scols, req request,
	answers []models.Answer,
	answerIDs, questionIDs, shakedQuestions []bson.ObjectId) *models.QA {

	myAnswereds := []models.Answer{}
	db.Answer.Find(bson.M{"openid": req.Auth.OpenID, "questionid": bson.M{"$in": questionIDs}}).All(&myAnswereds)
	myAnsweredQuestionIDs := []bson.ObjectId{}
	for _, v := range myAnswereds {
		myAnsweredQuestionIDs = append(myAnsweredQuestionIDs, v.QuestionID)
	}

	myNotAnsweredQuestions := subtracting(questionIDs, myAnsweredQuestionIDs)
	notShakedQuestionIDs := subtracting(myNotAnsweredQuestions, shakedQuestions)

	var r bson.ObjectId
	len := len(notShakedQuestionIDs)
	if len == 0 {
		return nil
	} else if len == 1 {
		r = notShakedQuestionIDs[0]
	} else {
		next, err := routerUtils.GetQuestionsWeightRandomNext(db.Question, notShakedQuestionIDs)
		if err != nil {
			return nil
		}
		r = *next
	}

	var myNotAnsweredQuestion *models.Question
	if err := db.Question.Find(bson.M{"_id": r}).One(&myNotAnsweredQuestion); err != nil {
		return nil
	}

	answer, err := routerUtils.FindAnswerWithQuesiton(db.cols, myNotAnsweredQuestion.ID, req.OpenID)
	if err != nil {
		return nil
	}
	answer.Content = ""
	return &models.QA{Question: *myNotAnsweredQuestion, Answer: *answer}
}

// getMyAnsweredAndHeNotAnsweredQuestion 我已答，他未答
func getMyAnsweredAndHeNotAnsweredQuestion(db scols, req request,
	answers []models.Answer,
	answerIDs, questionIDs, shakedQuestions []bson.ObjectId) *models.QA {

	heAnswereds := []models.Answer{}
	db.Answer.Find(bson.M{"openid": req.OpenID, "questionid": bson.M{"$in": questionIDs}}).All(&heAnswereds)
	heAnsweredQuestionIDs := []bson.ObjectId{}
	for _, v := range heAnswereds {
		heAnsweredQuestionIDs = append(heAnsweredQuestionIDs, v.QuestionID)
	}

	heNotAnsweredQuestions := subtracting(questionIDs, heAnsweredQuestionIDs)
	notShakedQuestionIDs := subtracting(heNotAnsweredQuestions, shakedQuestions)

	len := len(notShakedQuestionIDs)
	var r bson.ObjectId
	if len == 0 {
		return nil
	} else if len == 1 {
		r = notShakedQuestionIDs[0]
	} else {
		next, err := routerUtils.GetQuestionsWeightRandomNext(db.Question, notShakedQuestionIDs)
		if err != nil {
			return nil
		}
		r = *next
	}

	var heNotAnsweredQuestion *models.Question
	if err := db.Question.Find(bson.M{"_id": r}).One(&heNotAnsweredQuestion); err != nil {
		return nil
	}

	answer, err := routerUtils.FindAnswerWithQuesiton(db.cols, heNotAnsweredQuestion.ID, req.Auth.OpenID)
	if err != nil {
		return nil
	}
	answer.Content = ""
	return &models.QA{Question: *heNotAnsweredQuestion, Answer: *answer}
}

// getBothNotAnsweredQuestion 双方都未答
func getBothNotAnsweredQuestion(db scols, req request, shakedQuestions []bson.ObjectId) (*models.QA, bool) {
	answers := []models.Answer{}
	err := db.Answer.Find(bson.M{
		"openid": bson.M{"$in": []string{req.Auth.OpenID, req.OpenID}},
		"status": bson.M{"$ne": models.StatusDelete},
	}).All(&answers)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	answerIDs := make([]bson.ObjectId, len(answers))
	questionIDs := make([]bson.ObjectId, len(answers))
	for i, v := range answers {
		answerIDs[i] = v.ID
		questionIDs[i] = v.QuestionID
	}

	questions := []models.Question{}
	err = db.Question.Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).All(&questions)
	if err != nil {
		return nil, false
	}

	notAnsweredQuestions := []models.Question{}
	questionMap := map[bson.ObjectId]models.Question{}
	for _, q := range questions {
		for _, a := range questionIDs {
			if a != q.ID {
				notAnsweredQuestions = append(notAnsweredQuestions, q)
				questionMap[q.ID] = q
			}
		}
	}

	removeReps := []bson.ObjectId{}
	for i := range notAnsweredQuestions {
		flag := true
		for j := range removeReps {
			if notAnsweredQuestions[i].ID == removeReps[j] {
				flag = false
				break
			}
		}
		if flag {
			removeReps = append(removeReps, notAnsweredQuestions[i].ID)
		}
	}

	notShakedQuestionIDs := subtracting(removeReps, shakedQuestions)

	l := len(notShakedQuestionIDs)
	var r bson.ObjectId
	if l == 0 {
		return nil, len(removeReps) > 0
	} else if l == 1 {
		r = notShakedQuestionIDs[0]
	} else {
		next, err := routerUtils.GetQuestionsWeightRandomNext(db.Question, notShakedQuestionIDs)
		if err != nil {
			return nil, false
		}
		r = *next
	}

	if question, ok := questionMap[r]; ok {
		return &models.QA{Question: question, Answer: models.Answer{}}, false
	}

	return nil, false
}

type shakeUserInfoResponse struct {
	User  models.LeanUser `json:"user"`
	Owner models.LeanUser `json:"owner"`
}

// ShakeUserInfo 返回摇一摇或点一点页面用户信息
func ShakeUserInfo(c *gin.Context) {

	var params routers.OpenIDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	user, err := routerUtils.FindUser(cols, params.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	resp := shakeUserInfoResponse{Owner: auth.ToLean(), User: user.ToLean()}

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

type randomUserInfoResponse struct {
	User  *models.LeanUser `json:"user"`
	Owner models.LeanUser  `json:"owner"`
}

// Random 随机返回一个好友
func Random(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	userList := models.AnswerExchangeUserList{}
	answerExchangeUserListCol := cols.AnswerExchangeUserList()
	answerExchangeUserListCol.Find(bson.M{"openid": auth.OpenID}).One(&userList)

	l := len(userList.Exchanges)
	if l == 0 {
		resp := randomUserInfoResponse{Owner: auth.ToLean()}
		c.JSON(http.StatusOK, resp)
		return
	} else if l == 1 {
		user, err := routerUtils.FindUser(cols, userList.Exchanges[0].OpenID)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
			return
		}
		lean := user.ToLean()
		resp := randomUserInfoResponse{Owner: auth.ToLean(), User: &lean}
		c.JSON(http.StatusOK, resp)
		return
	} else {
		index := utils.Random(0, l-1)
		user, err := routerUtils.FindUser(cols, userList.Exchanges[index].OpenID)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
			return
		}
		lean := user.ToLean()
		resp := randomUserInfoResponse{Owner: auth.ToLean(), User: &lean}
		c.JSON(http.StatusOK, resp)
	}

}
