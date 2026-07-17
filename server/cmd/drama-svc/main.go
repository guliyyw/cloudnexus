package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/cloudnexus/server/internal/drama/handler"
	"github.com/cloudnexus/server/internal/drama/repository"
	"github.com/cloudnexus/server/internal/drama/service"
	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/cache"
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
		log.Fatalf("load config failed: %v", err)
	}

	if err := logger.Init(logger.Config{
		Level:   cfg.Log.Level,
		Format:  cfg.Log.Format,
		Service: "drama-svc",
		LogDir:  "/app/logs",
	}); err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	defer logger.Sync()
	logger.StartLogCleanup()

	snowflake.Init(7)

	db, err := database.NewPostgres(database.Config{DSN: cfg.Database.DSN})
	if err != nil {
		logger.Log.Fatal("connect database failed", zap.Error(err))
	}
	if err := migration.Up(db); err != nil {
		logger.Log.Warn("SQL migration skipped", zap.Error(err))
	}
	if err := db.AutoMigrate(
		&model.File{},
		&model.DramaProject{},
		&model.DramaStoryboard{},
		&model.DramaStoryboardMedia{},
		&model.DramaAsset{},
		&model.DramaTask{},
		&model.DramaSetting{},
	); err != nil {
		logger.Log.Fatal("database automigrate failed", zap.Error(err))
	}

	minioClient, err := storage.NewMinIO(storage.Config{
		Endpoint:  cfg.MinIO.Endpoint,
		AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey,
		UseSSL:    cfg.MinIO.UseSSL,
		Bucket:    cfg.MinIO.Bucket,
	})
	if err != nil {
		logger.Log.Fatal("connect minio failed", zap.Error(err))
	}
	redisClient, err := cache.NewRedis(cache.Config{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	if err != nil {
		logger.Log.Fatal("connect redis failed", zap.Error(err))
	}
	defer redisClient.Close()

	nodeReg := system.NewNodeRegistrar(db, os.Getenv("NODE_NAME"), os.Getenv("NODE_HOST"), "drama-svc", 8087)
	nodeReg.Start()
	defer nodeReg.Stop()

	jwtCfg := auth.Config{
		AccessSecret:  cfg.JWT.AccessSecret,
		RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL:     time.Duration(cfg.JWT.AccessTTL) * time.Second,
		RefreshTTL:    time.Duration(cfg.JWT.RefreshTTL) * time.Second,
	}

	dramaRepo := repository.NewDramaRepository(db)
	dramaSvc := service.NewDramaService(dramaRepo, minioClient, cfg.MinIO.Bucket)
	taskRunner := service.NewTaskRunner(dramaSvc, dramaRepo, redisClient)
	dramaSvc.SetTaskRunner(taskRunner)
	if err := taskRunner.Start(context.Background()); err != nil {
		logger.Log.Fatal("start drama task runner failed", zap.Error(err))
	}
	dramaH := handler.NewDramaHandler(dramaSvc)

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	api.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
	{
		drama := api.Group("/drama")
		{
			drama.GET("/projects", middleware.RequirePermission("drama:read"), dramaH.HandleListProjects)
			drama.POST("/projects", middleware.RequirePermission("drama:write"), dramaH.HandleCreateProject)
			drama.POST("/projects/import", middleware.RequirePermission("drama:write"), dramaH.HandleImportProject)
			drama.GET("/projects/:id", middleware.RequirePermission("drama:read"), dramaH.HandleGetProject)
			drama.PUT("/projects/:id", middleware.RequirePermission("drama:write"), dramaH.HandleUpdateProject)
			drama.DELETE("/projects/:id", middleware.RequirePermission("drama:write"), dramaH.HandleDeleteProject)
			drama.POST("/projects/:id/parse", middleware.RequirePermission("drama:write"), dramaH.HandleParseScript)
			drama.POST("/projects/:id/append", middleware.RequirePermission("drama:write"), dramaH.HandleAppendStoryboards)
			drama.GET("/projects/:id/export", middleware.RequirePermission("drama:read"), dramaH.HandleExportProject)
			drama.GET("/projects/:id/tasks", middleware.RequirePermission("drama:read"), dramaH.HandleListTasks)
			drama.POST("/projects/:id/tasks", middleware.RequirePermission("drama:generate"), dramaH.HandleCreateTask)
			drama.POST("/projects/:id/tasks/:taskId/cancel", middleware.RequirePermission("drama:generate"), dramaH.HandleCancelTask)
			drama.POST("/projects/:id/tasks/:taskId/retry", middleware.RequirePermission("drama:generate"), dramaH.HandleRetryTask)
			drama.PUT("/projects/:id/storyboards/:storyboardId", middleware.RequirePermission("drama:write"), dramaH.HandleUpdateStoryboard)
			drama.PUT("/projects/:id/storyboards/:storyboardId/media/:mediaId/select", middleware.RequirePermission("drama:write"), dramaH.HandleSelectStoryboardMedia)
			drama.POST("/projects/:id/storyboards/:storyboardId/audio", middleware.RequirePermission("drama:write"), dramaH.HandleUploadStoryboardAudio)
			drama.POST("/projects/:id/audio/import", middleware.RequirePermission("drama:write"), dramaH.HandleBatchImportAudio)
			drama.POST("/projects/:id/assets/import", middleware.RequirePermission("drama:write"), dramaH.HandleImportAssets)
			drama.PUT("/projects/:id/assets/:assetId", middleware.RequirePermission("drama:write"), dramaH.HandleUpdateAsset)
			drama.POST("/projects/:id/assets/:assetId/reference", middleware.RequirePermission("drama:write"), dramaH.HandleUploadAssetReference)
			drama.GET("/settings", middleware.RequirePermission("drama:read"), dramaH.HandleGetSetting)
			drama.GET("/settings/comfyui/status", middleware.RequirePermission("drama:read"), dramaH.HandleComfyStatus)
			drama.PUT("/settings", middleware.RequirePermission("drama:admin"), dramaH.HandleSaveSetting)
		}
	}
	r.GET("/ws/drama/tasks", middleware.AuthRequired(jwtCfg.AccessSecret), middleware.RequirePermission("drama:read"), dramaH.HandleTaskWebSocket)

	r.GET("/healthz", system.HealthzHandler("drama-svc",
		func() (string, string) {
			sqlDB, err := db.DB()
			if err != nil {
				return "database", "error: " + err.Error()
			}
			if err := sqlDB.Ping(); err != nil {
				return "database", "error: " + err.Error()
			}
			return "database", "ok"
		},
	))

	port := cfg.Server.Port
	if env := os.Getenv("SERVER_PORT"); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil {
			port = parsed
		}
	}
	logger.Log.Info("drama-svc starting", zap.Int("port", port))
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		logger.Log.Fatal("start failed", zap.Error(err))
	}
}
