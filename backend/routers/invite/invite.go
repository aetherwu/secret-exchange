package invite

import (
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"dagong.in/delphi-web-server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type getInviteResponse struct {
	ID string `json:"id"`
}

func GetInvite(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	answerID := bson.ObjectIdHex(params.ID)

	answer, err := routerUtils.FindAnswer(cols, answerID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	invitesCol := cols.Invites()
	invites := []models.Invite{}
	err = invitesCol.Find(bson.M{"answerid": answerID, "openid": auth.OpenID}).Limit(1).All(&invites)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	if len(invites) == 0 {
		invite := models.Invite{
			ID:         bson.NewObjectId(),
			QuestionID: answer.QuestionID,
			AnswerID:   answerID,
			Owner:      answer.OpenID,
			OpenID:     auth.OpenID,
			CreatedAt:  time.Now(),
		}
		err := invitesCol.Insert(invite)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}
		c.JSON(http.StatusOK, getInviteResponse{ID: invite.ID.Hex()})
	} else {
		c.JSON(http.StatusOK, getInviteResponse{ID: invites[0].ID.Hex()})
	}
}

type invitePermitListResponseMode struct {
	User         models.LeanUser `json:"user"`
	Question     string          `json:"question"`
	CreatedAt    time.Time       `json:"createdAt"`
	InvitePermit string          `json:"invitePermit"`
	Status       int             `json:"status"`
}

type invitePermitListResponse struct {
	InvitePermits []invitePermitListResponseMode `json:"invitePermits"`
}

func InvitePermitList(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	invitePermitsCol := cols.InvitePermits()
	invitePermits := []models.InvitePermit{}
	err := invitePermitsCol.Find(bson.M{"owner": auth.OpenID}).Sort("-createdat").All(&invitePermits)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	questionIDs := make([]bson.ObjectId, len(invitePermits))
	userIDs := make([]string, len(invitePermits))
	for i, v := range invitePermits {
		questionIDs[i] = v.QuestionID
		userIDs[i] = v.OpenID
	}

	userIDs = utils.StringRemoveRep(userIDs)
	questionIDs = utils.ObjectIDRemoveRep(questionIDs)

	questionCol := cols.Questions()
	questions := []models.Question{}
	if err := questionCol.Find(bson.M{"_id": bson.M{"$in": questionIDs}}).All(&questions); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	questionMap := map[bson.ObjectId]models.Question{}
	for _, v := range questions {
		questionMap[v.ID] = v
	}

	users, err := routerUtils.FindUsersAsMap(cols, userIDs)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	resp := invitePermitListResponse{InvitePermits: []invitePermitListResponseMode{}}
	for _, v := range invitePermits {
		if q, ok := questionMap[v.QuestionID]; ok {
			if u, ok := users[v.OpenID]; ok {
				i := invitePermitListResponseMode{
					User:         u.ToLean(),
					Question:     q.Content,
					InvitePermit: v.ID.Hex(),
					CreatedAt:    v.CreatedAt,
					Status:       v.Status,
				}
				resp.InvitePermits = append(resp.InvitePermits, i)
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}
