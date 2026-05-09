package main

import (
	"log"
	"os"
	"time"

	"github.com/cloudnexus/server/internal/camera/handler"
	"github.com/cloudnexus/server/internal/camera/repository"
	"github.com/cloudnexus/server/internal/camera/service"
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
		Service: "camera-svc",
		LogDir:  "/app/logs",
	}); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()
	logger.StartLogCleanup()

	snowflake.Init(5)

	db, err := database.NewPostgres(database.Config{DSN: cfg.Database.DSN})
	if err != nil {
		logger.Log.Fatal("连接数据库失败", zap.Error(err))
	}

	if err := migration.Up(db); err != nil {
		logger.Log.Warn("SQL migration skipped", zap.Error(err))
	}
	if err := db.AutoMigrate(
		&model.Camera{},
		&model.RecognitionEvent{},
		&model.FaceProfile{},
		&model.FaceRecognitionEvent{},
		&model.FaceAttendanceSession{},
	); err != nil {
		logger.Log.Fatal("数据库 AutoMigrate 失败", zap.Error(err))
	}

	nodeReg := system.NewNodeRegistrar(db, os.Getenv("NODE_NAME"), os.Getenv("NODE_HOST"), "camera-svc", 8085)
	nodeReg.Start()
	defer nodeReg.Stop()

	jwtCfg := auth.Config{
		AccessSecret:  cfg.JWT.AccessSecret,
		RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL:     time.Duration(cfg.JWT.AccessTTL) * time.Second,
		RefreshTTL:    time.Duration(cfg.JWT.RefreshTTL) * time.Second,
	}

	repo := repository.NewCameraRepository(db)

	mediamtxURL := os.Getenv("MEDIAMTX_API_URL")
	if mediamtxURL == "" {
		mediamtxURL = "http://mediamtx:8889"
	}
	inferenceURL := os.Getenv("AI_INFERENCE_URL")
	if inferenceURL == "" {
		inferenceURL = "http://ai-inference:8000"
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

	camSvc := service.NewCameraService(repo, mediamtxURL)
	recSvc := service.NewRecognitionService(repo, inferenceURL)
	faceSvc := service.NewFaceService(repo, minioClient, cfg.MinIO.Bucket)
	camH := handler.NewCameraHandler(camSvc, recSvc)
	faceH := handler.NewFaceHandler(faceSvc)

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	api.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
	{
		cameras := api.Group("/cameras")
		{
			cameras.GET("", camH.HandleListCameras)
			cameras.POST("", camH.HandleCreateCamera)
			cameras.PUT("/:id", camH.HandleUpdateCamera)
			cameras.DELETE("/:id", camH.HandleDeleteCamera)
			cameras.POST("/discover", camH.HandleDiscoverCameras)
			cameras.POST("/:id/stream/start", camH.HandleStartStream)
			cameras.POST("/:id/stream/stop", camH.HandleStopStream)
			cameras.POST("/:id/recognition/start", camH.HandleStartRecognition)
			cameras.POST("/:id/recognition/stop", camH.HandleStopRecognition)
			cameras.GET("/:id/events", camH.HandleListEvents)
			cameras.GET("/:id/faces", faceH.HandleListFaceEvents)
			cameras.DELETE("/:id/faces", faceH.HandleClearFaceEvents)
		}
		faces := api.Group("/faces")
		{
			faces.GET("", faceH.HandleListProfiles)
			faces.POST("", faceH.HandleCreateProfile)
			faces.PUT("/:id", faceH.HandleUpdateProfile)
			faces.DELETE("/:id", faceH.HandleDeleteProfile)
			faces.POST("/match", faceH.HandleMatchFace)
			faces.GET("/:id/thumbnail", faceH.HandleGetThumbnail)
			faces.GET("/attendance", faceH.HandleGetAttendanceByFace)
			faces.GET("/attendance/daily", faceH.HandleGetDailyAttendance)
			faces.GET("/attendance/status", faceH.HandleGetAttendanceStatus)
			faces.DELETE("/attendance/:id", faceH.HandleDeleteAttendanceSession)
			faces.DELETE("/attendance", faceH.HandleClearAttendance)
		}
		api.POST("/detect-image", camH.HandleDetectImage)
	}

	r.GET("/healthz", system.HealthzHandler("camera-svc",
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

	logger.Log.Info("camera-svc starting", zap.Int("port", 8085))
	if err := r.Run(":8085"); err != nil {
		logger.Log.Fatal("启动失败", zap.Error(err))
	}
}
