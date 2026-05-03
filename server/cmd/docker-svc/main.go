package main

import (
	"log"
	"os"
	"time"

	"github.com/cloudnexus/server/internal/dockermgr/handler"
	"github.com/cloudnexus/server/internal/dockermgr/service"
	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/config"
	"github.com/cloudnexus/server/pkg/middleware"

	"github.com/gin-gonic/gin"
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

	dockerSvc, err := service.NewDockerService()
	if err != nil {
		log.Fatalf("连接 Docker 失败: %v", err)
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
			}
		}
	}

	r.GET("/healthz", healthCheck)

	log.Println("docker-svc starting on :8083")
	if err := r.Run(":8083"); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok", "service": "docker-svc"})
}
