package router

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"kefu-server/config"
	"kefu-server/controllers"
	"kefu-server/middleware"
	"kefu-server/utils/logger"
)

// SetupRouter 初始化并返回 Gin 路由引擎。
// 该函数集中注册中间件、静态资源、HTTP API 与 WebSocket 端点。
func SetupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimit())
	r.Use(middleware.UploadsSecurity())

	// 创建控制器实例
	userController := &controllers.UserController{}
	appController := &controllers.AppController{}
	visitorController := &controllers.VisitorController{}
	agentController := &controllers.AgentController{}
	sessionController := &controllers.SessionController{}
	quickReplyController := &controllers.QuickReplyController{}
	knowledgeController := &controllers.KnowledgeController{}
	knowledgeWorkspaceController := &controllers.KnowledgeWorkspaceController{}
	apiModelConfigController := &controllers.APIModelConfigController{}
	faqController := &controllers.FAQController{}
	appAPIKeyController := &controllers.AppAPIKeyController{}
	agentSettingController := &controllers.AgentSettingController{}
	aiSuggestController := &controllers.AISuggestController{}
	auditController := &controllers.AuditController{}
	healthController := &controllers.HealthController{}
	staticController := &controllers.StaticController{}
	adminPanelController := &controllers.AdminPanelController{}
	uploadController := &controllers.UploadController{}
	wecomController := &controllers.WecomController{}
	notificationController := controllers.NewNotificationController()

	r.GET("/healthz", healthController.Health)
	staticController.Register(r)
	if cfg := config.GetConfig(); cfg != nil {
		uploadDir := strings.TrimSpace(cfg.Admin.UploadDir)
		if uploadDir != "" {
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				logger.Errorf("upload dir mkdir failed path=%s err=%v", uploadDir, err)
			}
			r.Static("/uploads", uploadDir)
		}
	}

	// API 路由组
	api := r.Group("/api/v1")
	{
		// 不需要认证的路由
		api.POST("/login", userController.Login)
		api.GET("/captcha", userController.GetCaptcha)
		api.GET("/config", appController.GetConfig)
		api.GET("/visitor/history", visitorController.History)
		api.POST("/upload", uploadController.CreatePublic)
		api.POST("/sessions/rate", middleware.RateLimitMiddleware(1, 10), sessionController.Rate)

		// 需要认证的路由
		auth := api.Group("/")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.POST("/logout", userController.Logout)
			auth.GET("/user/info", userController.GetUserInfo)
			auth.POST("/user/status", userController.SetUserStatus)
			auth.GET("/user/status", userController.GetUserStatus)

			session := auth.Group("/sessions")
			{
				session.GET("/list", sessionController.List)
				session.GET("/messages", sessionController.GetMessages)
				session.GET("/agents", sessionController.ListAgents)
				session.POST("/accept", middleware.RequireRoles("agent", "admin"), sessionController.Accept)
				session.POST("/transfer", middleware.RequireRoles("agent", "admin"), sessionController.Transfer)
				session.POST("/close", middleware.RequireRoles("agent", "admin"), sessionController.Close)
				session.POST("/read", middleware.RequireRoles("agent", "admin"), sessionController.MarkRead)
				session.POST("/follow-up", middleware.RequireRoles("agent", "admin"), sessionController.MarkFollowUp)
			}

			quickReplies := auth.Group("/quick-replies")
			quickReplies.Use(middleware.RequireRoles("agent", "admin"))
			{
				quickReplies.GET("/list", quickReplyController.List)
				quickReplies.POST("/create", quickReplyController.Create)
				quickReplies.PUT("/update", quickReplyController.Update)
				quickReplies.DELETE("/delete", quickReplyController.Delete)
				quickReplies.POST("/use", quickReplyController.Use)
			}

			// 用户管理路由（仅管理员）
			user := auth.Group("/user")
			user.Use(middleware.RequireRoles("admin"))
			{
				user.GET("/list", userController.ListUsers)
				user.POST("/create", userController.CreateUser)
				user.PUT("/update", userController.UpdateUser)
				user.POST("/batch-active", userController.BatchActive)
				user.DELETE("/delete", userController.DeleteUser)
			}

			// App 管理路由
			app := auth.Group("/apps")
			{
				app.GET("/list", middleware.RequireRoles("agent", "admin"), appController.GetApps)
				app.Use(middleware.RequireRoles("admin"))
				app.POST("/create", appController.CreateApp)
				app.PUT("/update", appController.UpdateApp)
				app.POST("/delete", appController.DeleteApp)
				app.DELETE("/delete", appController.DeleteApp)
			}

			audit := auth.Group("/audit")
			audit.Use(middleware.RequireRoles("admin"))
			{
				audit.GET("/list", auditController.List)
			}

			panel := auth.Group("/panel")
			{
				panel.GET("/dashboard", middleware.RequireRoles("admin"), adminPanelController.Dashboard)
				panel.GET("/visitors", middleware.RequireRoles("admin"), adminPanelController.Visitors)
				panel.GET("/user-stats", middleware.RequireRoles("admin"), adminPanelController.UserStats)
				panel.GET("/export/sessions", middleware.RequireRoles("admin"), adminPanelController.ExportSessions)
				panel.GET("/settings", middleware.RequireRoles("agent", "admin"), adminPanelController.GetSystemSettings)
				panel.PUT("/settings", middleware.RequireRoles("agent", "admin"), adminPanelController.UpdateSystemSettings)
				panel.GET("/profile-summary", middleware.RequireRoles("agent", "admin"), adminPanelController.ProfileSummary)
			}

			auth.PUT("/user/profile", userController.UpdateProfile)
			auth.POST("/user/password", userController.ChangePassword)
			auth.POST("/upload-auth", uploadController.Create)
			auth.GET("/export/sessions", middleware.RequireRoles("admin"), adminPanelController.ExportSessions)
			auth.GET("/agent/settings", middleware.RequireRoles("agent", "admin"), agentSettingController.Get)
			auth.PUT("/agent/settings", middleware.RequireRoles("agent", "admin"), agentSettingController.Update)
			auth.POST("/ai/suggest", middleware.RequireRoles("agent", "admin"), aiSuggestController.Suggest)
			auth.POST("/ai/bot-test", middleware.RequireRoles("agent", "admin"), visitorController.AIBotTest)

			knowledge := auth.Group("/knowledge")
			{
				knowledge.GET("/list", middleware.RequireRoles("agent", "admin"), knowledgeController.List)
				knowledge.POST("/create", middleware.RequireRoles("agent", "admin"), knowledgeController.Create)
				knowledge.PUT("/update", middleware.RequireRoles("agent", "admin"), knowledgeController.Update)
				knowledge.DELETE("/delete", middleware.RequireRoles("agent", "admin"), knowledgeController.Delete)
				knowledge.POST("/upload", middleware.RequireRoles("agent", "admin"), knowledgeController.UploadByURL)
				knowledge.POST("/rag-test", middleware.RequireRoles("agent", "admin"), knowledgeController.RAGTest)
			}

			knowledgeBases := auth.Group("/knowledge-bases")
			knowledgeBases.Use(middleware.RequireRoles("agent", "admin"))
			{
				knowledgeBases.GET("/list", knowledgeWorkspaceController.ListBases)
				knowledgeBases.POST("/create", knowledgeWorkspaceController.CreateBase)
				knowledgeBases.PUT("/update", knowledgeWorkspaceController.UpdateBase)
				knowledgeBases.DELETE("/delete", knowledgeWorkspaceController.DeleteBase)
				knowledgeBases.GET("/healthz", knowledgeWorkspaceController.ValidateConnectivity)
				knowledgeBases.GET("/:id/documents", knowledgeWorkspaceController.ListDocuments)
				knowledgeBases.POST("/:id/documents/upload", knowledgeWorkspaceController.UploadDocument)
				knowledgeBases.POST("/:id/documents/reindex", knowledgeWorkspaceController.ReindexDocument)
				knowledgeBases.DELETE("/:id/documents/:docID", knowledgeWorkspaceController.DeleteDocument)
				knowledgeBases.GET("/:id/chunks", knowledgeWorkspaceController.ListChunks)
				knowledgeBases.PUT("/:id/chunks/:chunkID", knowledgeWorkspaceController.UpdateChunk)
				knowledgeBases.DELETE("/:id/chunks/:chunkID", knowledgeWorkspaceController.DeleteChunk)
				knowledgeBases.POST("/:id/retrieve-test", knowledgeWorkspaceController.RetrieveTest)
				knowledgeBases.POST("/:id/qa-test", knowledgeWorkspaceController.QATest)
				knowledgeBases.POST("/:id/feedback", knowledgeWorkspaceController.SaveFeedback)
			}

			apiModels := auth.Group("/admin/api-models")
			apiModels.Use(middleware.RequireRoles("agent", "admin"))
			{
				apiModels.GET("", apiModelConfigController.List)
				apiModels.GET("/:id", apiModelConfigController.Get)
				apiModels.POST("", apiModelConfigController.Create)
				apiModels.PUT("/:id", apiModelConfigController.Update)
				apiModels.DELETE("/:id", apiModelConfigController.Delete)
				apiModels.POST("/:id/test", apiModelConfigController.Test)
				apiModels.POST("/:id/set-default", apiModelConfigController.SetDefault)
				apiModels.POST("/:id/rebuild", apiModelConfigController.TriggerRebuild)
				apiModels.GET("/:id/rebuild", apiModelConfigController.GetRebuildStatus)
			}

			faq := auth.Group("/faq")
			{
				faq.GET("/list", middleware.RequireRoles("agent", "admin"), faqController.List)
				faq.POST("/create", middleware.RequireRoles("agent", "admin"), faqController.Create)
				faq.PUT("/update", middleware.RequireRoles("agent", "admin"), faqController.Update)
				faq.DELETE("/delete", middleware.RequireRoles("agent", "admin"), faqController.Delete)
			}

			apiKey := auth.Group("/app-api-keys")
			apiKey.Use(middleware.RequireRoles("agent", "admin"))
			{
				apiKey.GET("/list", appAPIKeyController.List)
				apiKey.POST("/create", appAPIKeyController.Create)
				apiKey.POST("/rotate", appAPIKeyController.Rotate)
				apiKey.POST("/set-enabled", appAPIKeyController.SetEnabled)
				apiKey.DELETE("/delete", appAPIKeyController.Delete)
			}

			wecomAdmin := auth.Group("/admin/wecom")
			wecomAdmin.Use(middleware.RequireRoles("admin"))
			{
				wecomAdmin.GET("/config", wecomController.GetConfig)
				wecomAdmin.POST("/config", wecomController.SaveConfig)
				wecomAdmin.POST("/test", wecomController.TestConnection)
			}

			wecomAgent := auth.Group("/agent/wecom")
			wecomAgent.Use(middleware.RequireRoles("agent", "admin"))
			{
				wecomAgent.GET("/qrcode", wecomController.GetQrcode)
				wecomAgent.GET("/bind-status", wecomController.GetBindStatus)
				wecomAgent.GET("/bind-info", wecomController.GetBindInfo)
				wecomAgent.POST("/unbind", wecomController.Unbind)
			}

			notificationAdmin := auth.Group("/admin/notification")
			notificationAdmin.Use(middleware.RequireRoles("admin"))
			{
				notificationAdmin.GET("/channels", notificationController.GetChannels)
				notificationAdmin.GET("/channels/:channel", notificationController.GetChannelConfig)
				notificationAdmin.POST("/channels/:channel", notificationController.SaveChannelConfig)
				notificationAdmin.POST("/channels/:channel/test", notificationController.TestChannel)
			}

			notificationAgent := auth.Group("/agent/notification")
			notificationAgent.Use(middleware.RequireRoles("agent", "admin"))
			{
				notificationAgent.GET("/channels/:channel/qrcode", notificationController.GetBindQrcode)
				notificationAgent.GET("/channels/:channel/bind-status", notificationController.GetBindStatus)
				notificationAgent.GET("/channels/:channel/bind-info", notificationController.GetBindInfo)
				notificationAgent.POST("/channels/:channel/unbind", notificationController.Unbind)
			}
		}
	}

	r.GET("/api/wecom/callback", wecomController.Callback)
	r.GET("/api/notification/callback/:channel", notificationController.HandleCallback)

	// WebSocket 路由
	r.GET("/ws/chat", visitorController.WSHandler)
	r.GET("/ws/agent", middleware.AuthMiddleware(), agentController.WSHandler)

	return r
}
