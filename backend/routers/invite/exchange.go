package invite

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"dagong.in/delphi-web-server/routers/wechat"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

func ExchangeWithInvitePermit(c *gin.Context) {

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

	// 我的答案
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

	exchanged := routerUtils.IsExchanged(cols, question.ID, answer.OpenID, invitePermit.OpenID)

	if exchanged != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "已经交换过答案"))
		return
	}

	// 他的答案
	answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, invitePermit.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	answerCol := cols.Answers()
	answerCol.Update(bson.M{"_id": answer.ID}, bson.M{"$inc": bson.M{"beexchangedcount": 1}})
	answerCol.Update(bson.M{"_id": answered.ID}, bson.M{"$inc": bson.M{"exchangecount": 1}})

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

	if err := exchangeCol.Insert(&exchange); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

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

	err = invitePermitsCol.Update(bson.M{"_id": bson.ObjectIdHex(params.ID)}, bson.M{"$set": bson.M{"status": models.InvitePermitStatusPass}})
	if err != nil {
		log.Println(err)
	}

	inviteBadgeNumberCol := cols.InviteBadgeNumber()
	inviteBadgeNumberCol.Update(bson.M{"openid": invite.Owner, "number": bson.M{"$gt": 0}}, bson.M{"$inc": bson.M{"number": -1}})

	user, _ := routerUtils.FindUser(cols, answered.OpenID)

	notify := wechat.InvitePermitPassInviterNotify{
		Nickname: auth.Nickname + "接受了互换邀请。",
		Text:     fmt.Sprintf("%s 和 %s 成功互换了答案。", auth.Nickname, user.Nickname),
	}

	page := "pages/invite-permit-status?id=" + invitePermit.ID.Hex()
	wechat.SendNotify(invitePermit.InviteUserID, page, notify)

	msg := wechat.ExchangedNotify{
		Content: fmt.Sprintf("点击查看：%s 回答了 问题 %s", auth.Nickname, question.Content),
		Date:    time.Now().Format("2006/01/02 - 15:04:05"),
	}
	if err := wechat.SendNotify(answered.OpenID, "pages/answer?id="+answer.ID.Hex(), msg); err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, routers.OK())
}
