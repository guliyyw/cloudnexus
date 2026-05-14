package service

import (
	"time"

	"github.com/cloudnexus/server/internal/userfile/repository"
	"github.com/cloudnexus/server/pkg/logger"
	"go.uber.org/zap"
)

type CleanupScheduler struct {
	trashSvc  *TrashService
	quotaSvc  *QuotaService
	chunkRepo *repository.ChunkRepository
}

func NewCleanupScheduler(trashSvc *TrashService, quotaSvc *QuotaService, chunkRepo *repository.ChunkRepository) *CleanupScheduler {
	return &CleanupScheduler{trashSvc: trashSvc, quotaSvc: quotaSvc, chunkRepo: chunkRepo}
}

func (s *CleanupScheduler) Start() {
	// Recycle bin cleanup: every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			s.runTrashCleanup()
		}
	}()

	// Chunk expiration cleanup: every hour (cancel uploads older than 24h)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			s.runChunkCleanup()
		}
	}()

	// Quota reconciliation: every 6 hours
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			s.runQuotaReconcile()
		}
	}()
}

func (s *CleanupScheduler) runTrashCleanup() {
	threshold := time.Now().Add(-30 * 24 * time.Hour)
	deleted, err := s.trashSvc.CleanupExpiredTrash(threshold)
	if err != nil {
		logger.Log.Warn("[cleanup] 回收站清理失败", zap.Error(err))
	} else if deleted > 0 {
		logger.Log.Info("[cleanup] 已清理过期文件", zap.Int64("deleted", deleted))
	}
}

func (s *CleanupScheduler) runChunkCleanup() {
	threshold := time.Now().Add(-24 * time.Hour)
	if err := s.chunkRepo.DeleteExpired(threshold); err != nil {
		logger.Log.Warn("[cleanup] 分片清理失败", zap.Error(err))
	}
}

func (s *CleanupScheduler) runQuotaReconcile() {
	if err := s.quotaSvc.ReconcileAll(); err != nil {
		logger.Log.Warn("[cleanup] 配额校准失败", zap.Error(err))
	} else {
		logger.Log.Info("[cleanup] 配额校准完成")
	}
}
