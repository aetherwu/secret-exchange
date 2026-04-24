package dashboard

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

type activityDisplay struct {
	Question  models.Question      `json:"question"`
	Source    models.AnswerDisplay `json:"source"`
	Swap      models.AnswerDisplay `json:"swap"`
	CreatedAt time.Time            `json:"createdAt"`
}

// Activitys 动态
func Activitys(c *gin.Context) {

	var params queryRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	exchangeCol := cols.AnswerExchange()
	exchangeds := []models.AnswerExchange{}

	query := exchangeCol.Find(nil)
	total, err := query.Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	limit := 20

	skip := (params.Page - 1) * limit
	if skip < 0 {
		skip = 0
	}

	query = query.Limit(limit).Skip(skip)

	sort := ""
	if params.OrderBy == "createdAt" {
		sort = "createdat"
	}
	direction := "-"
	if params.OrderDirection == 1 {
		direction = "+"
	}

	sort = direction + sort
	if len([]rune(sort)) > 1 {
		query = query.Sort(sort)
	}

	if err := query.All(&exchangeds); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answerIDs := []bson.ObjectId{}
	questionIDs := []bson.ObjectId{}
	openIDs := []string{}
	for _, v := range exchangeds {
		answerIDs = append(answerIDs, v.Source)
		answerIDs = append(answerIDs, v.Swapper)
		questionIDs = append(questionIDs, v.Question)
		openIDs = append(openIDs, v.SourceUser)
		openIDs = append(openIDs, v.SwapperUser)
	}

	answerIDs = utils.ObjectIDRemoveRep(answerIDs)
	questionIDs = utils.ObjectIDRemoveRep(questionIDs)
	openIDs = utils.StringRemoveRep(openIDs)

	answerCol := cols.Answers()
	answers := []models.Answer{}

	if err := answerCol.Find(bson.M{"_id": bson.M{"$in": answerIDs}}).All(&answers); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answerMap := map[bson.ObjectId]models.Answer{}
	for _, v := range answers {
		answerMap[v.ID] = v
	}

	questionCol := cols.Questions()
	questions := []models.Question{}
	if err := questionCol.Find(bson.M{"_id": bson.M{"$in": questionIDs}}).All(&questions); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	questionMap := map[bson.ObjectId]models.Question{}
	for _, v := range questions {
		questionMap[v.ID] = v
	}

	users, err := routerUtils.FindUsersAsMap(cols, openIDs)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	result := []activityDisplay{}
	for _, ex := range exchangeds {
		r := activityDisplay{CreatedAt: ex.CreatedAt}
		if q, ok := questionMap[ex.Question]; ok {
			r.Question = q
		}
		if a, ok := answerMap[ex.Source]; ok {
			if u, ok := users[ex.SourceUser]; ok {
				r.Source = models.AnswerDisplay{Answer: a, User: u.ToLean()}
			}
		}
		if a, ok := answerMap[ex.Swapper]; ok {
			if u, ok := users[ex.SwapperUser]; ok {
				r.Swap = models.AnswerDisplay{Answer: a, User: u.ToLean()}
			}
		}
		result = append(result, r)
	}

	c.JSON(http.StatusOK, gin.H{"activitys": result, "total": total})
}
