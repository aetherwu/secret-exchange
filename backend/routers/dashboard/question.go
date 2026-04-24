package dashboard

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

type questionAnswerCount struct {
	ID    bson.ObjectId `bson:"_id"`
	Count int
}

type queryRequest struct {
	Page           int
	OrderBy        string
	OrderDirection int
	FilterBy       string
	FilterValue    int
}

type questionsResponse struct {
	Questions []question `json:"questions"`
	Total     int        `json:"total"`
}

type question struct {
	ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Content     string        `json:"content"`
	AnswerCount int           `json:"answerCount"`
	Level       int           `json:"level"`
	Status      int           `json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

func Questions(c *gin.Context) {

	var params queryRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	questionCol := cols.Questions()

	limit := 200

	sort := "createdat"
	if params.OrderBy == "answerCount" {
		sort = "answercount"
	} else if params.OrderBy == "level" {
		sort = "level"
	}

	questions := []question{}
	skip := (params.Page - 1) * limit
	if skip < 0 {
		skip = 0
	}

	direction := "-"
	if params.OrderDirection == 1 {
		direction = "+"
	}

	selector := bson.M{"status": bson.M{"$ne": models.StatusDelete}}
	if params.FilterBy != "" {
		selector[params.FilterBy] = params.FilterValue
	}

	query := questionCol.Find(selector)
	total, err := query.Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	sort = direction + sort
	if len([]rune(sort)) > 1 {
		query = query.Sort(sort)
	}

	query = query.Limit(limit).Skip(skip)

	if err := query.All(&questions); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	resp := questionsResponse{Questions: questions, Total: total}

	c.JSON(http.StatusOK, resp)
}

type addRequest struct {
	Content string `binding:"required"`
}

func AddQuestion(c *gin.Context) {

	var params addRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	col := cols.Questions()

	questions := []models.Question{}
	if err := col.Find(bson.M{"content": params.Content}).All(&questions); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	if len(questions) > 0 {
		q := questions[0]
		if q.Status == models.StatusDelete {
			set := bson.M{"status": models.StatusNormal}
			err := col.Update(bson.M{"_id": q.ID}, bson.M{"$set": set})
			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
				return
			}
			c.JSON(http.StatusOK, routers.OK())
			return
		}
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	now := time.Now()

	if err := col.Insert(models.Question{
		ID:        bson.NewObjectId(),
		Content:   params.Content,
		Status:    models.StatusNormal,
		Level:     0,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	c.JSON(http.StatusOK, routers.OK())
}

func DeleteQuestion(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	col := cols.Questions()

	set := bson.M{"status": models.StatusDelete}
	err := col.Update(bson.M{"_id": bson.ObjectIdHex(params.ID)}, bson.M{"$set": set})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	c.JSON(http.StatusOK, routers.OK())
}

type editRequest struct {
	ID      string `binding:"required"`
	Content string `binding:"required"`
}

func EditQuestion(c *gin.Context) {

	var params editRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	col := cols.Questions()

	set := bson.M{"updatedat": time.Now(), "content": params.Content}
	err := col.Update(bson.M{"_id": bson.ObjectIdHex(params.ID)}, bson.M{"$set": set})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	c.JSON(http.StatusOK, routers.OK())
}

type detailResponse struct {
	Question models.Question        `json:"question"`
	Answers  []models.AnswerDisplay `json:"answers"`
}

func QuestionDetail(c *gin.Context) {

	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	question, err := routerUtils.FindQuestion(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answerCol := cols.Answers()
	answers := models.Answers{}
	query := bson.M{"questionid": question.ID, "status": bson.M{"$ne": models.StatusDelete}}
	if err := answerCol.Find(query).Sort("-createdat").All(&answers); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	userIDs := make([]string, len(answers))
	for i, v := range answers {
		userIDs[i] = v.OpenID
	}

	users, err := routerUtils.FindUsersAsMap(cols, userIDs)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answerDisplays := []models.AnswerDisplay{}
	for _, v := range answers {
		if u, ok := users[v.OpenID]; ok {
			a := models.AnswerDisplay{Answer: v, User: u.ToLean()}
			answerDisplays = append(answerDisplays, a)
		}
	}

	resp := detailResponse{Question: *question, Answers: answerDisplays}

	c.JSON(http.StatusOK, resp)
}

type findQuestionsRequest struct {
	queryRequest
	Content string `binding:"required"`
}

func FindQuestions(c *gin.Context) {

	var params findQuestionsRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	col := cols.Questions()

	limit := 200

	sort := "createdat"
	if params.OrderBy == "answerCount" {
		sort = "answercount"
	} else if params.OrderBy == "level" {
		sort = "level"
	}

	questions := []question{}
	skip := (params.Page - 1) * limit
	if skip < 0 {
		skip = 0
	}

	direction := "-"
	if params.OrderDirection == 1 {
		direction = "+"
	}

	selector := bson.M{
		"content": bson.M{"$regex": bson.RegEx{Pattern: params.Content, Options: "i"}},
		"status":  bson.M{"$ne": models.StatusDelete},
	}
	if params.FilterBy != "" {
		selector[params.FilterBy] = params.FilterValue
	}

	query := col.Find(selector)
	total, err := query.Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	sort = direction + sort
	if len([]rune(sort)) > 1 {
		query = query.Sort(sort)
	}

	query = query.Limit(limit).Skip(skip)

	query.All(&questions)

	resp := questionsResponse{}
	resp.Questions = questions
	resp.Total = total

	c.JSON(http.StatusOK, resp)
}

type editQuestionLevelRequest struct {
	Questions []string `binding:"required"`
	Level     int
}

func EditQuestionLevel(c *gin.Context) {

	var params editQuestionLevelRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	col := cols.Questions()

	questionIDs := make([]bson.ObjectId, len(params.Questions))
	for i, v := range params.Questions {
		questionIDs[i] = bson.ObjectIdHex(v)
	}

	_, err := col.UpdateAll(bson.M{"_id": bson.M{"$in": questionIDs}}, bson.M{"$set": bson.M{"level": params.Level}})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	c.JSON(http.StatusOK, routers.OK())
}
