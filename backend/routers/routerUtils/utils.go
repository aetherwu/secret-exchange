package routerUtils

import (
	"errors"
	"log"
	"time"

	"dagong.in/delphi-web-server/config"
	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/utils"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// FindQuestion 根据 question id 查找问题
func FindQuestion(cols db.Collections, id bson.ObjectId) (*models.Question, error) {
	col := cols.Questions()

	var result *models.Question
	if err := col.FindId(id).One(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// FindAnswer 根据 answer id 查找答案
func FindAnswer(cols db.Collections, answer bson.ObjectId) (*models.Answer, error) {
	col := cols.Answers()

	var result *models.Answer
	if err := col.FindId(answer).One(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// FindQA 根据 answe id 查找问题和答案
func FindQA(cols db.Collections, answer bson.ObjectId) (*models.Question, *models.Answer, error) {
	a, err := FindAnswer(cols, answer)
	if err != nil {
		return nil, nil, err
	}
	q, err := FindQuestion(cols, a.QuestionID)
	if err != nil {
		return nil, nil, err
	}
	return q, a, nil
}

// FindAnswerWithQuesiton 根据 question id 查找答案
func FindAnswerWithQuesiton(cols db.Collections, id bson.ObjectId, openID string) (*models.Answer, error) {
	col := cols.Answers()

	var result *models.Answer
	if err := col.Find(bson.M{"questionid": id, "openid": openID}).One(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// FindUser 查找单个用户
func FindUser(cols db.Collections, openID string) (*models.User, error) {
	col := cols.Users()

	var result *models.User
	if err := col.Find(bson.M{"openid": openID}).One(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// FindUsers 批量查找用户
func FindUsers(cols db.Collections, openIDs []string) ([]models.User, error) {
	users := []models.User{}
	if len(openIDs) == 0 {
		return users, nil
	}

	col := cols.Users()
	if err := col.Find(bson.M{"openid": bson.M{"$in": openIDs}}).All(&users); err != nil {
		return nil, err
	}

	return users, nil
}

// FindUsersAsMap 批量查找用户并转换成map
func FindUsersAsMap(cols db.Collections, openIDs []string) (map[string]models.User, error) {
	users, err := FindUsers(cols, openIDs)
	if err != nil {
		return nil, err
	}

	userMap := map[string]models.User{}
	for _, user := range users {
		userMap[user.OpenID] = user
	}
	return userMap, nil
}

// IsExchanged 答案是否单独交换过
func IsExchanged(cols db.Collections, question bson.ObjectId, source string, swapper string) *models.AnswerExchange {

	col := cols.AnswerExchange()

	var exchanged *models.AnswerExchange
	or := []bson.M{
		bson.M{"sourceuser": source, "swapperuser": swapper},
		bson.M{"sourceuser": swapper, "swapperuser": source},
	}
	err := col.Find(bson.M{"question": question, "$or": or}).One(&exchanged)
	if err != nil {
		log.Println(err)
		return nil
	}
	return exchanged
}

// GetAnswerComments 返回答案的所有评论
func GetAnswerComments(cols db.Collections, answer bson.ObjectId) ([]models.AnswerCommentDisplay, error) {
	commentDisplays := []models.AnswerCommentDisplay{}
	commentMaps := []models.CommentMapPipeGroup{}
	answerCommentCol := cols.AnswerCommentMap()
	answerCommentCol.Pipe([]bson.M{
		bson.M{"$match": bson.M{"answer": answer}},
		bson.M{"$unwind": "$comments"},
		bson.M{"$sort": bson.M{"comments.createdat": 1}},
	}).All(&commentMaps)

	userIDs := []string{}
	for _, v := range commentMaps {
		userIDs = append(userIDs, v.Comments.OpenID)
		if v.Comments.IsReply() {
			userIDs = append(userIDs, v.Comments.Reply)
		}
	}

	userIDs = utils.StringRemoveRep(userIDs)

	users, err := FindUsersAsMap(cols, userIDs)
	if err != nil {
		return commentDisplays, nil
	}

	for _, v := range commentMaps {
		if u, ok := users[v.Comments.OpenID]; ok {
			if v.Comments.Audio != nil {
				v.Comments.Audio.Src = QiniuKeyToURL(v.Comments.Audio.Key)
			}

			for i, image := range v.Comments.Images {
				v.Comments.Images[i].Src = QiniuKeyToImageDefaultQualityURL(image.Key)
			}

			comment := models.AnswerCommentDisplay{AnswerComment: v.Comments, LeanUser: u.ToLean()}
			if v.Comments.IsReply() {
				if r, ok := users[v.Comments.Reply]; ok {
					l := r.ToLean()
					comment.Reply = &l
				}
			}
			commentDisplays = append(commentDisplays, comment)
		}
	}

	return commentDisplays, nil
}

// IsLike 答案是否点过赞
func IsLike(cols db.Collections, answer bson.ObjectId, openID string) bool {
	col := cols.AnswerLikes()

	var like models.AnswerLike
	if err := col.Find(bson.M{
		"answer": answer,
		"likes": bson.M{
			"$elemMatch": bson.M{
				"openid": openID,
			},
		},
	}).One(&like); err != nil {
		return false
	}
	return true
}

// Likes 答案的所有点赞用户
func Likes(cols db.Collections, answer bson.ObjectId) map[int][]models.AnswerLikeUser {
	likes := map[int][]models.AnswerLikeUser{}
	col := cols.AnswerLikes()

	like := models.AnswerLike{}
	if err := col.Find(bson.M{
		"answer": answer,
	}).One(&like); err != nil {
		return likes
	}

	openids := make([]string, len(like.Likes))
	for i, v := range like.Likes {
		openids[i] = v.OpenID
	}

	users, err := FindUsersAsMap(cols, openids)
	if err != nil {
		return likes
	}

	for _, v := range like.Likes {
		if u, ok := users[v.OpenID]; ok {
			ts := likes[v.Type]
			if ts == nil {
				ts = []models.AnswerLikeUser{}
			}
			ts = append(ts, models.AnswerLikeUser{Nickname: u.Nickname, Type: v.Type})
			likes[v.Type] = ts
		}
	}

	return likes
}

func UpdateExchangeUserList(col *mgo.Collection, questionID bson.ObjectId, answer models.Answer, exchange models.Answer) error {
	now := time.Now()
	count, _ := col.Find(bson.M{"openid": exchange.OpenID}).Count()
	if count == 0 {
		err := col.Insert(bson.M{"openid": exchange.OpenID})
		if err != nil {
			return err
		}
	}

	answerExchangeUserListUpdate := bson.M{
		"$set": bson.M{
			"exchanges.$.openid":     answer.OpenID,
			"exchanges.$.questionid": questionID,
			"exchanges.$.answerid":   answer.ID,
			"exchanges.$.updatedat":  now,
		},
		"$inc": bson.M{"exchanges.$.count": 1},
	}

	if err := col.Update(bson.M{"openid": exchange.OpenID, "exchanges.openid": answer.OpenID}, answerExchangeUserListUpdate); err != nil {
		log.Println(err)
		if err := col.Update(bson.M{"openid": exchange.OpenID}, bson.M{"$push": bson.M{"exchanges": bson.M{
			"openid":     answer.OpenID,
			"questionid": questionID,
			"answerid":   answer.ID,
			"updatedat":  now,
			"count":      1,
		}}}); err != nil {
			return err
		}
	}

	return nil
}

func UpdateExchangeAtUser(col *mgo.Collection, questionID bson.ObjectId, answer models.Answer, exchange models.Answer) error {
	now := time.Now()
	count, _ := col.Find(bson.M{"openid": exchange.OpenID, "user": answer.OpenID}).Count()
	if count == 0 {
		err := col.Insert(models.AnswerExchangeAtUser{
			OpenID:    exchange.OpenID,
			User:      answer.OpenID,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return err
		}
	}

	push := bson.M{"answers": models.AnswerExchangeAtUserDetail{
		QuestionID: questionID,
		AnswerID:   answer.ID,
		CreatedAt:  now,
	}}
	if err := col.Update(bson.M{"openid": exchange.OpenID, "user": answer.OpenID}, bson.M{
		"$push": push,
		"$set":  bson.M{"updatedat": now},
	}); err != nil {
		return err
	}

	return nil
}

var levelWieght map[int]int

// GetQuestionsWeightRandomNext 根据权重获随机取一个问题
func GetQuestionsWeightRandomNext(col *mgo.Collection, questionIDs []bson.ObjectId) (*bson.ObjectId, error) {
	if levelWieght == nil {
		// weight 权重，总和不能小于等于0，可以不是100的倍数
		levelWieght = map[int]int{0: 7500, 1: 1500, 2: 1000, 3: 500, 4: 1}
	}

	questions := []models.Question{}
	err := col.Pipe([]bson.M{
		bson.M{"$match": bson.M{
			"status": bson.M{"$ne": models.StatusDelete},
			"_id":    bson.M{"$in": questionIDs},
		}},
		bson.M{"$project": bson.M{"_id": 1, "level": 1}},
	}).All(&questions)

	if err != nil {
		return nil, err
	}

	categorys := make([]utils.WeightCategory, len(questions))

	for i, v := range questions {
		w := levelWieght[v.Level]
		categorys[i] = utils.WeightCategory{Category: v.ID, Weight: w}
	}
	r, err := utils.NewWeightRandom(categorys)
	if err != nil {
		return nil, err
	}

	n := r.Next().Category

	if n, ok := n.(bson.ObjectId); ok {
		return &n, nil
	}
	return nil, errors.New("not found Category")
}

// QiniuKeyToURL 七牛下载URL
func QiniuKeyToURL(key string) string {
	return config.QiniuDownload + key
}

// QiniuKeyToImageDefaultQualityURL 七牛图片下载 默认画质
func QiniuKeyToImageDefaultQualityURL(key string) string {
	return QiniuKeyToURL(key) + config.QiniuDownloadImageDefaultQuality
}

// QiniuKeyToVideoCover 七牛 视频封面
func QiniuKeyToVideoCover(key string) string {
	return QiniuKeyToURL(key) + "?vframe/jpg/offset/1"
}
