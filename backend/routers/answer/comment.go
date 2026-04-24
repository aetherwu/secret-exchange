package answer

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/routerUtils"
	"dagong.in/delphi-web-server/routers/wechat"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
)

type commentRequest struct {
	ID      string `binding:"required"`
	Content string
	Images  []routers.UploadCallbackModel
	Audio   *models.CommentAudioInfo
	Reply   string
}

// Comment 创建评论
func Comment(c *gin.Context) {
	var params commentRequest
	if err := c.ShouldBindWith(&params, binding.JSON); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, routers.ParamsError())
		return
	}

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	// 查找答案是否存在
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

	comment := models.AnswerComment{
		OpenID:    auth.OpenID,
		Content:   params.Content,
		Audio:     params.Audio,
		Reply:     params.Reply,
		CreatedAt: time.Now(),
	}

	// 是否有图片
	if len(params.Images) > 0 {
		imageInfos := []models.CommentImageInfo{}
		for _, v := range params.Images {
			w, _ := strconv.ParseFloat(v.Width, 32)
			h, _ := strconv.ParseFloat(v.Height, 32)
			info := models.CommentImageInfo{
				Key:    v.Key,
				Size:   v.Fsize,
				Width:  w,
				Height: h,
			}
			imageInfos = append(imageInfos, info)
		}
		comment.Images = imageInfos
	}

	// 插入评论
	commentMapCol := cols.AnswerCommentMap()
	push := bson.M{"comments": comment}
	_, err = commentMapCol.Upsert(bson.M{"answer": answer.ID}, bson.M{"$push": push})
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusOK, routers.DefaultError())
		return
	}

	// 判断是否需要发送模板消息
	if user.OpenID != auth.OpenID || (params.Reply != "" && params.Reply != auth.OpenID) {
		page := "pages/answer?id=" + answer.ID.Hex()
		content := ""
		if len(comment.Content) > 0 {
			content = comment.Content
		}
		if len(comment.Images) > 0 {
			content += " [图片]"
		}
		if comment.Audio != nil {
			content += " [音频]"
		}
		msg := wechat.CommentNotify{
			Type:     "已回复",
			Nickname: auth.Nickname,
			Content:  string([]rune(content)[:20]),
		}
		if params.Reply != "" {
			if err := wechat.SendNotify(params.Reply, page, msg); err != nil {
				log.Println(err)
			}
		} else {
			if err := wechat.SendNotify(user.OpenID, page, msg); err != nil {
				log.Println(err)
			}
		}
	}

	c.JSON(http.StatusOK, routers.OK())
}
