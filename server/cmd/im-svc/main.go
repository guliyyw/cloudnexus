package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/cloudnexus/server/internal/im/handler"
	"github.com/cloudnexus/server/internal/im/repository"
	"github.com/cloudnexus/server/internal/im/service"
	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/cache"
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

	if err := logger.Init(logger.Config{
		Level:   cfg.Log.Level,
		Format:  cfg.Log.Format,
		Service: "im-svc",
		LogDir:  "/app/logs",
	}); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()
	logger.StartLogCleanup()

	snowflake.Init(2)

	db, err := database.NewPostgres(database.Config{DSN: cfg.Database.DSN})
	if err != nil {
		logger.Log.Fatal("连接数据库失败", zap.Error(err))
	}

	if err := migration.Up(db); err != nil {
		logger.Log.Warn("SQL migration skipped", zap.Error(err))
	}
	if err := db.AutoMigrate(
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Message{},
		&model.Friend{},
		&model.Blocklist{},
		&model.DockerNode{},
		&model.NodeOnlineSession{},
		&model.AlertRule{},
		&model.AlertHistory{},
	); err != nil {
		logger.Log.Fatal("数据库AutoMigrate失败", zap.Error(err))
	}

	nodeReg := system.NewNodeRegistrar(db, os.Getenv("NODE_NAME"), os.Getenv("NODE_HOST"), "im-svc", 8082)
	nodeReg.Start()
	defer nodeReg.Stop()

	jwtCfg := auth.Config{
		AccessSecret:  cfg.JWT.AccessSecret,
		RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL:     time.Duration(cfg.JWT.AccessTTL) * time.Second,
		RefreshTTL:    time.Duration(cfg.JWT.RefreshTTL) * time.Second,
	}

	hub := service.NewHub(nil)
	go hub.Run()

	rdb, err := cache.NewRedis(cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.Log.Warn("Redis 连接失败，跨节点消息同步将不可用", zap.Error(err))
	} else {
		hub.EnableRedisRelay(rdb)
		logger.Log.Info("Redis 跨节点消息中继已启用")
	}

	imRepo := repository.NewIMRepository(db)
	imSvc := service.NewIMService(imRepo, hub)
	imH := handler.NewIMHandler(imSvc, hub)
	blSvc := service.NewBlocklistService(db)
	presenceSvc := service.NewPresenceService(rdb)
	friendH := handler.NewFriendEnhanceHandler(imSvc, blSvc, presenceSvc)
	hub.SetPresenceService(presenceSvc)

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	api.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
	{
		im := api.Group("/im")
		{
			im.GET("/conversations", imH.HandleGetConversations)
				im.POST("/conversations/import", imH.HandleImportConversation)
				im.GET("/conversations/:id/export", imH.HandleExportConversation)
			im.POST("/conversations", imH.HandleCreateConversation)
			im.GET("/conversations/:id/messages", imH.HandleGetMessages)
			im.DELETE("/conversations/:id", imH.HandleDeleteConversation)
			im.GET("/conversations/:id/members", imH.HandleGetGroupMembers)
			im.POST("/conversations/:id/members", imH.HandleAddGroupMember)
			im.DELETE("/conversations/:id/members/:uid", imH.HandleRemoveGroupMember)
			im.POST("/conversations/:id/leave", imH.HandleLeaveGroup)

			friends := im.Group("/friends")
			{
				friends.POST("/requests", imH.HandleSendFriendRequest)
				friends.GET("/requests", imH.HandleListPendingRequests)
				friends.PUT("/requests/:id/accept", imH.HandleAcceptRequest)
				friends.PUT("/requests/:id/reject", imH.HandleRejectRequest)
				friends.GET("", imH.HandleListFriends)
				friends.GET("/", imH.HandleListFriends)
				friends.DELETE("/:friend_id", imH.HandleRemoveFriend)
				friends.POST("/:id/block", friendH.HandleBlockUser)
				friends.DELETE("/:id/block", friendH.HandleUnblockUser)
				friends.PUT("/:id/remark", friendH.HandleSetRemark)
				friends.GET("/online", friendH.HandleGetOnlineStatus)
			}

			im.GET("/blocklist", friendH.HandleGetBlocklist)
			im.POST("/link-preview", imH.HandleLinkPreview)
		}
	}

	// WebSocket needs auth via query param
	r.GET("/ws", middleware.AuthRequired(jwtCfg.AccessSecret), imH.HandleWebSocket)

	r.GET("/healthz", system.HealthzHandler("im-svc",
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

	logger.Log.Info("im-svc starting", zap.Int("port", 8082))
	if err := r.Run(":8082"); err != nil {
		logger.Log.Fatal("启动失败", zap.Error(err))
	}
}
