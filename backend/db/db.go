package db

import (
	"log"
	"time"

	"dagong.in/delphi-web-server/config"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
)

var session *mgo.Session
var databaseName string

func getDatabaseName() string {
	n := "test"
	if gin.Mode() == gin.ReleaseMode {
		n = "mxdb"
	}
	databaseName = n
	return n
}

func dialInfo() *mgo.DialInfo {
	n := "test"
	var info *mgo.DialInfo
	switch gin.Mode() {
	case gin.DebugMode:
		info = &mgo.DialInfo{
			Addrs: []string{config.DatabaseDebugURI},
		}
	case gin.TestMode:
		info = &mgo.DialInfo{
			Addrs:    []string{config.DatabaseTestURI},
			Database: n,
			Username: config.DatabaseUser,
			Password: config.DatabasePassword,
		}
	case gin.ReleaseMode:
		n = "mxdb"
		info = &mgo.DialInfo{
			Addrs:    []string{config.DatabaseReleaseURI},
			Database: n,
			Username: config.DatabaseUser,
			Password: config.DatabasePassword,
		}
	}
	databaseName = n
	return info
}

// Connect 连接数据库
func Connect() {
	info := dialInfo()
	info.Timeout = time.Second * 30
	s, err := mgo.DialWithInfo(info)
	if err != nil {
		log.Println("Can't connect to mongo:", info.Addrs[0], info.Database, info.Username)
		panic(err.Error())
	}
	s.SetSafe(&mgo.Safe{})
	log.Println("Connected to", info.Addrs[0])
	session = s

	if err := ensureIndex(); err != nil {
		log.Println("EnsureIndex error:", err)
	}

	upgrade()
}

// upgrade 更新数据库
func upgrade() {
	session, cols := Default()
	defer session.Close()

	versionCol := cols.Version()

	version := version{}
	versionCol.Find(nil).One(&version)

	for {
		switch version.V {
		case 0:

		case 1:
			addAnswerExchangeAtUser()
		case 2:
			addAnswerTotalExchangeCount()
		case 3:
			addQuestionAnswerCount()
		case 4:
			addGroupQuestionID()
		case 5:
			addAnswerPrivacy()
		case 6:

		case 7:
			resetAnswerQRCode()
		case 8:
			addActivitys()
		case 9:
			addGroupExchangedOpenID()
		case 10:
			addAnswerAllExchangedCount()
		default:
			return
		}
		version.V++
		versionCol.RemoveAll(nil)
		versionCol.Insert(version)
	}

}

// CopySession copy 连接
func CopySession() *mgo.Session {
	return session.Copy()
}

// Database 根据 数据库名称获取数据库连接
func Database(name string) (*mgo.Session, Collections) {
	s := CopySession()
	return s, Collections{db: s.DB(name)}
}

// Default 获取默认数据库连接
func Default() (*mgo.Session, Collections) {
	return Database(databaseName)
}

// ensureIndex 创建 索引
func ensureIndex() error {
	s, i := Default()
	defer s.Close()

	if err := ensureIndexWithWeChat(i); err != nil {
		return err
	}

	if err := ensureIndexWithUser(i); err != nil {
		return err
	}

	if err := ensureIndexWithFormID(i); err != nil {
		return err
	}

	// if err := ensureIndexWithQuestionLimit(i); err != nil {
	// 	return err
	// }
	return nil
}

func ensureIndexWithWeChat(cols Collections) error {
	col := cols.WeChat()

	sessionTTL := mgo.Index{
		Key:         []string{"createdat"},
		Unique:      false,
		DropDups:    false,
		Background:  true,
		ExpireAfter: 7200*time.Second - 5*time.Minute,
	}

	if err := col.EnsureIndex(sessionTTL); err != nil {
		return err
	}
	return nil
}

func ensureIndexWithFormID(cols Collections) error {
	col := cols.FormIDs()
	sessionTTL := mgo.Index{
		Key:         []string{"createdat"},
		Unique:      false,
		DropDups:    false,
		Background:  true,
		ExpireAfter: 7*24*time.Hour - 1*time.Hour,
	}

	if err := col.EnsureIndex(sessionTTL); err != nil {
		return err
	}
	return nil
}

func ensureIndexWithUser(cols Collections) error {
	col := cols.Users()

	index := mgo.Index{
		Key:        []string{"openid"},
		Unique:     true,
		DropDups:   false,
		Background: true,
	}

	if err := col.EnsureIndex(index); err != nil {
		return err
	}

	emailIndex := mgo.Index{
		Key:        []string{"email"},
		Unique:     true,
		Sparse:     true,
		DropDups:   false,
		Background: true,
	}
	if err := col.EnsureIndex(emailIndex); err != nil {
		return err
	}

	return nil
}

func ensureIndexWithQuestionLimit(cols Collections) error {
	col := cols.QuestionLimit()
	sessionTTL := mgo.Index{
		Key:         []string{"expiredat"},
		Unique:      false,
		DropDups:    false,
		Background:  true,
		ExpireAfter: 0 * time.Second,
	}

	if err := col.EnsureIndex(sessionTTL); err != nil {
		return err
	}
	return nil
}
