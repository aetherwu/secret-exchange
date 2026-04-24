package answer

import (
	"log"
	"net/http"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"dagong.in/delphi-web-server/routers/wechat"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type likeRequest struct {
	ID    string `binding:required`
	Emoji int
}

// Like 点赞
func Like(c *gin.Context) {

	var params likeRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	// 查询答案是否存在
	answer, err := routerUtils.FindAnswer(cols, bson.ObjectIdHex(params.ID))
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 查找答案的创建者
	user, err := routerUtils.FindUser(cols, answer.OpenID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	// 查找问题是否存在
	question, err := routerUtils.FindQuestion(cols, answer.QuestionID)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.NotFoundError())
		return
	}

	likeCol := cols.AnswerLikes()

	// 是否点过赞
	isLike := routerUtils.IsLike(cols, answer.ID, auth.OpenID)

	// 如果点过赞
	if isLike {
		// 取消点赞
		pull := bson.M{"$pull": bson.M{"likes": bson.M{"openid": auth.OpenID}}}
		if err := likeCol.Update(bson.M{"answer": answer.ID}, pull); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}
		answer.Likes--
		if answer.Likes < 0 {
			answer.Likes = 0
		}
	} else {
		// 点赞
		push := bson.M{"$push": bson.M{"likes": models.AnswerLikeType{OpenID: auth.OpenID, Type: params.Emoji}}}
		if _, err := likeCol.Upsert(bson.M{"answer": answer.ID}, push); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
			return
		}
		answer.Likes++
	}

	// 更新
	update := bson.M{"$set": bson.M{"likes": answer.Likes}}
	if err := cols.Answers().Update(bson.M{"_id": answer.ID}, update); err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// 是否要发送模板消息
	if user.OpenID != auth.OpenID {
		page := "pages/answer?id=" + answer.ID.Hex()
		msg := wechat.LikeNotify{
			Question: question.Content,
			Answer:   answer.Content,
			Nickname: auth.Nickname,
		}
		if err := wechat.SendNotify(user.OpenID, page, msg); err != nil {
			log.Println(err)
		}
	}

	c.JSON(http.StatusOK, routers.OK())
}
