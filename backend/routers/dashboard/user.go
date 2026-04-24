package dashboard

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

type pipeCount struct {
	ID    string `bson:"_id"`
	Count int
}

type usersResponse struct {
	Users []dashboardUser `json:"users"`
	Total int             `json:"total"`
}

type dashboardUser struct {
	models.User

	AnswerCount   int `json:"answerCount"`
	ExchangeCount int `json:"exchangeCount"`
}

func Users(c *gin.Context) {

	var params queryRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	userCol := cols.Users()
	users := models.Users{}

	query := userCol.Find(nil)
	total, err := query.Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	limit := 20

	if params.OrderBy == "" || params.OrderBy == "updatedAt" {
		sort := "updatedat"

		skip := (params.Page - 1) * limit
		if skip < 0 {
			skip = 0
		}

		direction := "-"
		if params.OrderDirection == 1 {
			direction = "+"
		}

		sort = direction + sort
		if len([]rune(sort)) > 1 {
			query = query.Sort(sort)
		}

		query = query.Limit(limit).Skip(skip)

		if err := query.All(&users); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}

		openids := users.MapToOpenID()

		answerExchangeUserListCol := cols.UserAnswerCounters()
		exchangeCount := []models.UserAnswerCounters{}
		answerExchangeUserListCol.Find(bson.M{"openid": bson.M{"$in": openids}}).All(&exchangeCount)

		answerExchangeMap := map[string]models.UserAnswerCounters{}
		for _, v := range exchangeCount {
			answerExchangeMap[v.OpenID] = v
		}

		results := []dashboardUser{}

		for _, user := range users {
			r := dashboardUser{User: user}
			if c, ok := answerExchangeMap[user.OpenID]; ok {
				r.AnswerCount = c.AnswerCount
				r.ExchangeCount = c.AllExchangeCount
			}
			results = append(results, r)
		}

		resp := usersResponse{Users: results, Total: total}

		c.JSON(http.StatusOK, resp)

	} else if params.OrderBy == "exchangeCount" || params.OrderBy == "answerCount" {
		answerExchangeUserListCol := cols.UserAnswerCounters()
		exchangeCount := []models.UserAnswerCounters{}
		direction := "-"
		if params.OrderDirection == 1 {
			direction = "+"
		}
		orderBy := ""
		if params.OrderBy == "exchangeCount" {
			orderBy = "allexchangecount"
		} else if params.OrderBy == "answerCount" {
			orderBy = "answercount"
		}
		sort := direction + orderBy
		skip := (params.Page - 1) * limit
		if skip < 0 {
			skip = 0
		}
		answerExchangeUserListCol.Find(nil).Sort(sort).Limit(limit).Skip(skip).All(&exchangeCount)

		openIDs := make([]string, len(exchangeCount))
		answerExchangeMap := map[string]models.UserAnswerCounters{}
		for i, v := range exchangeCount {
			openid := v.OpenID
			answerExchangeMap[openid] = v
			openIDs[i] = openid
		}

		users, err := routerUtils.FindUsersAsMap(cols, openIDs)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}

		results := []dashboardUser{}

		for _, openid := range openIDs {
			if u, ok := users[openid]; ok {
				r := dashboardUser{User: u}
				if c, ok := answerExchangeMap[openid]; ok {
					r.ExchangeCount = c.AllExchangeCount
					r.AnswerCount = c.AnswerCount
				}
				results = append(results, r)
			}
		}

		resp := usersResponse{Users: results, Total: total}

		c.JSON(http.StatusOK, resp)
	}
}

type userDetailRequest struct {
	queryRequest
	routers.OpenIDRequest
}

type userDetailResponse struct {
	models.User

	AnswerCount   int `json:"answerCount"`
	ExchangeCount int `json:"exchangeCount"`

	QAs   []models.QA `json:"qas"`
	Total int         `json:"total"`
}

func UserDetail(c *gin.Context) {

	var params userDetailRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	user, err := routerUtils.FindUser(cols, params.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answerCol := cols.Answers()
	answers := models.Answers{}

	query := answerCol.Find(bson.M{"openid": params.OpenID, "status": bson.M{"$ne": models.StatusDelete}})

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

	query = query.Skip(skip)

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

	if err := query.All(&answers); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	answerIDs := answers.MapToObjectID(func(v models.Answer) bson.ObjectId {
		return v.QuestionID
	})

	answerExchangeCol := cols.AnswerExchange()
	exchangeCount, err := answerExchangeCol.Find(bson.M{"$or": []bson.M{bson.M{"sourceuser": params.OpenID}, bson.M{"swapperuser": params.OpenID}}}).Count()
	if err != nil {
		log.Println(err)
	}

	questionCol := cols.Questions()
	questions := []models.Question{}
	if err := questionCol.Find(bson.M{"_id": bson.M{"$in": answerIDs}}).All(&questions); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	questionMap := map[bson.ObjectId]models.Question{}
	for _, v := range questions {
		questionMap[v.ID] = v
	}

	qas := []models.QA{}
	for _, v := range answers {
		if q, ok := questionMap[v.QuestionID]; ok {
			qa := models.QA{Question: q, Answer: v}
			qas = append(qas, qa)
		}
	}

	resp := userDetailResponse{
		User:          *user,
		AnswerCount:   len(answers),
		ExchangeCount: exchangeCount,
		QAs:           qas,
		Total:         total,
	}
	c.JSON(http.StatusOK, resp)
}
