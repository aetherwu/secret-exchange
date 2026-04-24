package answer

import (
	"log"
	"net/http"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

// Privacy 获取答案的隐私等级
func Privacy(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	answerCol := cols.Answers()

	answer := &models.Answer{}
	err := answerCol.Find(bson.M{"_id": bson.ObjectIdHex(params.ID)}).One(&answer)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if auth.OpenID != answer.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.PermissionError())
		return
	}

	c.JSON(http.StatusOK, gin.H{"privacy": answer.Privacy})
}

type modifyPrivacyRequest struct {
	ID      string `binding:"required"`
	Privacy int
}

// ModifyPrivacy 修改答案的隐私等级
func ModifyPrivacy(c *gin.Context) {

	var params modifyPrivacyRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	if params.Privacy < models.AnswerPrivacyFree || params.Privacy > models.AnswerPrivacyMyself {
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	answerCol := cols.Answers()

	answer := &models.Answer{}
	err := answerCol.Find(bson.M{"_id": bson.ObjectIdHex(params.ID)}).One(&answer)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	if auth.OpenID != answer.OpenID {
		c.AbortWithStatusJSON(http.StatusOK, routers.PermissionError())
		return
	}

	err = answerCol.Update(bson.M{"_id": answer.ID}, bson.M{"$set": bson.M{"privacy": params.Privacy}})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// update (add or remove) activities to follower

	c.JSON(http.StatusOK, gin.H{"privacy": params.Privacy})
}
