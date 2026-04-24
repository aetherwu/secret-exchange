package answer

import (
	"log"
	"net/http"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/draw"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"dagong.in/delphi-web-server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type createRequest struct {
	ID      string `binding:"required"`
	Answer  string `binding:"required"`
	Private int
}

// Create 新建答案
func Create(c *gin.Context) {

	var params createRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	// 查找问题
	question, err := routerUtils.FindQuestion(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 查询是否已经回答过
	answered, err := routerUtils.FindAnswerWithQuesiton(cols, question.ID, auth.OpenID)
	if answered != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "已经回答过此问题"))
		return
	}

	// 查询今天回答次数
	limitCol := cols.QuestionLimit()
	questionLimit := models.QuestionLimit{}
	limitCol.Find(bson.M{"openid": auth.OpenID}).One(&questionLimit)

	limit := 5
	if questionLimit.Limit != 0 {
		limit -= questionLimit.Limit
	}

	// 超过5次
	if limit == 0 {
		c.AbortWithStatusJSON(http.StatusOK, routers.NewError(1111, "每天最多回答5个问题，请明天再来"))
		return
	}

	now := time.Now()
	answer := models.Answer{
		ID:         bson.NewObjectId(),
		OpenID:     auth.OpenID,
		QuestionID: question.ID,
		Content:    params.Answer,
		Status:     models.StatusNormal,
		Privacy:    question.Level,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	qa := models.QADisplay{Question: *question, Answer: models.AnswerDisplay{
		Answer: answer,
		User:   auth.ToLean(),
	}}

	// 上传分享图片
	image, err := draw.GenWeChatShareImageOfDraw(qa)
	if err != nil {
		log.Println(err)
	} else {
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
		} else {
			answer.WeChatShareImage = key
			answer.WeChatShareImageStatus = models.WeChatShareImageStatusGenerated
		}
	}

	col := cols.Answers()
	if err := col.Insert(&answer); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// 更新问题的答案数
	questionCol := cols.Questions()
	if err := questionCol.Update(bson.M{"_id": question.ID}, bson.M{"$inc": bson.M{"answercount": 1}}); err != nil {
		log.Println(err)
	}

	// 更新用户的答案数
	userAnswerCountersCol := cols.UserAnswerCounters()
	userAnswerCountersCol.Upsert(bson.M{"openid": auth.OpenID}, bson.M{"$inc": bson.M{
		"answercount": 1,
	}})

	// 更新 activity

	// update personal activity
	// 在动态里显示我自己的回答
	// 不受隐私控制，仅自己可见
	activitysCol := cols.Activitys()
	activity := models.Feed{
		OpenID:       auth.OpenID,
		Type:         0,
		Question:     question.ID,
		Answer:       answer.ID,
		AnswerOpenID: answer.OpenID,
		CreatedAt:    now,
	}
	activitysCol.Insert(activity)

	// broadcast only follower would receive these broadcast
	// 向关注者群发动态
	// 仅群发可公开交换的 -1 0 1 三类
	// 关注者可在自己的动态里看到 xxx 的新答案动态
	if question.Level < 2 {

		// 更新粉丝的 activitys
		followsCol := cols.Follows()
		follow := models.Follows{}
		followsCol.Find(bson.M{"openid": auth.OpenID}).One(&follow)

		var activitys []interface{}
		for _, v := range follow.Followers {
			activity := models.Feed{
				OpenID:       v.OpenID,
				Type:         2,
				Question:     question.ID,
				Answer:       answer.ID,
				AnswerOpenID: answer.OpenID,
				CreatedAt:    now,
			}
			activitys = append(activitys, activity)
		}

		if len(activitys) > 0 {
			activitysCol.Insert(activitys...)
		}
	}

	// 更新今天回答次数
	update := bson.M{
		"$inc": bson.M{"limit": 1},
		"$set": bson.M{"expiredat": utils.ZeroTomorrow()},
	}
	if _, err := limitCol.Upsert(bson.M{"openid": auth.OpenID}, update); err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, gin.H{"id": answer.ID})
}

// WeChatShareImage ...
func WeChatShareImage(c *gin.Context) {
	var params routers.IDRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	session, cols := db.Default()
	defer session.Close()

	answer, err := routerUtils.FindAnswer(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	question, err := routerUtils.FindQuestion(cols, answer.QuestionID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	user, err := routerUtils.FindUser(cols, answer.OpenID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	qa := models.QADisplay{
		Question: *question,
		Answer: models.AnswerDisplay{
			Answer: *answer,
			User:   user.ToLean(),
		}}

	if answer.WeChatShareImageStatus < models.WeChatShareImageStatusGenerating {
		image, err := draw.GenWeChatShareImageOfDraw(qa)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusOK, gin.H{"weChatShareImage": "https://gamecard.dagong.in/fe4fcc28cc157b8054d96d59a8b3b8ab.png"})
			return
		}
		key, err := routers.UploadImage(image, "jpeg")
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusOK, gin.H{"weChatShareImage": "https://gamecard.dagong.in/fe4fcc28cc157b8054d96d59a8b3b8ab.png"})
			return
		}

		col := cols.Answers()
		answer.WeChatShareImage = key
		if err := col.UpdateId(answer.ID, bson.M{"$set": bson.M{
			"wechatshareimage":       key,
			"wechatshareimagestatus": models.WeChatShareImageStatusGenerated,
		}}); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}

	}

	c.JSON(http.StatusOK, gin.H{"weChatShareImage": routerUtils.QiniuKeyToURL(answer.WeChatShareImage)})
}
