package main

import (
	"fmt"
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

	if err := logger.Init(logger.Config{Level: cfg.Log.Level, Format: cfg.Log.Format, Service: "docker-svc", LogDir: "/app/logs"}); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	snowflake.Init(3) // node 1=user-file, 2=im, 3=docker, 4=reserved, 5=camera, 6=collab

	db, err := database.NewPostgres(database.Config{DSN: cfg.Database.DSN})
	if err != nil {
		logger.Log.Warn("连接数据库失败，节点注册和远程端点不可用", zap.Error(err))
	}

	if db != nil {
		if err := migration.Up(db); err != nil {
			logger.Log.Warn("SQL migration skipped", zap.Error(err))
		}
		if err := db.AutoMigrate(&model.NodeOnlineSession{}); err != nil {
			logger.Log.Warn("DockerNode AutoMigrate 失败", zap.Error(err))
		}
		nodeReg := system.NewNodeRegistrar(db, os.Getenv("NODE_NAME"), os.Getenv("NODE_HOST"), "docker-svc", 8083)
		nodeReg.Start()
		defer nodeReg.Stop()
	}

	endpointMgr := service.NewEndpointManager(db)
	dockerSvc := service.NewDockerService(endpointMgr)

	// Background: ping all Docker endpoints every 30s and update status
	if db != nil {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				endpointMgr.PingAll()
			}
		}()
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
	api.Use(middleware.LoadPermissions(db))
	{
		docker := api.Group("/docker")
		docker.Use(middleware.RequirePermission("module:docker"))
		{
			docker.GET("/endpoints", dockerH.HandleListEndpoints)
			docker.GET("/ping", dockerH.HandlePingEndpoint)

			containers := docker.Group("/containers")
			{
				containers.GET("", middleware.RequirePermission("docker:read"), dockerH.HandleListContainers)
				containers.POST("", middleware.RequirePermission("docker:admin"), dockerH.HandleCreateContainer)
				containers.POST("/:id/start", middleware.RequirePermission("docker:control"), dockerH.HandleStartContainer)
				containers.POST("/:id/stop", middleware.RequirePermission("docker:control"), dockerH.HandleStopContainer)
				containers.POST("/:id/restart", middleware.RequirePermission("docker:control"), dockerH.HandleRestartContainer)
				containers.DELETE("/:id", middleware.RequirePermission("docker:admin"), dockerH.HandleRemoveContainer)
				containers.GET("/:id/logs", middleware.RequirePermission("docker:read"), dockerH.HandleGetLogs)
				containers.GET("/:id/stats", middleware.RequirePermission("docker:read"), dockerH.HandleGetStats)
			}
			images := docker.Group("/images")
			{
				images.GET("", middleware.RequirePermission("docker:read"), dockerH.HandleListImages)
				images.POST("/pull", middleware.RequirePermission("docker:admin"), dockerH.HandlePullImage)
				images.DELETE("/:image", middleware.RequirePermission("docker:admin"), dockerH.HandleRemoveImage)
			}
		}
	}

	r.GET("/healthz", system.HealthzHandler("docker-svc",
		func() (string, string) {
			eps := endpointMgr.ListEndpoints()
			for _, ep := range eps {
				if err := endpointMgr.PingEndpoint(ep.Name); err != nil {
					return "docker:" + ep.Name, "error: " + err.Error()
				}
			}
			return "docker", "ok"
		},
	))

	logger.Log.Info("docker-svc starting", zap.Int("port", cfg.Server.Port))
	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		logger.Log.Fatal("启动失败", zap.Error(err))
	}
}
