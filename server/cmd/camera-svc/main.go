package main

import (
	"fmt"
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
		&model.CameraRecording{},
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

	inferenceToken := os.Getenv("AI_INFERENCE_TOKEN")

	camSvc := service.NewCameraService(repo, mediamtxURL)
	recordingDir := os.Getenv("CAMERA_RECORDING_DIR")
	if recordingDir == "" {
		recordingDir = "/app/recordings"
	}
	recordingSvc := service.NewRecordingService(repo, recordingDir)
	recSvc := service.NewRecognitionService(repo, inferenceURL, inferenceToken)
	faceSvc := service.NewFaceService(repo, minioClient, cfg.MinIO.Bucket)
	camH := handler.NewCameraHandler(camSvc, recSvc)
	recordingH := handler.NewRecordingHandler(recordingSvc)
	faceH := handler.NewFaceHandler(faceSvc)

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	api.Use(middleware.AuthRequired(jwtCfg.AccessSecret))
	api.Use(middleware.LoadPermissions(db))
	api.Use(middleware.RequirePermission("module:cameras"))
	{
		cameras := api.Group("/cameras")
		{
			cameras.GET("", middleware.RequirePermission("camera:read"), camH.HandleListCameras)
			cameras.POST("", middleware.RequirePermission("camera:admin"), camH.HandleCreateCamera)
			cameras.PUT("/:id", middleware.RequirePermission("camera:admin"), camH.HandleUpdateCamera)
			cameras.DELETE("/:id", middleware.RequirePermission("camera:admin"), camH.HandleDeleteCamera)
			cameras.POST("/discover", middleware.RequirePermission("camera:admin"), camH.HandleDiscoverCameras)
			cameras.POST("/:id/stream/start", middleware.RequirePermission("camera:control"), camH.HandleStartStream)
			cameras.POST("/:id/stream/stop", middleware.RequirePermission("camera:control"), camH.HandleStopStream)
			cameras.POST("/:id/recording/start", middleware.RequirePermission("camera:control"), recordingH.HandleStart)
			cameras.POST("/:id/recording/stop", middleware.RequirePermission("camera:control"), recordingH.HandleStop)
			cameras.GET("/:id/recording/status", middleware.RequirePermission("camera:read"), recordingH.HandleStatus)
			cameras.GET("/:id/recordings", middleware.RequirePermission("camera:read"), recordingH.HandleList)
			cameras.GET("/:id/recordings/:recording_id/play", middleware.RequirePermission("camera:read"), recordingH.HandlePlayback)
			cameras.DELETE("/:id/recordings/:recording_id", middleware.RequirePermission("camera:control"), recordingH.HandleDelete)
			cameras.POST("/:id/recognition/start", middleware.RequirePermission("camera:control"), camH.HandleStartRecognition)
			cameras.POST("/:id/recognition/stop", middleware.RequirePermission("camera:control"), camH.HandleStopRecognition)
			cameras.GET("/:id/events", middleware.RequirePermission("camera:read"), camH.HandleListEvents)
			cameras.GET("/:id/faces", middleware.RequirePermission("face:read"), faceH.HandleListFaceEvents)
			cameras.DELETE("/:id/faces", middleware.RequirePermission("face:admin"), faceH.HandleClearFaceEvents)
		}
		faces := api.Group("/faces")
		{
			faces.GET("", middleware.RequirePermission("face:read"), faceH.HandleListProfiles)
			faces.POST("", middleware.RequirePermission("face:write"), faceH.HandleCreateProfile)
			faces.PUT("/:id", middleware.RequirePermission("face:write"), faceH.HandleUpdateProfile)
			faces.DELETE("/:id", middleware.RequirePermission("face:admin"), faceH.HandleDeleteProfile)
			faces.POST("/match", middleware.RequirePermission("face:read"), faceH.HandleMatchFace)
			faces.GET("/:id/thumbnail", middleware.RequirePermission("face:read"), faceH.HandleGetThumbnail)
			faces.GET("/attendance", middleware.RequirePermission("attendance:read"), faceH.HandleGetAttendanceByFace)
			faces.GET("/attendance/daily", middleware.RequirePermission("attendance:read"), faceH.HandleGetDailyAttendance)
			faces.GET("/attendance/status", middleware.RequirePermission("attendance:read"), faceH.HandleGetAttendanceStatus)
			faces.DELETE("/attendance/:id", middleware.RequirePermission("attendance:admin"), faceH.HandleDeleteAttendanceSession)
			faces.DELETE("/attendance", middleware.RequirePermission("attendance:admin"), faceH.HandleClearAttendance)
		}
		api.POST("/detect-image", middleware.RequirePermission("camera:read"), camH.HandleDetectImage)
		api.POST("/detect-video", middleware.RequirePermission("camera:read"), camH.HandleDetectVideo)
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

	logger.Log.Info("camera-svc starting", zap.Int("port", cfg.Server.Port))
	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		logger.Log.Fatal("启动失败", zap.Error(err))
	}
}
