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
	"github.com/cloudnexus/server/pkg/middleware"
	"github.com/cloudnexus/server/pkg/model"

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

	db, err := database.NewPostgres(database.Config{DSN: cfg.Database.DSN})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.RefreshToken{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
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
			}
		}
	}

	r.GET("/healthz", healthCheck)

	log.Printf("user-file-svc starting on :%d", cfg.Server.Port)
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok", "service": "user-file-svc"})
}
