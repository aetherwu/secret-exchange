package profile

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

type reputationResponse struct {
	User          models.LeanUser `json:"user"`
	Owner         models.LeanUser `json:"owner"`
	ExchangeCount int             `json:"exchangeCount"`
}

// Reputation 声望
func Reputation(c *gin.Context) {

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

	resp := reputationResponse{Owner: auth.ToLean(), User: user.ToLean()}

	answerExchangeAtUserCol := cols.AnswerExchangeAtUser()
	exchangeCount := bson.M{}
	err = answerExchangeAtUserCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"openid": auth.OpenID, "user": params.OpenID}},
		bson.M{"$project": bson.M{"count": bson.M{"$size": "$answers"}}},
	}).One(&exchangeCount)
	if err != nil {
		log.Println(err)
	} else {
		resp.ExchangeCount = exchangeCount["count"].(int)
	}

	c.JSON(http.StatusOK, resp)
}
