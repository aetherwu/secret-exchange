package answer

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
	"github.com/globalsign/mgo/bson"
)

type detailResponse struct {
	Question       models.Question        `json:"question"`       // 问题
	Answer         models.AnswerDisplay   `json:"answer"`         // 答案
	Owner          bool                   `json:"owner"`          // 是否是答案创建者
	Exchanged      bool                   `json:"exchanged"`      // 是否已经交换过
	Answered       *models.AnswerDisplay  `json:"answered"`       // 如果不是答案的创建者 回答过的答案
	ExchangedAt    *time.Time             `json:"exchangedAt"`    // 如果交换过 交换的日期
	Exchangeds     []models.AnswerDisplay `json:"exchangeds"`     // 如果是答案创建者 已经交换到的答案
	ExchangedCount int                    `json:"exchangedCount"` // 如果是答案创建者 交换到的答案数
	FormIDCount    int                    `json:"formIDCount"`    // 已经登录的话 Form id 数
	IsFriend       bool                   `json:"isfriend"`       // 是否相互关注
}

// Detail 答案详情
func Detail(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.TryGetAuth(c)

	session, cols := db.Default()
	defer session.Close()

	// 根据答案 ID 查询问题和答案是否存在
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

	// 查询答案创建者
	user, err := routerUtils.FindUser(cols, answer.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	resp := detailResponse{}
	// 是否已登录
	isLogin := auth != nil
	// 是否是答案的创建者
	resp.Owner = isLogin && auth.OpenID == answer.OpenID
	// 答案文字数
	answer.Length = len([]rune(answer.Content))

	// 已登录
	if isLogin {

		// 查询 from id 数
		formIDCount, _ := cols.FormIDs().Find(bson.M{"user": auth.OpenID}).Count()
		resp.FormIDCount = formIDCount

		// 不是答案创建者
		if !resp.Owner {

			// 是否已经交换过
			exchanged := routerUtils.IsExchanged(cols, question.ID, answer.OpenID, auth.OpenID)
			resp.Exchanged = exchanged != nil
			if resp.Exchanged {
				resp.ExchangedAt = &exchanged.CreatedAt
			}

			// check mutual following - friendship
			col := cols.Follows()
			var follower *models.Follows

			col.Find(bson.M{
				"openid": auth.OpenID,
				"follower": bson.M{
					"$elemMatch": bson.M{
						"openid": answer.OpenID,
					},
				},
			}).One(&follower)

			if follower != nil {
				resp.IsFriend = true
			} else {
				resp.IsFriend = false
			}

			// 查询时候已经回答过这个问题
			answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, auth.OpenID)
			if err != nil {
				log.Println(err)
			}
			// 回答过这个问题
			if answered != nil {
				// 查询我回答过的答案
				user, err := routerUtils.FindUser(cols, answered.OpenID)
				if err != nil {
					log.Println(err)
					c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
					return
				}

				if len(answered.WeChatShareImage) > 0 {
					answered.WeChatShareImage = routerUtils.QiniuKeyToURL(answered.WeChatShareImage)
				}

				// 获取评论
				comments, err := routerUtils.GetAnswerComments(cols, answered.ID)
				if err != nil {
					log.Println(err)
				}
				answered.Comments = comments
				// 是否点过赞
				answered.IsLike = routerUtils.IsLike(cols, answered.ID, auth.OpenID)
				// 点过赞的用户
				answered.LikeUsers = routerUtils.Likes(cols, answered.ID)

				resp.Answered = &models.AnswerDisplay{Answer: *answered, User: user.ToLean()}
			}
		}
	}

	// 如果不是答案创建者 没交换过 问题不是公开的 就不返回答案的内容
	if !resp.Owner && !resp.Exchanged && answer.Privacy != models.AnswerPrivacyFree {
		answer.Content = ""
	}

	if len(answer.WeChatShareImage) > 0 {
		answer.WeChatShareImage = routerUtils.QiniuKeyToURL(answer.WeChatShareImage)
	}

	// 如果是答案的创建者 或 已经交换过
	if resp.Owner || resp.Exchanged {
		// 查询评论
		comments, err := routerUtils.GetAnswerComments(cols, answer.ID)
		if err != nil {
			log.Println(err)
		}
		answer.Comments = comments
		// 查询是否点过赞
		answer.IsLike = routerUtils.IsLike(cols, answer.ID, auth.OpenID)
		// 查询点过赞的用户
		answer.LikeUsers = routerUtils.Likes(cols, answer.ID)
	} else {
		answer.Comments = []models.AnswerCommentDisplay{}
	}

	resp.Question = *question
	resp.Answer = models.AnswerDisplay{Answer: *answer, User: user.ToLean()}

	// 是答案的创建者
	if resp.Owner {
		// 查询已经交过换的答案
		exchangeCol := cols.AnswerExchange()
		exchangeds := []models.AnswerExchange{}
		or := []bson.M{bson.M{"source": answer.ID}, bson.M{"swapper": answer.ID}}
		if err := exchangeCol.Find(bson.M{"question": question.ID, "$or": or}).All(&exchangeds); err != nil {
			log.Println(err)
		}

		answerIDs := []bson.ObjectId{}
		for _, v := range exchangeds {
			if v.Source == answer.ID {
				answerIDs = append(answerIDs, v.Swapper)
			} else if v.Swapper == answer.ID {
				answerIDs = append(answerIDs, v.Source)
			}
		}

		answerCol := cols.Answers()
		swapperAnswers := []models.Answer{}
		if err := answerCol.Find(bson.M{
			"_id": bson.M{"$in": answerIDs},
		}).Sort("-createdat").All(&swapperAnswers); err != nil {
			log.Println(err)
		}

		openids := make([]string, len(swapperAnswers))
		for i, v := range swapperAnswers {
			openids[i] = v.OpenID
		}

		users, err := routerUtils.FindUsersAsMap(cols, openids)
		if err != nil {
			log.Println(err)
		}

		exchangedDisplays := []models.AnswerDisplay{}
		for _, v := range swapperAnswers {
			if u, ok := users[v.OpenID]; ok {
				v.Length = len([]rune(v.Content))
				exchangedDisplays = append(exchangedDisplays, models.AnswerDisplay{
					Answer: v,
					User:   u.ToLean(),
				})
			}
		}

		resp.ExchangedCount = len(exchangedDisplays)
		resp.Exchangeds = exchangedDisplays
	} else {
		resp.Exchangeds = []models.AnswerDisplay{}
	}

	c.JSON(http.StatusOK, resp)
}
