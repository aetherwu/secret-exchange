package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dagong.in/delphi-web-server/db"
	"dagong.in/delphi-web-server/middleware"
	"dagong.in/delphi-web-server/routers"
	"dagong.in/delphi-web-server/routers/activity"
	"dagong.in/delphi-web-server/routers/answer"
	"dagong.in/delphi-web-server/routers/badge"
	"dagong.in/delphi-web-server/routers/dashboard"
	"dagong.in/delphi-web-server/routers/group"
	"dagong.in/delphi-web-server/routers/invite"
	"dagong.in/delphi-web-server/routers/profile"
	"dagong.in/delphi-web-server/routers/question"
	"dagong.in/delphi-web-server/routers/user"
	"dagong.in/delphi-web-server/routers/wechat"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

func main() {

	log.SetFlags(log.Llongfile)

	router := initRouters()
	s := &http.Server{
		Addr:         ":5000",
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen: %s\n", err)
		}
	}()

	// shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	// 已有短连接在后台 保持 15 秒
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}

	log.Println("Server exiting")
}

func init() {
	log.Println("Pid: ", os.Getpid())
	// 连接数据库
	db.Connect()
}

// initRouters 初始化 routers
func initRouters() *gin.Engine {

	router := gin.New()
	// 日志 和 报错 中间件
	router.Use(middleware.Logger(), gin.Recovery())
	// 跨域 中间件
	router.Use(cors.Default())
	// Gzip 中间件
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// 首页
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 延迟返回 中间件
	// router.Use(middleware.Sleep(2000))

	// static files
	router.LoadHTMLGlob("./static/*.html")
	router.Use(static.Serve("/static", static.LocalFile("./static/static", false)))
	router.Use(static.Serve("/assets", static.LocalFile("./assets", false)))

	// 404 页面
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusOK, routers.ErrorResponse{ErrCode: 404, ErrMsg: "404"})
	})

	// 注册 登录
	router.POST("/auth/regist", routers.Regist)
	router.POST("/auth/register", routers.RegisterLocal)
	router.POST("/auth/login", routers.LoginLocal)

	// 后台
	poolRoute := router.Group("/pool")
	poolRoute.Use(middleware.BaseAuth())
	{
		poolRoute.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{})
		})
	}

	// 后台 接口
	dashboardRoute := router.Group("/dashboard")
	dashboardRoute.Use(middleware.BaseAuth())
	{
		dashboardRoute.POST("/count", dashboard.Count)

		dashboardRoute.POST("/users", dashboard.Users)
		dashboardRoute.POST("/users/detail", dashboard.UserDetail)

		dashboardRoute.POST("/activitys", dashboard.Activitys)

		dashboardRoute.POST("/questions", dashboard.Questions)
		dashboardRoute.POST("/questions/add", dashboard.AddQuestion)
		dashboardRoute.POST("/questions/remove", dashboard.DeleteQuestion)
		dashboardRoute.POST("/questions/edit", dashboard.EditQuestion)
		dashboardRoute.POST("/questions/detail", dashboard.QuestionDetail)
		dashboardRoute.POST("/questions/find", dashboard.FindQuestions)
		dashboardRoute.POST("/questions/edit/level", dashboard.EditQuestionLevel)

		dashboardRoute.POST("/answers", dashboard.Answers)
		dashboardRoute.POST("/answers/edit", dashboard.EditAnswer)
		dashboardRoute.POST("/answers/delete", dashboard.DeleteAnswer)

		dashboardRoute.POST("/group/topics", dashboard.GroupTopics)
		dashboardRoute.POST("/group/list", dashboard.GroupList)
	}

	// 问题
	questionRoute := router.Group("/question")
	// 强制认证 中间件
	questionRoute.Use(routers.AuthRequired())
	{
		questionRoute.POST("/detail", question.Detail)
		questionRoute.POST("/random", question.Random)
		questionRoute.POST("/detail-with-answer", question.DetailWithAnswer)
	}

	// 答案
	answerRoute := router.Group("/answer")
	{
		answerRoute.POST("/detail", answer.Detail)
		answerRoute.POST("/wechat/share-image", answer.WeChatShareImage)
	}

	// 强制认证 中间件
	answerRoute.Use(routers.AuthRequired())
	{
		answerRoute.POST("/create", answer.Create)
		answerRoute.POST("/exchange", answer.Exchange)
		answerRoute.POST("/exchange-answer", answer.ExchangeWithAnswer)
		answerRoute.POST("/comment", answer.Comment)
		answerRoute.POST("/like", answer.Like)
		answerRoute.POST("/privacy", answer.Privacy)
		answerRoute.POST("/privacy/modify", answer.ModifyPrivacy)
		answerRoute.POST("/qrcode", answer.GetQRCode)
		answerRoute.POST("/qrcode/update", answer.UpdateQRCode)
	}

	// 邀请
	inviteRoute := router.Group("/invite")
	{
		inviteRoute.POST("/detail", invite.Detail)
	}

	inviteRoute.Use(routers.AuthRequired())
	{
		inviteRoute.POST("/getid", invite.GetInvite)
		inviteRoute.POST("/gen-permit", invite.GenInvitePermit)
		inviteRoute.POST("/gen-permit-answer", invite.GenInvitePermitWithAnswer)
		inviteRoute.POST("/exchange", invite.ExchangeWithInvitePermit)
		inviteRoute.POST("/permit/list", invite.InvitePermitList)
		inviteRoute.POST("/detail/permit", invite.DetailWithInvitePermit)
		inviteRoute.POST("/permit/status", invite.InvitePermitStatus)
	}

	// 个人主页
	profileRoute := router.Group("/profile")
	profileRoute.Use(routers.AuthRequired())
	{
		profileRoute.POST("", profile.Profile)
		profileRoute.POST("/list", profile.RecentList)
		profileRoute.POST("/list/most", profile.ProfileListMost)
		profileRoute.POST("/list/follow", profile.Follows)
		profileRoute.POST("/reputation", profile.Reputation)
		profileRoute.POST("/shake", profile.Shake)
		profileRoute.POST("/shake/user-info", profile.ShakeUserInfo)
		profileRoute.POST("/qrcode", profile.GetQRCode)
		profileRoute.POST("/qrcode/update", profile.UpdateQRCode)
		profileRoute.POST("/random", profile.Random)
		profileRoute.POST("/follow", profile.Follow)
		profileRoute.POST("/my", profile.My)
		profileRoute.POST("/my/exchange", profile.MyExchange)
		profileRoute.POST("/my-exchange/most", profile.MyAnswerExchangeAtMost)
	}

	// 动态
	activityRoute := router.Group("/activity")
	activityRoute.Use(routers.AuthRequired())
	{
		activityRoute.POST("/list", activity.List)
	}

	// 模板消息
	notifyRouter := router.Group("/notify")
	notifyRouter.Use(routers.AuthRequired())
	{
		notifyRouter.POST("/form-id/add", wechat.AddFormID)
		notifyRouter.POST("/form-id/count", wechat.FormIDCount)
	}

	// 小红点
	badgeRouter := router.Group("/badge")
	badgeRouter.Use(routers.AuthRequired())
	{
		badgeRouter.POST("/invite/count", badge.InviteBadgeNumber)
		badgeRouter.POST("/invite/clean", badge.CleanInviteBadgeNumber)
	}

	// user 相关
	userRouter := router.Group("/user")
	{
		userRouter.GET("/wRAz1pYLmv.txt", func(c *gin.Context) {
			c.File("./static/wRAz1pYLmv.txt")
		})
	}

	userRouter.Use(routers.AuthRequired())
	{
		userRouter.POST("/userInfo", user.Profile)
		userRouter.POST("/update", user.UpdateUserInfo)
		userRouter.POST("/update-intro", user.UpdateUserIntro)
		userRouter.POST("/wechat-share-image", user.UpdateProfileWeChatShareImage)
	}

	// 群
	gorupRouter := router.Group("/group")
	gorupRouter.Use(routers.AuthRequired())
	{
		gorupRouter.POST("/detail", group.Detail)
		gorupRouter.POST("/exchange", group.Exchange)
		gorupRouter.POST("/exchange-answer", group.ExchangeWithAnswered)
		gorupRouter.POST("/rank/user", group.UserRank)
		gorupRouter.POST("/rank/question", group.QuestionRank)
	}

	// 微信相关
	wechatRouter := router.Group("/wechat")
	wechatRouter.Use(routers.AuthRequired())
	{
		wechatRouter.POST("/share-info/decrypt", wechat.DecryptShareInfo)
	}

	// 图片上传
	router.POST("/upload-token", routers.UploadToken)
	router.POST("/upload/callback", routers.UploadCallback)

	// router.GET("/test", routers.Test)
	return router
}
