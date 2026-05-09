package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/cloudnexus/server/internal/collab/handler"
	"github.com/cloudnexus/server/internal/collab/service"
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
		log.Fatalf("加载配置失败: %v", err)
	}

	if err := logger.Init(logger.Config{
		Level:   cfg.Log.Level,
		Format:  cfg.Log.Format,
		Service: "collab-svc",
		LogDir:  "/app/logs",
	}); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()
	logger.StartLogCleanup()

	snowflake.Init(6)

	db, err := database.NewPostgres(database.Config{DSN: cfg.Database.DSN})
	if err != nil {
		logger.Log.Fatal("连接数据库失败", zap.Error(err))
	}

	if err := migration.Up(db); err != nil {
		logger.Log.Warn("SQL migration skipped", zap.Error(err))
	}
	if err := db.AutoMigrate(
		&model.File{},
	); err != nil {
		logger.Log.Fatal("数据库 AutoMigrate 失败", zap.Error(err))
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

	nodeReg := system.NewNodeRegistrar(db, os.Getenv("NODE_NAME"), os.Getenv("NODE_HOST"), "collab-svc", 8086)
	nodeReg.Start()
	defer nodeReg.Stop()

	jwtCfg := auth.Config{
		AccessSecret:  cfg.JWT.AccessSecret,
		RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL:     time.Duration(cfg.JWT.AccessTTL) * time.Second,
		RefreshTTL:    time.Duration(cfg.JWT.RefreshTTL) * time.Second,
	}

	docHub := service.NewDocHub(db, minioClient, cfg.MinIO.Bucket)

	// Redis 跨节点同步
	rdb, err := cache.NewRedis(cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.Log.Warn("Redis 不可用，跨节点同步已禁用", zap.Error(err))
	} else {
		docHub.EnableRedisRelay(rdb)
	}

	collabH := handler.NewCollabHandler(docHub)

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// WebSocket for Yjs (auth via query param)
	r.GET("/ws/collab/:id", middleware.AuthRequired(jwtCfg.AccessSecret), collabH.HandleWebSocket)

	r.GET("/healthz", system.HealthzHandler("collab-svc",
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
		func() (string, string) {
			if rdb == nil {
				return "redis", "disabled"
			}
			if err := rdb.Ping(context.Background()).Err(); err != nil {
				return "redis", "error: " + err.Error()
			}
			return "redis", "ok"
		},
	))

	logger.Log.Info("collab-svc starting", zap.Int("port", 8086))
	if err := r.Run(":8086"); err != nil {
		logger.Log.Fatal("启动失败", zap.Error(err))
	}
}
