package dashboard

import (
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/routers"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type groupTopic struct {
	ID         bson.ObjectId `json:"id" bson:"_id,omitempty"`
	QuestionID bson.ObjectId `json:"questionID"` // 问题 id
	GroupID    string        `json:"groupID"`    // 群 id
	AddUsers   []string      `json:"addUsers"`   // 参加回答的人
	Answers    []groupAnswer `json:"answers"`    // 群里的答案
	CreatedAt  time.Time     `json:"createdAt"`
	Content    string        `json:"content"`
}

type groupAnswer struct {
	ID        bson.ObjectId `bson:"_id"`
	OpenID    string
	CreatedAt time.Time
}

type topicRequest struct {
	Page           int
	OrderBy        string
	OrderDirection int
	FilterBy       string
	FilterValue    int
}

type topicResponse struct {
	GroupTopics []groupTopic `json:"groupTopics"`
	Total       int          `json:"total"`
}

func GroupTopics(c *gin.Context) {

	var params topicRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	groupCol := cols.Groups()

	limit := 20

	sort := "createdat"
	if params.OrderBy == "addUsers.length" {
		sort = "addusersCount"
	} else if params.OrderBy == "level" {
		sort = "level"
	}

	groupTopics := []groupTopic{}
	skip := (params.Page - 1) * limit
	if skip < 0 {
		skip = 0
	}

	direction := -1
	if params.OrderDirection == 1 {
		direction = 1
	}

	query := groupCol.Find(nil)
	total, err := query.Count()
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	groupCol.Pipe(
		[]bson.M{
			bson.M{
				"$lookup": bson.M{
					"from":         "questions",
					"localField":   "questionid",
					"foreignField": "_id",
					"as":           "question",
				},
			},
			bson.M{
				"$unwind": "$question",
			},
			bson.M{
				"$project": bson.M{
					"_id":           1,
					"questionid":    1,
					"answerid":      1,
					"groupid":       1,
					"addusers":      1,
					"addusersCount": bson.M{"$size": "$addusers"},
					"answers":       1,
					"createdat":     1,
					"content":       "$question.content",
				},
			},
			bson.M{"$sort": bson.M{sort: direction}},
			bson.M{"$skip": skip},
			bson.M{"$limit": limit},
		},
	).All(&groupTopics)

	resp := topicResponse{GroupTopics: groupTopics, Total: total}

	c.JSON(http.StatusOK, resp)
}

type group struct {
	GroupID string `json:"groupID"` // 群 id
	Count   int    `json:"count"`   // 群里分享的问题数
}

type groupRequest struct {
	Page           int
	OrderBy        string
	OrderDirection int
	FilterBy       string
	FilterValue    int
}

type groupResponse struct {
	GroupList []group `json:"groupList"`
	Total     int     `json:"total"`
}

func GroupList(c *gin.Context) {

	var params groupRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	groupCol := cols.Groups()

	limit := 20

	sort := "count"

	groups := []group{}
	skip := (params.Page - 1) * limit
	if skip < 0 {
		skip = 0
	}

	direction := -1
	if params.OrderDirection == 1 {
		direction = 1
	}

	groupCol.Pipe(
		[]bson.M{
			bson.M{
				"$group": bson.M{
					"_id":   bson.M{"groupid": "$groupid"},
					"count": bson.M{"$sum": 1},
				},
			},
			bson.M{
				"$project": bson.M{
					"_id":     1,
					"groupid": "$_id.groupid",
					"count":   1,
				},
			},
			bson.M{"$sort": bson.M{
				sort: direction,
			},
			},
			bson.M{"$skip": skip},
			bson.M{"$limit": limit},
		},
	).All(&groups)

	var groupCount bson.M
	cols.Groups().Pipe([]bson.M{
		bson.M{"$group": bson.M{
			"_id": "$groupid",
		}},
		bson.M{"$count": "total"},
	}).One(&groupCount)

	resp := groupResponse{GroupList: groups, Total: groupCount["total"].(int)}

	c.JSON(http.StatusOK, resp)
}

// func DeleteQuestion(c *gin.Context) {

// 	var params routers.IDRequest
// 	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
// 		log.Println(err)
// 		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
// 		return
// 	}

// 	session, cols := db.Default()
// 	defer session.Close()

// 	col := cols.Questions()

// 	set := bson.M{"status": models.StatusDelete}
// 	err := col.Update(bson.M{"_id": bson.ObjectIdHex(params.ID)}, bson.M{"$set": set})
// 	if err != nil {
// 		log.Println(err)
// 		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
// 		return
// 	}

// 	c.JSON(http.StatusOK, routers.OK())
// }

// type detailResponse struct {
// 	Question models.Question        `json:"question"`
// 	Answers  []models.AnswerDisplay `json:"answers"`
// }

// func QuestionDetail(c *gin.Context) {

// 	var params routers.IDRequest
// 	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
// 		log.Println(err)
// 		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
// 		return
// 	}

// 	session, cols := db.Default()
// 	defer session.Close()

// 	question, err := routerUtils.FindQuestion(cols, bson.ObjectIdHex(params.ID))
// 	if err != nil {
// 		log.Println(err)
// 		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
// 		return
// 	}

// 	answerCol := cols.Answers()
// 	answers := models.Answers{}
// 	query := bson.M{"questionid": question.ID, "status": bson.M{"$ne": models.StatusDelete}}
// 	if err := answerCol.Find(query).Sort("-createdat").All(&answers); err != nil {
// 		log.Println(err)
// 		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
// 		return
// 	}

// 	userIDs := make([]string, len(answers))
// 	for i, v := range answers {
// 		userIDs[i] = v.OpenID
// 	}

// 	users, err := routerUtils.FindUsersAsMap(cols, userIDs)
// 	if err != nil {
// 		log.Println(err)
// 		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
// 		return
// 	}

// 	answerDisplays := []models.AnswerDisplay{}
// 	for _, v := range answers {
// 		if u, ok := users[v.OpenID]; ok {
// 			a := models.AnswerDisplay{Answer: v, User: u.ToLean()}
// 			answerDisplays = append(answerDisplays, a)
// 		}
// 	}

// 	resp := detailResponse{Question: *question, Answers: answerDisplays}

// 	c.JSON(http.StatusOK, resp)
// }
