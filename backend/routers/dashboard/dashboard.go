package dashboard

import (
	"log"
	"net/http"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
)

func Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func Count(c *gin.Context) {

	session, cols := db.Default()
	defer session.Close()

	questionCount, err := cols.Questions().Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answerCount, err := cols.Answers().Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	exchangeCount, err := cols.AnswerExchange().Find(nil).Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	userCount, err := cols.Users().Find(nil).Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	var groupCount bson.M
	cols.Groups().Pipe([]bson.M{
		bson.M{"$group": bson.M{
			"_id": "$groupid",
		}},
		bson.M{"$count": "total"},
	}).One(&groupCount)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	groupTopicCount, err := cols.Groups().Find(nil).Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"questions":   questionCount,
		"answers":     answerCount,
		"exchanges":   exchangeCount,
		"users":       userCount,
		"groups":      groupCount["total"].(int),
		"groupTopics": groupTopicCount,
	})
}
