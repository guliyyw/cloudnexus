package main

import (
	"log"
	"os"
	"time"

	"github.com/cloudnexus/server/internal/dockermgr/handler"
	"github.com/cloudnexus/server/internal/dockermgr/service"
	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/config"
	"github.com/cloudnexus/server/pkg/database"
	"github.com/cloudnexus/server/pkg/logger"
	"github.com/cloudnexus/server/pkg/middleware"
	"github.com/cloudnexus/server/pkg/migration"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/snowflake"
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

	if err := logger.Init(logger.Config{Level: cfg.Log.Level, Format: cfg.Log.Format}); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	dockerSvc, err := service.NewDockerService()
	if err != nil {
		logger.Log.Fatal("连接 Docker 失败", zap.Error(err))
	}

	snowflake.Init(3)

	db, err := database.NewPostgres(database.Config{DSN: cfg.Database.DSN})
	if err != nil {
		logger.Log.Warn("连接数据库失败，节点注册不可用", zap.Error(err))
	}

	if db != nil {
		if err := migration.Up(db); err != nil {
			logger.Log.Warn("SQL migration skipped", zap.Error(err))
		}
		if err := db.AutoMigrate(&model.DockerNode{}, &model.NodeOnlineSession{}); err != nil {
			logger.Log.Warn("DockerNode AutoMigrate 失败", zap.Error(err))
		}
		nodeReg := system.NewNodeRegistrar(db, os.Getenv("NODE_NAME"), os.Getenv("NODE_HOST"), "docker-svc", 8083)
		nodeReg.Start()
		defer nodeReg.Stop()
	}

	jwtCfg := auth.Config{
		AccessSecret:  cfg.JWT.AccessSecret,
		RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL:     time.Duration(cfg.JWT.AccessTTL) * time.Second,
		RefreshTTL:    time.Duration(cfg.JWT.RefreshTTL) * time.Second,
	}

	dockerH := handler.NewDockerHandler(dockerSvc)

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	api.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
	{
		docker := api.Group("/docker")
		{
			containers := docker.Group("/containers")
			{
				containers.GET("", dockerH.HandleListContainers)
				containers.POST("", dockerH.HandleCreateContainer)
				containers.POST("/:id/start", dockerH.HandleStartContainer)
				containers.POST("/:id/stop", dockerH.HandleStopContainer)
				containers.POST("/:id/restart", dockerH.HandleRestartContainer)
				containers.DELETE("/:id", dockerH.HandleRemoveContainer)
				containers.GET("/:id/logs", dockerH.HandleGetLogs)
				containers.GET("/:id/stats", dockerH.HandleGetStats)
			}
			images := docker.Group("/images")
			{
				images.GET("", dockerH.HandleListImages)
				images.POST("/pull", dockerH.HandlePullImage)
				images.DELETE("/:image", dockerH.HandleRemoveImage)
			}
		}
	}

	r.GET("/healthz", system.HealthzHandler("docker-svc",
		func() (string, string) {
			if err := dockerSvc.Ping(); err != nil {
				return "docker", "error: " + err.Error()
			}
			return "docker", "ok"
		},
	))

	logger.Log.Info("docker-svc starting", zap.Int("port", 8083))
	if err := r.Run(":8083"); err != nil {
		logger.Log.Fatal("启动失败", zap.Error(err))
	}
}
