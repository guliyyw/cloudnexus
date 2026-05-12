package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudnexus/server/internal/userfile/handler"
	"github.com/cloudnexus/server/internal/userfile/repository"
	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/cache"
	"github.com/cloudnexus/server/pkg/captcha"
	"github.com/cloudnexus/server/pkg/config"
	"github.com/cloudnexus/server/pkg/database"
	"github.com/cloudnexus/server/pkg/email"
	"github.com/cloudnexus/server/pkg/logger"
	"github.com/cloudnexus/server/pkg/middleware"
	"github.com/cloudnexus/server/pkg/migration"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/snowflake"
	"github.com/cloudnexus/server/pkg/storage"
	"github.com/cloudnexus/server/pkg/system"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config/config.single.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if err := logger.Init(logger.Config{
		Level:   cfg.Log.Level,
		Format:  cfg.Log.Format,
		Service: "user-file-svc",
		LogDir:  "/app/logs",
	}); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()
	logger.StartLogCleanup()

	snowflake.Init(1)

	db, err := database.NewPostgres(database.Config{DSN: cfg.Database.DSN})
	if err != nil {
		logger.Log.Fatal("连接数据库失败", zap.Error(err))
	}

	if err := migration.Up(db); err != nil {
		logger.Log.Warn("SQL migration skipped", zap.Error(err))
	}
	rdb, err := cache.NewRedis(cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.Log.Warn("连接 Redis 失败，验证码不可用", zap.Error(err))
	}

	if err := db.AutoMigrate(&model.User{}, &model.RefreshToken{}, &model.File{}, &model.FileShare{}, &model.FileVersion{}, &model.DockerNode{}, &model.AlertRule{}, &model.AlertHistory{}, &model.EmailVerification{}, &model.PhoneVerification{}, &model.PasswordResetToken{}, &model.UserSession{}, &model.OAuthBinding{}, &model.Permission{}, &model.Role{}, &model.RolePermission{}, &model.UserRole{}); err != nil {
		logger.Log.Fatal("数据库AutoMigrate失败", zap.Error(err))
	}

	minioClient, err := storage.NewMinIO(storage.Config{
		Endpoint:  cfg.MinIO.Endpoint,
		AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey,
		UseSSL:    cfg.MinIO.UseSSL,
		Bucket:    cfg.MinIO.Bucket,
	})
	if err != nil {
		logger.Log.Fatal("连接 MinIO 失败", zap.Error(err))
	}

	jwtCfg := auth.Config{
		AccessSecret:  cfg.JWT.AccessSecret,
		RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL:     time.Duration(cfg.JWT.AccessTTL) * time.Second,
		RefreshTTL:    time.Duration(cfg.JWT.RefreshTTL) * time.Second,
	}

	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo, jwtCfg)
	userSvc.SeedDefaultAdmin()
	userH := handler.NewUserHandler(userSvc)
	var captchaMgr *captcha.Manager
	if rdb != nil {
		captchaMgr = captcha.NewManager(rdb)
	}

	emailSender := email.NewSender(*cfg)
	verifySvc := service.NewVerifyService(db, emailSender)
	verifyH := handler.NewVerifyHandler(verifySvc)
	resetSvc := service.NewResetService(db, emailSender)
	resetH := handler.NewResetHandler(resetSvc)
	sessionSvc := service.NewSessionService(db, rdb, *cfg)
	sessionSvc.CleanExpiredSessions()
	sessionH := handler.NewSessionHandler(sessionSvc)
	userH.WithSessionService(sessionSvc)
	roleRepo := repository.NewRoleRepository(db)
	roleSvc := service.NewRoleService(roleRepo)
	if err := roleSvc.SeedRBAC(); err != nil {
		logger.Log.Warn("RBAC 种子数据写入失败", zap.Error(err))
	}
	if err := service.AssignDefaultRoleToAllUsers(db); err != nil {
		logger.Log.Warn("默认角色分配失败", zap.Error(err))
	}
	userSvc.WithRoleRepo(roleRepo)
	roleH := handler.NewRoleHandler(roleSvc)
	deleteSvc := service.NewDeleteService(db)
	deleteH := handler.NewDeleteHandler(deleteSvc)
	oauthSvc := service.NewOAuthService(db)
	oauthH := handler.NewOAuthHandler(oauthSvc)
	searchH := handler.NewSearchHandler(userRepo)

	fileRepo := repository.NewFileRepository(db)
	fileSvc := service.NewFileService(fileRepo, minioClient, cfg.MinIO.Bucket)
	fileH := handler.NewFileHandler(fileSvc)
	systemH := handler.NewSystemHandler(db, minioClient)
	go systemH.StartMetricsCollector()

	nodeH := handler.NewNodeHandler(db)
	alertH := handler.NewAlertHandler(db)

	nodeReg := system.NewNodeRegistrar(db, os.Getenv("NODE_NAME"), os.Getenv("NODE_HOST"), "user-file-svc", 8081)
	nodeReg.Start()
	defer nodeReg.Stop()

	alerter := system.NewAlertEvaluator(db)
	aggregator := system.NewHealthAggregator(db, alerter)
	aggregator.RegisterInfra(system.InfraNode{
		Name: "postgres", Host: cfg.DBHost(), Port: 5432,
		ProbeFn: system.TCPProbe(cfg.DBHost(), 5432),
	})
	aggregator.RegisterInfra(system.InfraNode{
		Name: "redis", Host: cfg.RedisHost(), Port: 6379,
		ProbeFn: system.TCPProbe(cfg.RedisHost(), 6379),
	})
	aggregator.RegisterInfra(system.InfraNode{
		Name: "minio", Host: cfg.MinIOHost(), Port: 9000,
		ProbeFn: system.HTTPProbe(fmt.Sprintf("http://%s:9000/minio/health/live", cfg.MinIOHost())),
	})
	aggregator.Start()
	defer aggregator.Stop()

	shareRepo := repository.NewShareRepository(db)
	shareSvc := service.NewShareService(shareRepo, fileRepo)
	shareH := handler.NewShareHandler(shareSvc, fileSvc)

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	{
		user := api.Group("/user")
		{
			user.POST("/register", userH.HandleRegister)
			user.POST("/login", userH.HandleLogin)
			user.POST("/refresh", userH.HandleRefresh)

			if captchaMgr != nil {
				captchaH := handler.NewCaptchaHandler(captchaMgr)
				user.GET("/captcha", captchaH.HandleGenerate)
				user.POST("/captcha/verify", captchaH.HandleVerify)
			}

			user.POST("/email/send-code", verifyH.HandleSendEmailCode)
			user.POST("/email/verify", verifyH.HandleVerifyEmail)
			user.POST("/phone/send-code", verifyH.HandleSendPhoneCode)
			user.POST("/phone/verify", verifyH.HandleVerifyPhone)
			user.POST("/password/forgot", resetH.HandleForgotPassword)
			user.POST("/password/reset", resetH.HandleResetPassword)

			protected := user.Group("")
			protected.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
			{
				protected.GET("/profile", userH.HandleGetProfile)
				protected.PUT("/profile", userH.HandleUpdateProfile)
				protected.PUT("/password", userH.HandleChangePassword)
				protected.GET("/sessions", sessionH.HandleListSessions)
				protected.DELETE("/sessions/:jti", sessionH.HandleRevokeSession)
				protected.DELETE("/sessions", sessionH.HandleRevokeAllSessions)
				protected.POST("/delete/request", deleteH.HandleRequestDelete)
				protected.POST("/delete/cancel", deleteH.HandleCancelDelete)
				protected.GET("/oauth/bindings", oauthH.HandleListBindings)
				protected.DELETE("/oauth/unbind", oauthH.HandleUnbind)
				protected.GET("/search", searchH.HandleSearch)
				protected.GET("/privacy", userH.HandleGetPrivacy)
				protected.PUT("/privacy", userH.HandleUpdatePrivacy)
			}
		}

		file := api.Group("/file")
		file.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
		{
			file.POST("/upload", middleware.RequirePermission("file:write"), fileH.HandleUpload)
			file.GET("/list", middleware.RequirePermission("file:read"), fileH.HandleList)
			file.GET("/download/:id", middleware.RequirePermission("file:read"), fileH.HandleDownload)
			file.DELETE("/:id", middleware.RequirePermission("file:delete"), fileH.HandleDelete)
			file.POST("/mkdir", middleware.RequirePermission("file:write"), fileH.HandleMkdir)
			file.GET("/search", middleware.RequirePermission("file:read"), fileH.HandleSearch)
			file.POST("/batch-delete", middleware.RequirePermission("file:delete"), fileH.HandleBatchDelete)
			file.POST("/batch-download", middleware.RequirePermission("file:read"), fileH.HandleBatchDownload)
			file.POST("/move", middleware.RequirePermission("file:write"), fileH.HandleMove)
			file.POST("/copy", middleware.RequirePermission("file:write"), fileH.HandleCopy)
			file.POST("/collab", middleware.RequirePermission("file:write"), fileH.HandleCreateCollab)
			// 文件版本
			file.GET("/:id/versions", middleware.RequirePermission("file:read"), fileH.HandleListVersions)
			file.POST("/:id/versions/:versionId/restore", middleware.RequirePermission("file:write"), fileH.HandleRestoreVersion)
			file.GET("/:id/versions/:versionId/download", middleware.RequirePermission("file:read"), fileH.HandleDownloadVersion)
			file.GET("/:id/meta", middleware.RequirePermission("file:read"), fileH.HandleGetFileMeta)
			file.POST("/:id/share", middleware.RequirePermission("file:share"), shareH.HandleCreateShare)
			file.GET("/:id/shares", middleware.RequirePermission("file:read"), shareH.HandleListSharesByFile)
		}

		shares := api.Group("/shares")
		shares.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
		{
			shares.GET("/my", shareH.HandleListMyShares)
			shares.DELETE("/:id", shareH.HandleDeleteShare)
		}

		share := api.Group("/share")
		{
			share.GET("/:code", shareH.HandleGetShareByCode)
			share.POST("/:code/verify", shareH.HandleVerifyPassword)
			share.GET("/:code/download", shareH.HandleDownloadShare)
		}

		api.GET("/metrics", systemH.HandleMetrics)

		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/users", userH.HandleAdminListUsers)
			admin.PUT("/users/:id/toggle-admin", userH.HandleAdminToggleAdmin)
			admin.PUT("/users/:id/toggle-status", userH.HandleAdminToggleStatus)
			admin.GET("/logs", systemH.HandleLogs)
			admin.GET("/logs/services", systemH.HandleLogServices)
			admin.GET("/logs/files", systemH.HandleLogFiles)
			admin.GET("/logs/download", systemH.HandleLogDownload)
			admin.GET("/metrics/resources", systemH.HandleResourceMetrics)
			admin.GET("/metrics/history", systemH.HandleMetricsHistory)
			admin.GET("/nodes", nodeH.HandleListNodes)
			admin.GET("/nodes/:name", nodeH.HandleGetNode)
			admin.GET("/nodes/:name/sessions", nodeH.HandleGetNodeSessions)
			admin.POST("/nodes", nodeH.HandleAddNode)
			admin.DELETE("/nodes/:name", nodeH.HandleDeleteNode)

				alerts := admin.Group("/alerts")
				{
					alerts.GET("/rules", alertH.HandleListRules)
					alerts.POST("/rules", alertH.HandleCreateRule)
					alerts.PUT("/rules/:id", alertH.HandleUpdateRule)
					alerts.DELETE("/rules/:id", alertH.HandleDeleteRule)
					alerts.GET("/history", alertH.HandleListHistory)
				}

				roles := admin.Group("/roles")
				{
					roles.GET("", roleH.HandleListRoles)
					roles.POST("", roleH.HandleCreateRole)
					roles.PUT("/:id", roleH.HandleUpdateRole)
					roles.DELETE("/:id", roleH.HandleDeleteRole)
					roles.GET("/permissions", roleH.HandleListPermissions)
					roles.POST("/:id/permissions", roleH.HandleAssignPermissions)
				}

				admin.GET("/users/:id/roles", roleH.HandleGetUserRoles)
				admin.POST("/users/:id/roles", roleH.HandleAssignUserRole)
				admin.DELETE("/users/:id/roles/:roleId", roleH.HandleRemoveUserRole)
			}
	}

	r.GET("/healthz", systemH.HandleHealthz)
	r.GET("/metrics", systemH.HandleMetrics)
	r.GET("/metrics/resources", systemH.HandleResourceMetrics)

	logger.Log.Info("user-file-svc starting", zap.Int("port", cfg.Server.Port))
	if err := r.Run(":8081"); err != nil {
		logger.Log.Fatal("启动失败", zap.Error(err))
	}
}
