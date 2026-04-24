package profile

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

type myRequest struct {
	LastID string
}

type myResponse struct {
	Answer   models.Answer   `json:"answer"`
	Question models.Question `json:"question"`
}

type myResponses []myResponse

func (p myResponses) Len() int {
	return len(p)
}

func (p myResponses) Less(i, j int) bool {
	return p[i].Answer.ExchangeCount > p[j].Answer.ExchangeCount
}

func (p myResponses) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// My 我的主页 最近回答
func My(c *gin.Context) {

	var params myRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	currentUser := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	answerCol := cols.Answers()

	answereds := []models.Answer{}
	query := bson.M{"openid": currentUser.OpenID, "status": bson.M{"$ne": models.StatusDelete}}

	// ?? what is this for
	if params.LastID != "" {
		last, err := routerUtils.FindAnswer(cols, bson.ObjectIdHex(params.LastID))
		if err != nil {
			log.Println(err)
		} else {
			query["createdat"] = bson.M{"$lt": last.CreatedAt}
		}
	}

	// Fetch personal answers from collection
	err := answerCol.
		Find(query).
		Sort("-createdat").
		Limit(20).
		All(&answereds)
	if err != nil {
		log.Println(err)
	}

	// Reveal question id, extract redundant question IDs
	questionIDs := []bson.ObjectId{}
	for _, v := range answereds {
		questionIDs = append(questionIDs, v.QuestionID)
	}

	// Fetch questions using IDs
	questions := []models.Question{}
	questionCol := cols.Questions()
	if err := questionCol.Find(bson.M{"_id": bson.M{"$in": questionIDs}, "status": bson.M{"$ne": models.StatusDelete}}).All(&questions); err != nil {
		log.Println(err)
	}

	// Convert result to data set
	questionsMap := map[bson.ObjectId]models.Question{}
	for _, v := range questions {
		questionsMap[v.ID] = v
	}

	// Prepare response
	resp := myResponses{}
	for _, v := range answereds {
		if q, ok := questionsMap[v.QuestionID]; ok {
			v.Length = len([]rune(v.Content))

			// Reduce response body to save traffic and avoid pravicy disclose
			//v.Content = ""

			r := myResponse{Answer: v, Question: q}
			resp = append(resp, r)
		}
	}

	// Return to client
	c.JSON(http.StatusOK, resp)
}

// For paginate use
type myAnswerExchangeAtMostRequest struct {
	Skip int
}

// MyAnswerExchangeAtMost 我的主页 最多交换
func MyAnswerExchangeAtMost(c *gin.Context) {

	var params myAnswerExchangeAtMostRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	currentUser := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	answerCol := cols.Answers()
	answers := models.Answers{}
	answerCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{
			"openid": currentUser.OpenID,
			"status": bson.M{"$gt": models.StatusDelete},
		}},
		bson.M{"$sort": bson.M{"allexchangecount": -1}},
		bson.M{"$skip": params.Skip},
		bson.M{"$limit": 20},
	}).All(&answers)

	questionIDs := make([]bson.ObjectId, len(answers))
	for i, v := range answers {
		questionIDs[i] = v.QuestionID
	}

	questionCol := cols.Questions()
	questions := []models.Question{}
	if err := questionCol.Find(bson.M{"_id": bson.M{"$in": questionIDs}}).All(&questions); err != nil {
		log.Println(err)
	}

	questionMap := map[bson.ObjectId]models.Question{}
	for _, v := range questions {
		questionMap[v.ID] = v
	}

	resp := myResponses{}

	for _, v := range answers {
		r := myResponse{}
		if question, ok := questionMap[v.QuestionID]; ok {
			r.Question = question
		}
		v.Length = len([]rune(v.Content))
		//v.Content = ""
		r.Answer = v
		resp = append(resp, r)
	}

	c.JSON(http.StatusOK, resp)
}

type myAnswerExchangeRequest struct {
	LastID string
}

// MyExchange 我的主页 最近交换
func MyExchange(c *gin.Context) {

	var params myAnswerExchangeRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	currentUser := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	lastDate := time.Now()
	if params.LastID != "" {
		last, err := routerUtils.FindAnswer(cols, bson.ObjectIdHex(params.LastID))
		if err != nil {
			log.Println(err)
		} else {
			lastDate = last.CreatedAt
		}
	}

	exchangeCol := cols.AnswerExchange()
	exchanges := []models.AnswerExchange{}
	if err := exchangeCol.
		Find(bson.M{"swapperuser": currentUser.OpenID, "createdat": bson.M{"$lt": lastDate}}).
		Sort("-createdat").
		Limit(20).
		All(&exchanges); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answerIDs := []bson.ObjectId{}
	questionIDs := []bson.ObjectId{}
	for _, v := range exchanges {
		answerIDs = append(answerIDs, v.Swapper)
		questionIDs = append(questionIDs, v.Question)
	}

	answerIDs = utils.ObjectIDRemoveRep(answerIDs)
	questionIDs = utils.ObjectIDRemoveRep(questionIDs)

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

	results := []myResponse{}
	for _, v := range exchanges {
		r := myResponse{}
		if q, ok := questionMap[v.Question]; ok {
			r.Question = q
		}
		if a, ok := answerMap[v.Swapper]; ok {
			a.Length = len([]rune(a.Content))
			a.Content = ""
			r.Answer = a
		}
		results = append(results, r)
	}

	c.JSON(http.StatusOK, results)
}
