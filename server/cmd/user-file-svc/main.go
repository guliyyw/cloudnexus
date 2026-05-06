package main

import (
	"log"
	"os"
	"time"

	"github.com/cloudnexus/server/internal/userfile/handler"
	"github.com/cloudnexus/server/internal/userfile/repository"
	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/config"
	"github.com/cloudnexus/server/pkg/database"
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
	if err := db.AutoMigrate(&model.User{}, &model.RefreshToken{}, &model.File{}, &model.FileShare{}); err != nil {
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
	userH := handler.NewUserHandler(userSvc)

	fileRepo := repository.NewFileRepository(db)
	fileSvc := service.NewFileService(fileRepo, minioClient, cfg.MinIO.Bucket)
	fileH := handler.NewFileHandler(fileSvc)
	systemH := handler.NewSystemHandler(db, minioClient)
	go systemH.StartMetricsCollector()

	nodeReg := system.NewNodeRegistrar(db, os.Getenv("NODE_NAME"), os.Getenv("NODE_HOST"), 8081)
	nodeReg.Start()
	defer nodeReg.Stop()

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

			protected := user.Group("")
			protected.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
			{
				protected.GET("/profile", userH.HandleGetProfile)
				protected.PUT("/profile", userH.HandleUpdateProfile)
				protected.PUT("/password", userH.HandleChangePassword)
			}
		}

		file := api.Group("/file")
		file.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
		{
			file.POST("/upload", fileH.HandleUpload)
			file.GET("/list", fileH.HandleList)
			file.GET("/download/:id", fileH.HandleDownload)
			file.DELETE("/:id", fileH.HandleDelete)
			file.POST("/mkdir", fileH.HandleMkdir)
			file.GET("/search", fileH.HandleSearch)
			file.POST("/batch-delete", fileH.HandleBatchDelete)
			file.POST("/batch-download", fileH.HandleBatchDownload)
			file.POST("/move", fileH.HandleMove)
			file.POST("/copy", fileH.HandleCopy)
			file.POST("/:id/share", shareH.HandleCreateShare)
			file.GET("/:id/shares", shareH.HandleListSharesByFile)
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
