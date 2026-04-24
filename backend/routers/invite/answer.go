package invite

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

type genInvitePermitRequest struct {
	Invite string `binding:"required"`
	Answer string `binding:"required"`
}

func GenInvitePermit(c *gin.Context) {

	var params genInvitePermitRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	invitesCol := cols.Invites()
	invite := models.Invite{}
	err := invitesCol.Find(bson.M{"_id": bson.ObjectIdHex(params.Invite)}).One(&invite)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	question, err := routerUtils.FindQuestion(cols, invite.QuestionID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	answerCol := cols.Answers()

	answered, err := routerUtils.FindAnswerWithQuesiton(cols, invite.QuestionID, auth.OpenID)
	if answered != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "已经回答过此问题"))
		return
	}

	now := time.Now()

	answer := models.Answer{
		ID:         bson.NewObjectId(),
		OpenID:     auth.OpenID,
		QuestionID: invite.QuestionID,
		Content:    params.Answer,
		Status:     models.StatusNormal,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	qa := models.QADisplay{Question: *question, Answer: models.AnswerDisplay{
		Answer: answer,
		User:   auth.ToLean(),
	}}

	image, err := draw.GenWeChatShareImageOfDraw(qa)
	if err != nil {
		log.Println(err)
	} else {
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
		} else {
			answer.WeChatShareImage = key
			answer.WeChatShareImageStatus = models.WeChatShareImageStatusGenerated
		}
	}

	if err := answerCol.Insert(&answer); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	questionCol := cols.Questions()
	if err := questionCol.Update(bson.M{"_id": question.ID}, bson.M{"$inc": bson.M{"answercount": 1}}); err != nil {
		log.Println(err)
	}

	userAnswerCountersCol := cols.UserAnswerCounters()
	userAnswerCountersCol.Upsert(bson.M{"openid": auth.OpenID}, bson.M{"$inc": bson.M{
		"answercount": 1,
	}})

	invitePermitsCol := cols.InvitePermits()
	invitePermit := models.InvitePermit{
		ID:           bson.NewObjectId(),
		InviteID:     invite.ID,
		QuestionID:   question.ID,
		AnswerID:     answer.ID,
		Owner:        invite.Owner,
		InviteUserID: invite.OpenID,
		OpenID:       answer.OpenID,
		Status:       models.InvitePermitStatusWait,
		CreatedAt:    time.Now(),
	}
	err = invitePermitsCol.Insert(invitePermit)

	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	inviteBadgeNumberCol := cols.InviteBadgeNumber()
	inviteBadgeNumberCol.Upsert(bson.M{"openid": invite.Owner}, bson.M{"$inc": bson.M{"number": 1}})

	msg := wechat.InvitePermitPassOwnerNotify{
		Question: question.Content,
		Text:     fmt.Sprintf("用户 %s 申请和你交换答案。请点此查看详情。", auth.Nickname),
	}
	page := "pages/invite-permit?id=" + invitePermit.ID.Hex()
	wechat.SendNotify(invite.Owner, page, msg)

	c.JSON(http.StatusOK, gin.H{"id": invitePermit.ID})
}

type genInvitePermitWithAnswerRequest struct {
	Invite string `binding:"required"`
}

func GenInvitePermitWithAnswer(c *gin.Context) {

	var params genInvitePermitWithAnswerRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	invitesCol := cols.Invites()
	invite := models.Invite{}
	err := invitesCol.Find(bson.M{"_id": bson.ObjectIdHex(params.Invite)}).One(&invite)
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
	// 我的答案
	answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, auth.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	invitePermitsCol := cols.InvitePermits()
	invitePermit := models.InvitePermit{
		ID:           bson.NewObjectId(),
		InviteID:     invite.ID,
		QuestionID:   question.ID,
		AnswerID:     answered.ID,
		Owner:        invite.Owner,
		InviteUserID: invite.OpenID,
		OpenID:       answered.OpenID,
		Status:       models.InvitePermitStatusWait,
		CreatedAt:    time.Now(),
	}
	err = invitePermitsCol.Insert(invitePermit)

	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	inviteBadgeNumberCol := cols.InviteBadgeNumber()
	inviteBadgeNumberCol.Upsert(bson.M{"openid": invite.Owner}, bson.M{"$inc": bson.M{"number": 1}})

	c.JSON(http.StatusOK, gin.H{"id": invitePermit.ID})
}
