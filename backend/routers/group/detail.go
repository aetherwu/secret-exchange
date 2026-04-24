package group

import (
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type detailRequest struct {
	Answer string `binding:"required"`
	Group  string `binding:"required"`
}

type answerDisplay struct {
	models.AnswerDisplay
	Exchanged bool `json:"exchanged"`
}

type detailResponse struct {
	Question    models.Question       `json:"question"`    // 问题
	Answer      models.AnswerDisplay  `json:"answer"`      // 答案
	Owner       bool                  `json:"owner"`       // 是否是答案创建者
	GroupID     string                `json:"groupID"`     // 群id
	Answers     []answerDisplay       `json:"answers"`     //  已经交换到的答案
	Users       []models.LeanUser     `json:"users" `      // 已经交换的用户
	AddCount    int                   `json:"addCount"`    // 已经交换的用户数
	Exchanged   bool                  `json:"exchanged"`   // 是否已经交换过
	ExchangedAt *time.Time            `json:"exchangedAt"` // 交换时间
	Answered    *models.AnswerDisplay `json:"answered"`    // 我的答案
	User        models.LeanUser       `json:"user"`        // 我
	FormIDCount int                   `json:"formIDCount"` // 已经登录的话 Form id 数
}

// Detail 群分享的问题详情
func Detail(c *gin.Context) {

	var params detailRequest
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

	// 查询答案的创建者
	user, err := routerUtils.FindUser(cols, answer.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 查询问题
	question, err := routerUtils.FindQuestion(cols, answer.QuestionID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 查询群
	groupCol := cols.Groups()
	group := models.Group{}
	err = groupCol.Find(bson.M{"groupid": params.Group, "questionid": question.ID}).One(&group)
	// 群不存在 初始化群
	if err != nil {
		if err == mgo.ErrNotFound {
			group = models.Group{
				ID:         bson.NewObjectId(),
				QuestionID: question.ID,
				GroupID:    params.Group,
				OpenID:     answer.OpenID,
				AddUsers:   []string{answer.OpenID},
				Answers: []models.GroupAnswer{models.GroupAnswer{
					ID:        answer.ID,
					OpenID:    answer.OpenID,
					CreatedAt: time.Now(),
				}},
				CreatedAt: time.Now(),
			}
			err = groupCol.Insert(group)
			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
				return
			}
		} else {
			c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
			return
		}
	}

	resp := detailResponse{}
	resp.GroupID = params.Group
	resp.Question = *question
	resp.Owner = auth.OpenID == answer.OpenID
	resp.User = auth.ToLean()

	formIDCount, _ := cols.FormIDs().Find(bson.M{"user": auth.OpenID}).Count()
	resp.FormIDCount = formIDCount

	// 不是答案的创建者
	if !resp.Owner {
		// 查询我的答案
		answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, auth.OpenID)
		if err != nil {
			log.Println(err)
		}
		// 有回答过这个问题
		if answered != nil {
			if len(answered.WeChatShareImage) > 0 {
				answered.WeChatShareImage = routerUtils.QiniuKeyToURL(answered.WeChatShareImage)
			}

			// 获取我的答案的评论
			comments, err := routerUtils.GetAnswerComments(cols, answered.ID)
			if err != nil {
				log.Println(err)
			}
			answered.Comments = comments

			// 获取我的答案是否点过赞
			answered.IsLike = routerUtils.IsLike(cols, answered.ID, auth.OpenID)
			// 获取我的答案的点赞用户
			answered.LikeUsers = routerUtils.Likes(cols, answered.ID)

			resp.Answered = &models.AnswerDisplay{Answer: *answered, User: auth.ToLean()}
			// 是否交换过
			isJoinedQuestion := isJoinedQuestion(cols, params.Group, question.ID, auth.OpenID)
			if isJoinedQuestion != nil {
				resp.Exchanged = isJoinedQuestion != nil
				resp.ExchangedAt = &isJoinedQuestion.CreatedAt
			}
		}
	}

	answer.Length = len([]rune(answer.Content))
	// 是否需要显示答案内容
	if !resp.Owner && !resp.Exchanged && answer.Privacy != models.AnswerPrivacyFree {
		answer.Content = ""
	}

	if len(answer.WeChatShareImage) > 0 {
		answer.WeChatShareImage = routerUtils.QiniuKeyToURL(answer.WeChatShareImage)
	}

	if resp.Owner || resp.Exchanged {
		// 答案的评论
		comments, err := routerUtils.GetAnswerComments(cols, answer.ID)
		if err != nil {
			log.Println(err)
		}
		answer.Comments = comments
		// 是否给答案点过赞
		answer.IsLike = routerUtils.IsLike(cols, answer.ID, auth.OpenID)
		// 给答案点过赞的用户
		answer.LikeUsers = routerUtils.Likes(cols, answer.ID)
	} else {
		answer.Comments = []models.AnswerCommentDisplay{}
	}

	resp.Answer = models.AnswerDisplay{Answer: *answer, User: user.ToLean()}

	// 如果是答案的创建者或交换过 就返回群里所有答案
	if resp.Owner || resp.Exchanged {
		answerIDs := []bson.ObjectId{}
		for _, v := range group.Answers {
			if resp.Owner {
				if v.ID != answer.ID {
					answerIDs = append(answerIDs, v.ID)
				}
			} else if resp.Exchanged {
				if v.ID != resp.Answered.ID {
					answerIDs = append(answerIDs, v.ID)
				}
			}
		}

		answerCol := cols.Answers()
		answers := []models.Answer{}
		answerCol.Find(bson.M{"_id": bson.M{"$in": answerIDs}}).All(&answers)

		answerMap := map[bson.ObjectId]models.Answer{}
		openIDs := make([]string, len(answers))
		for i, v := range answers {
			answerMap[v.ID] = v
			openIDs[i] = v.OpenID
		}

		users, err := routerUtils.FindUsersAsMap(cols, openIDs)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
			return
		}

		answerDisplays := []answerDisplay{}

		for _, v := range answers {
			if u, ok := users[v.OpenID]; ok {
				// 判断是否单独交换过
				isExchanged := routerUtils.IsExchanged(cols, question.ID, auth.OpenID, v.OpenID)
				// v.Length = len([]rune(v.Content))
				// v.Content = ""
				answerDisplay := answerDisplay{
					AnswerDisplay: models.AnswerDisplay{Answer: v, User: u.ToLean()},
					Exchanged:     isExchanged != nil,
				}
				answerDisplays = append(answerDisplays, answerDisplay)
			}
		}
		resp.Answers = answerDisplays
	} else {
		resp.Answers = []answerDisplay{}
	}

	// 在群里交换过这个问题的用户
	addOpenIDs := make([]string, len(group.AddUsers))
	groupAddUsers := group.AddUsers
	resp.AddCount = len(group.AddUsers)
	if len(group.AddUsers) > 10 {
		groupAddUsers = group.AddUsers[:10]
	}

	for i, v := range groupAddUsers {
		addOpenIDs[i] = v
	}

	addUsers, err := routerUtils.FindUsersAsMap(cols, addOpenIDs)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	addUserDisplays := []models.LeanUser{}
	for _, v := range groupAddUsers {
		if u, ok := addUsers[v]; ok {
			addUserDisplays = append(addUserDisplays, u.ToLean())
		}
	}

	resp.Users = addUserDisplays

	c.JSON(http.StatusOK, resp)
}

type rankRequest struct {
	GID            string `binding:"required"`
	OrderBy        string
	OrderDirection int
}

type userRankResponse struct {
	Ranks []userRankResultDisplay `json:"ranks"`
	Owner *userRankResultDisplay  `json:"owner"`
}

type userRankResult struct {
	ID               string `bson:"_id"`
	AnswerCount      int    `bson:"answercount"`
	ExchangeCount    int    `bson:"exchangecount"`
	BeExchangedCount int    `bson:"beexchangedcount"`
	AllExchangeCount int    `bson:"allexchangecount"`
}

type userRankResultDisplay struct {
	AnswerCount      int             `json:"answerCount"`
	ExchangeCount    int             `json:"exchangeCount"`
	BeExchangedCount int             `json:"beExchangedCount"`
	AllExchangeCount int             `json:"allExchangeCount"`
	User             models.LeanUser `json:"user"`
	Rank             int             `json:"rank"`
}

// UserRank 群用户排行榜
func UserRank(c *gin.Context) {

	var params rankRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	sort := "answercount"
	if params.OrderBy == "exchangedCount" {
		sort = "allexchangecount"
	}

	orderDirection := -1
	if params.OrderDirection == 1 {
		orderDirection = 1
	}

	groupCol := cols.Groups()
	results := []userRankResult{}
	groupCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"groupid": params.GID}},
		bson.M{"$unwind": "$addusers"},
		bson.M{"$project": bson.M{"addusers": 1}},
		bson.M{"$lookup": bson.M{
			"from":         "useranswercounters",
			"localField":   "addusers",
			"foreignField": "openid",
			"as":           "as",
		}},
		bson.M{"$unwind": "$as"},
		bson.M{"$group": bson.M{
			"_id":              "$as.openid",
			"answercount":      bson.M{"$first": "$as.answercount"},
			"exchangecount":    bson.M{"$first": "$as.exchangecount"},
			"beexchangedcount": bson.M{"$first": "$as.beexchangedcount"},
			"allexchangecount": bson.M{"$first": "$as.allexchangecount"},
		}},
		bson.M{"$sort": bson.M{sort: orderDirection}},
	}).All(&results)
	resp := userRankResponse{}

	openIDs := make([]string, len(results))
	for i, v := range results {
		openIDs[i] = v.ID
	}

	users, err := routerUtils.FindUsersAsMap(cols, openIDs)
	if err != nil {
		log.Println(err)
	}

	displays := []userRankResultDisplay{}
	for i, v := range results {
		if u, ok := users[v.ID]; ok {
			r := userRankResultDisplay{
				AnswerCount:      v.AnswerCount,
				ExchangeCount:    v.ExchangeCount,
				BeExchangedCount: v.BeExchangedCount,
				AllExchangeCount: v.AllExchangeCount,
				User:             u.ToLean(),
				Rank:             i + 1,
			}
			displays = append(displays, r)
			if u.OpenID == auth.OpenID {
				resp.Owner = &r
			}
		}
	}

	resp.Ranks = displays

	c.JSON(http.StatusOK, resp)
}

type questionRankResult struct {
	ID       bson.ObjectId   `bson:"_id"`
	Count    int             `bson:"count"`
	As       models.Question `bson:"as"`
	AnswerID bson.ObjectId   `bson:"answerid"`
}

type questionRankDisplay struct {
	Question models.Question `json:"question"`
	AnswerID string          `json:"answer"`
	Count    int             `json:"count"`
}

type questionRankResponse struct {
	Ranks []questionRankDisplay `json:"ranks"`
}

// QuestionRank 群问题排行榜
func QuestionRank(c *gin.Context) {

	var params rankRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	sort := "count"
	if params.OrderBy == "updatedAt" {
		sort = "updatedat"
	}

	orderDirection := -1
	if params.OrderDirection == 1 {
		orderDirection = 1
	}

	groupCol := cols.Groups()
	results := []questionRankResult{}
	groupCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"groupid": params.GID}},
		bson.M{"$lookup": bson.M{
			"from":         "questions",
			"localField":   "questionid",
			"foreignField": "_id",
			"as":           "as",
		}},
		bson.M{"$unwind": "$as"},
		bson.M{"$project": bson.M{
			"as":       1,
			"count":    bson.M{"$size": "$addusers"},
			"answerid": 1,
		}},
		bson.M{"$sort": bson.M{sort: orderDirection}},
	}).All(&results)

	displays := make([]questionRankDisplay, len(results))
	for i, v := range results {
		d := questionRankDisplay{Question: v.As, Count: v.Count, AnswerID: v.AnswerID.Hex()}
		displays[i] = d
	}

	resp := questionRankResponse{Ranks: displays}
	c.JSON(http.StatusOK, resp)
}
