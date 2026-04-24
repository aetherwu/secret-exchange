package db

import (
	"fmt"
	"log"
	"time"

	"dagong.in/delphi-web-server/models"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func addAnswerExchangeCount() {
	session, cols := Default()
	defer session.Close()

	answerCol := cols.Answers()
	answers := []models.Answer{}
	err := answerCol.Find(nil).All(&answers)
	if err != nil {
		fmt.Println(err)
	}

	exchangeCol := cols.AnswerExchange()
	for _, v := range answers {
		be, _ := exchangeCol.Find(bson.M{"source": v.ID}).Count()
		s, _ := exchangeCol.Find(bson.M{"swapper": v.ID}).Count()
		answerCol.Update(bson.M{"_id": v.ID}, bson.M{"$set": bson.M{"exchangecount": s, "beexchangedcount": be}})
		fmt.Println(be, s)
	}
}

func addAnswerExchangeAtUser() {
	session, cols := Default()
	defer session.Close()

	answerExchangeUserListCol := cols.AnswerExchangeUserList()
	answerExchangeUserListCol.RemoveAll(nil)
	answerExchangeAtUserCol := cols.AnswerExchangeAtUser()
	answerExchangeAtUserCol.RemoveAll(nil)

	exchangeCol := cols.AnswerExchange()
	exchanges := []models.AnswerExchange{}
	exchangeCol.Find(nil).All(&exchanges)
	fmt.Println(len(exchanges))
	for _, v := range exchanges {
		updateExchangeUserList(
			answerExchangeUserListCol,
			v.Question,
			models.Answer{OpenID: v.SourceUser, ID: v.Source},
			models.Answer{OpenID: v.SwapperUser, ID: v.Swapper},
			v.CreatedAt)

		updateExchangeUserList(
			answerExchangeUserListCol,
			v.Question,
			models.Answer{OpenID: v.SwapperUser, ID: v.Swapper},
			models.Answer{OpenID: v.SourceUser, ID: v.Source},
			v.CreatedAt)

		updateExchangeAtUser(
			answerExchangeAtUserCol,
			v.Question,
			models.Answer{OpenID: v.SourceUser, ID: v.Source},
			models.Answer{OpenID: v.SwapperUser, ID: v.Swapper},
			v.CreatedAt)

		updateExchangeAtUser(
			answerExchangeAtUserCol,
			v.Question,
			models.Answer{OpenID: v.SwapperUser, ID: v.Swapper},
			models.Answer{OpenID: v.SourceUser, ID: v.Source},
			v.CreatedAt)
	}

}

func updateExchangeUserList(col *mgo.Collection, questionID bson.ObjectId, answer models.Answer, exchange models.Answer, date time.Time) error {
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
			"exchanges.$.updatedat":  date,
		},
		"$inc": bson.M{"exchanges.$.count": 1},
	}

	if err := col.Update(bson.M{"openid": exchange.OpenID, "exchanges.openid": answer.OpenID}, answerExchangeUserListUpdate); err != nil {
		log.Println(err)
		if err := col.Update(bson.M{"openid": exchange.OpenID}, bson.M{"$push": bson.M{"exchanges": bson.M{
			"openid":     answer.OpenID,
			"questionid": questionID,
			"answerid":   answer.ID,
			"updatedat":  date,
			"count":      1,
		}}}); err != nil {
			return err
		}
	}

	return nil
}

func updateExchangeAtUser(col *mgo.Collection, questionID bson.ObjectId, answer models.Answer, exchange models.Answer, date time.Time) error {
	count, _ := col.Find(bson.M{"openid": exchange.OpenID, "user": answer.OpenID}).Count()
	if count == 0 {
		err := col.Insert(models.AnswerExchangeAtUser{
			OpenID:    exchange.OpenID,
			User:      answer.OpenID,
			CreatedAt: date,
			UpdatedAt: date,
		})
		if err != nil {
			return err
		}
	}

	push := bson.M{"answers": models.AnswerExchangeAtUserDetail{
		QuestionID: questionID,
		AnswerID:   answer.ID,
		CreatedAt:  date,
	}}
	if err := col.Update(bson.M{"openid": exchange.OpenID, "user": answer.OpenID}, bson.M{
		"$push": push,
		"$set":  bson.M{"updatedat": date},
	}); err != nil {
		return err
	}

	return nil
}

func addAnswerTotalExchangeCount() {
	session, cols := Default()
	defer session.Close()

	answerTotalExchangeCountCol := cols.UserAnswerCounters()

	userCol := cols.Users()
	users := []models.User{}
	err := userCol.Find(nil).All(&users)
	if err != nil {
		fmt.Println(err)
	}

	exchangeCol := cols.AnswerExchange()
	answerCol := cols.Answers()
	for _, v := range users {
		be, _ := exchangeCol.Find(bson.M{"sourceuser": v.OpenID}).Count()
		s, _ := exchangeCol.Find(bson.M{"swapperuser": v.OpenID}).Count()
		as, _ := answerCol.Find(bson.M{"openid": v.OpenID, "status": bson.M{"$gt": -1}}).Count()

		answerTotalExchangeCountCol.Insert(models.UserAnswerCounters{
			OpenID:           v.OpenID,
			AnswerCount:      as,
			ExchangeCount:    s,
			BeExchangedCount: be,
			AllExchangeCount: s + be,
		})
	}
}

func addQuestionAnswerCount() {
	session, cols := Default()
	defer session.Close()

	questionCol := cols.Questions()
	questions := []models.Question{}
	questionCol.Find(nil).All(&questions)

	answerCol := cols.Answers()
	for _, v := range questions {
		c, _ := answerCol.Find(bson.M{"questionid": v.ID}).Count()
		questionCol.Update(bson.M{"_id": v.ID}, bson.M{"$set": bson.M{"answercount": c}})
	}

}

func addGroupQuestionID() {
	// session, cols := Default()
	// defer session.Close()

	// groupCol := cols.Groups()
	// gorups := []models.Group{}
	// groupCol.Find(nil).All(&gorups)

	// answerCol := cols.Answers()
	// for _, v := range gorups {
	// 	var answer *models.Answer
	// 	if err := answerCol.FindId(v.AnswerID).One(&answer); err != nil {
	// 		log.Println(err)
	// 	} else {
	// 		groupCol.Update(bson.M{"_id": v.ID}, bson.M{"$set": bson.M{"questionid": answer.QuestionID}})
	// 	}
	// }
}

func addAnswerPrivacy() {
	session, cols := Default()
	defer session.Close()

	answers := []models.Answer{}
	cols.Answers().Find(nil).All(&answers)
	for _, v := range answers {
		cols.Answers().Update(bson.M{"_id": v.ID}, bson.M{"$set": bson.M{"privacy": 0}})
	}
}

func resetAnswerQRCode() {
	session, cols := Default()
	defer session.Close()

	cols.Answers().UpdateAll(nil, bson.M{"$set": bson.M{"qrcodeimagestatus": 1}})
}

func addActivitys() {

	session, cols := Default()
	defer session.Close()

	exchangeCol := cols.AnswerExchange()
	exchanges := []models.AnswerExchange{}
	exchangeCol.Find(nil).All(&exchanges)

	activitysCol := cols.Activitys()
	activitysCol.RemoveAll(nil)

	for _, v := range exchanges {
		// f := models.Feed{
		// 	OpenID:    v.SourceUser,
		// 	Question:  v.Question,
		// 	Answer:    v.Source,
		// 	Type:      1,
		// 	CreatedAt: v.CreatedAt,
		// }
		ff := models.Feed{
			OpenID:       v.SourceUser,
			Question:     v.Question,
			Answer:       v.Swapper,
			AnswerOpenID: v.SwapperUser,
			Type:         1,
			CreatedAt:    v.CreatedAt,
		}
		// fs := models.Feed{
		// 	OpenID:    v.SwapperUser,
		// 	Question:  v.Question,
		// 	Answer:    v.Swapper,
		// 	Type:      1,
		// 	CreatedAt: v.CreatedAt,
		// }
		ffs := models.Feed{
			OpenID:       v.SwapperUser,
			Question:     v.Question,
			Answer:       v.Source,
			AnswerOpenID: v.SourceUser,
			Type:         1,
			CreatedAt:    v.CreatedAt,
		}
		activitysCol.Insert(ff, ffs)
	}

	answersCol := cols.Answers()
	answers := []models.Answer{}
	answersCol.Find(bson.M{"status": bson.M{"$ne": models.StatusDelete}}).All(&answers)

	for _, v := range answers {
		f := models.Feed{
			OpenID:       v.OpenID,
			Question:     v.QuestionID,
			Answer:       v.ID,
			AnswerOpenID: v.OpenID,
			Type:         0,
			CreatedAt:    v.CreatedAt,
		}
		activitysCol.Insert(f)
	}
}

func addGroupExchangedOpenID() {

	session, cols := Default()
	defer session.Close()

	groupCol := cols.Groups()
	groups := []models.Group{}
	groupCol.Find(nil).All(&groups)

	answerCol := cols.Answers()
	for _, v := range groups {
		answerIDs := make([]bson.ObjectId, len(v.Answers))
		for i, e := range v.Answers {
			answerIDs[i] = e.ID
		}
		answers := []models.Answer{}
		answerCol.Find(bson.M{"_id": bson.M{"$in": answerIDs}}).All(&answers)
		answerMap := map[bson.ObjectId]models.Answer{}
		for _, a := range answers {
			answerMap[a.ID] = a
		}

		rs := []models.GroupAnswer{}
		for _, v := range v.Answers {
			if a, ok := answerMap[v.ID]; ok {
				r := models.GroupAnswer{
					ID:        v.ID,
					OpenID:    a.OpenID,
					CreatedAt: v.CreatedAt,
				}
				rs = append(rs, r)
			}
		}
		groupCol.Update(bson.M{"_id": v.ID}, bson.M{"$set": bson.M{"answers": rs}})
	}
}

func addAnswerAllExchangedCount() {

	session, cols := Default()
	defer session.Close()

	col := cols.Answers()
	answers := []models.Answer{}
	col.Find(nil).All(&answers)

	for _, v := range answers {
		all := v.ExchangeCount + v.BeExchangedCount
		col.Update(bson.M{"_id": v.ID}, bson.M{"$set": bson.M{"allexchangecount": all}})
	}
}
