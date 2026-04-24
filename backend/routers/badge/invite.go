package badge

import (
	"net/http"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/models"
	"dagong.in/delphi-web-server/routers"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
)

// InviteBadgeNumber 邀请的小红点数量
func InviteBadgeNumber(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	col := cols.InviteBadgeNumber()
	resp := models.InviteBadgeNumber{}
	col.Find(bson.M{"openid": auth.OpenID}).One(&resp)

	c.JSON(http.StatusOK, gin.H{"number": resp.Number})
}

// CleanInviteBadgeNumber 清除所有邀请的小红点
func CleanInviteBadgeNumber(c *gin.Context) {

	auth := routers.GetContextAuth(c)

	session, cols := db.Default()
	defer session.Close()

	col := cols.InviteBadgeNumber()
	col.Upsert(bson.M{"openid": auth.OpenID}, bson.M{"$set": bson.M{"number": 0}})

	c.JSON(http.StatusOK, routers.OK())
}
