package invite

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
	Question           models.Question        `json:"question"`
	Answer             models.AnswerDisplay   `json:"answer"`
	Owner              bool                   `json:"owner"`
	Exchanged          bool                   `json:"exchanged"`
	Answered           *models.AnswerDisplay  `json:"answered"`
	ExchangedAt        *time.Time             `json:"exchangedAt"`
	Exchangeds         []models.AnswerDisplay `json:"exchangeds"`
	ExchangedCount     int                    `json:"exchangedCount"`
	InvitePermitStatus int                    `json:"invitePermitStatus"`
	Inviter            models.LeanUser        `json:"inviter"`
	FormIDCount        int                    `json:"formIDCount"`
}

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

	invitesCol := cols.Invites()
	invite := models.Invite{}
	err := invitesCol.Find(bson.M{"_id": bson.ObjectIdHex(params.ID)}).One(&invite)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	question, answer, err := routerUtils.FindQA(cols, invite.AnswerID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if question == nil || answer == nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	user, err := routerUtils.FindUser(cols, answer.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	inviter, err := routerUtils.FindUser(cols, invite.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	resp := detailResponse{}
	isLogin := auth != nil
	resp.Owner = isLogin && auth.OpenID == answer.OpenID

	answer.Length = len([]rune(answer.Content))

	if auth != nil {
		formIDCount, _ := cols.FormIDs().Find(bson.M{"user": auth.OpenID}).Count()
		resp.FormIDCount = formIDCount

		if !resp.Owner {
			exchanged := routerUtils.IsExchanged(cols, question.ID, answer.OpenID, auth.OpenID)
			resp.Exchanged = exchanged != nil
			if resp.Exchanged {
				resp.ExchangedAt = &exchanged.CreatedAt
			}

			answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, auth.OpenID)
			if err != nil {
				log.Println(err)
			}
			if answered != nil {
				user, err := routerUtils.FindUser(cols, answered.OpenID)
				if err != nil {
					log.Println(err)
					c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
					return
				}
				if len(answered.WeChatShareImage) > 0 {
					answered.WeChatShareImage = routerUtils.QiniuKeyToURL(answered.WeChatShareImage)
				}

				comments, err := routerUtils.GetAnswerComments(cols, answered.ID)
				if err != nil {
					log.Println(err)
				}
				answered.Comments = comments

				answered.IsLike = routerUtils.IsLike(cols, answered.ID, auth.OpenID)
				answered.LikeUsers = routerUtils.Likes(cols, answered.ID)

				resp.Answered = &models.AnswerDisplay{Answer: *answered, User: user.ToLean()}
			}
		}
	}

	if !resp.Owner && !resp.Exchanged {
		answer.Content = ""
	}

	if len(answer.WeChatShareImage) > 0 {
		answer.WeChatShareImage = routerUtils.QiniuKeyToURL(answer.WeChatShareImage)
	}

	if resp.Owner || resp.Exchanged {
		comments, err := routerUtils.GetAnswerComments(cols, answer.ID)
		if err != nil {
			log.Println(err)
		}
		answer.Comments = comments
		answer.IsLike = routerUtils.IsLike(cols, answer.ID, auth.OpenID)
		answer.LikeUsers = routerUtils.Likes(cols, answer.ID)
	} else {
		answer.Comments = []models.AnswerCommentDisplay{}
		answer.LikeUsers = map[int][]models.AnswerLikeUser{}
	}

	resp.Question = *question
	resp.Answer = models.AnswerDisplay{Answer: *answer, User: user.ToLean()}

	if resp.Owner {
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

		if auth != nil {
			invitePermitsCol := cols.InvitePermits()
			invitePermit := models.InvitePermit{}
			err := invitePermitsCol.Find(bson.M{"inviteid": invite.ID}).One(&invitePermit)
			if err == nil {
				resp.InvitePermitStatus = invitePermit.Status + 1
			}
		}
	}

	resp.Inviter = inviter.ToLean()

	c.JSON(http.StatusOK, resp)
}

type detailWithInvitePermitResponse struct {
	Question  models.Question      `json:"question"`
	Answer    models.AnswerDisplay `json:"answer"`
	Answered  models.AnswerDisplay `json:"answered"`
	Exchanged bool                 `json:"exchanged"`
	Inviter   models.LeanUser      `json:"inviter"`
}

func DetailWithInvitePermit(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	invitePermitsCol := cols.InvitePermits()
	invitePermit := models.InvitePermit{}
	err := invitePermitsCol.Find(bson.M{"_id": bson.ObjectIdHex(params.ID)}).One(&invitePermit)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	invitesCol := cols.Invites()
	invite := models.Invite{}
	err = invitesCol.Find(bson.M{"_id": invitePermit.InviteID}).One(&invite)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	question, answered, err := routerUtils.FindQA(cols, invite.AnswerID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if question == nil || answered == nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if answered.OpenID != auth.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.PermissionError())
		return
	}

	answer, err := routerUtils.FindAnswer(cols, invitePermit.AnswerID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	user, err := routerUtils.FindUser(cols, answer.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	inviter, err := routerUtils.FindUser(cols, invitePermit.InviteUserID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	isExchanged := routerUtils.IsExchanged(cols, question.ID, answered.OpenID, answer.OpenID)

	answeredDisplay := models.AnswerDisplay{Answer: *answered, User: auth.ToLean()}
	resp := detailWithInvitePermitResponse{Question: *question, Answered: answeredDisplay}
	if isExchanged == nil {
		resp.Exchanged = false
		answer.Length = len([]rune(answer.Content))
		answer.Content = ""
	} else {
		resp.Exchanged = true
	}
	resp.Answer = models.AnswerDisplay{Answer: *answer, User: user.ToLean()}
	resp.Inviter = inviter.ToLean()

	c.JSON(http.StatusOK, resp)
}

type invitePermitStatusResponse struct {
	Question models.Question `json:"question"`
	Status   int             `json:"status"`
	Inviter  models.LeanUser `json:"inviter"`
	Owner    models.LeanUser `json:"owner"`
	Invitee  models.LeanUser `json:"invitee"`
}

func InvitePermitStatus(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	invitePermitsCol := cols.InvitePermits()
	invitePermit := models.InvitePermit{}
	err := invitePermitsCol.Find(bson.M{"_id": bson.ObjectIdHex(params.ID)}).One(&invitePermit)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	invitesCol := cols.Invites()
	invite := models.Invite{}
	err = invitesCol.Find(bson.M{"_id": invitePermit.InviteID}).One(&invite)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	question, err := routerUtils.FindQuestion(cols, invitePermit.QuestionID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	owner, err := routerUtils.FindUser(cols, invitePermit.Owner)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	inviter, err := routerUtils.FindUser(cols, invitePermit.InviteUserID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	invitee, err := routerUtils.FindUser(cols, invitePermit.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if owner.OpenID == auth.OpenID || inviter.OpenID == auth.OpenID || invitee.OpenID == auth.OpenID {
		resp := invitePermitStatusResponse{
			Status:   invitePermit.Status,
			Question: *question,
			Owner:    owner.ToLean(),
			Inviter:  inviter.ToLean(),
			Invitee:  invitee.ToLean(),
		}

		c.JSON(http.StatusOK, resp)
	} else {
		c.AbortWithStatusJSON(http.StatusOK, routers.PermissionError())
		return
	}

}
