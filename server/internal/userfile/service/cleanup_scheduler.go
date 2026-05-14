package service

import (
	"log"
	"time"

	"gorm.io/gorm"
)

type CleanupScheduler struct {
	db       *gorm.DB
	trashSvc *TrashService
	quotaSvc *QuotaService
}

func NewCleanupScheduler(db *gorm.DB, trashSvc *TrashService, quotaSvc *QuotaService) *CleanupScheduler {
	return &CleanupScheduler{db: db, trashSvc: trashSvc, quotaSvc: quotaSvc}
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
		log.Printf("[cleanup] 回收站清理失败: %v", err)
	} else if deleted > 0 {
		log.Printf("[cleanup] 已清理 %d 个过期文件", deleted)
	}
}

func (s *CleanupScheduler) runQuotaReconcile() {
	if err := s.quotaSvc.ReconcileAll(); err != nil {
		log.Printf("[cleanup] 配额校准失败: %v", err)
	} else {
		log.Println("[cleanup] 配额校准完成")
	}
}
