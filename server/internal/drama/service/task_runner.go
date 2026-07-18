package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cloudnexus/server/internal/drama/repository"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/redis/go-redis/v9"
)

const dramaTaskQueue = "drama:tasks:queue"

type TaskEvent struct {
	Type string          `json:"type"`
	Task model.DramaTask `json:"task"`
}

type TaskRunner struct {
	svc         *DramaService
	repo        *repository.DramaRepository
	redis       *redis.Client
	mu          sync.Mutex
	cancelFuncs map[uint64]context.CancelFunc
	subscribers map[uint64]map[chan TaskEvent]struct{}
}

func NewTaskRunner(svc *DramaService, repo *repository.DramaRepository, redisClient *redis.Client) *TaskRunner {
	return &TaskRunner{
		svc:         svc,
		repo:        repo,
		redis:       redisClient,
		cancelFuncs: make(map[uint64]context.CancelFunc),
		subscribers: make(map[uint64]map[chan TaskEvent]struct{}),
	}
}

func (r *TaskRunner) Start(ctx context.Context) error {
	if err := r.repo.RecoverRunningTasks(); err != nil {
		return err
	}
	pending, err := r.repo.ListPendingTasks()
	if err != nil {
		return err
	}
	for _, task := range pending {
		if err := r.Enqueue(task.ID); err != nil {
			return err
		}
	}
	go r.run(ctx)
	return nil
}

func (r *TaskRunner) Enqueue(taskID uint64) error {
	return r.redis.LPush(context.Background(), dramaTaskQueue, strconv.FormatUint(taskID, 10)).Err()
}

func (r *TaskRunner) Subscribe(ownerID uint64) (<-chan TaskEvent, func()) {
	ch := make(chan TaskEvent, 16)
	r.mu.Lock()
	if r.subscribers[ownerID] == nil {
		r.subscribers[ownerID] = make(map[chan TaskEvent]struct{})
	}
	r.subscribers[ownerID][ch] = struct{}{}
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		if subscribers := r.subscribers[ownerID]; subscribers != nil {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(r.subscribers, ownerID)
			}
		}
		r.mu.Unlock()
	}
}

func (r *TaskRunner) Publish(task model.DramaTask) {
	event := TaskEvent{Type: "task_update", Task: task}
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.subscribers[task.OwnerID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (r *TaskRunner) Cancel(ownerID, projectID, taskID uint64) (*model.DramaTask, error) {
	task, err := r.repo.GetTask(ownerID, projectID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Status != "pending" && task.Status != "running" {
		return task, nil
	}
	r.mu.Lock()
	cancel := r.cancelFuncs[taskID]
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	now := time.Now()
	task.Status = "canceled"
	task.Message = "任务已取消"
	task.FinishedAt = &now
	if err := r.repo.UpdateTask(task); err != nil {
		return nil, err
	}
	r.Publish(*task)
	return task, nil
}

func (r *TaskRunner) Retry(ownerID, projectID, taskID uint64) (*model.DramaTask, error) {
	task, err := r.repo.GetTask(ownerID, projectID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Status != "failed" && task.Status != "canceled" {
		return nil, fmt.Errorf("只有失败或已取消的任务可以重试")
	}
	task.Status = "pending"
	task.Progress = 0
	task.Message = "任务已重新排队"
	task.StartedAt = nil
	task.FinishedAt = nil
	if err := r.repo.UpdateTask(task); err != nil {
		return nil, err
	}
	if err := r.Enqueue(task.ID); err != nil {
		return nil, err
	}
	r.Publish(*task)
	return task, nil
}

func (r *TaskRunner) run(ctx context.Context) {
	for {
		result, err := r.redis.BRPop(ctx, 5*time.Second, dramaTaskQueue).Result()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			continue
		}
		if len(result) != 2 {
			continue
		}
		taskID, err := strconv.ParseUint(result[1], 10, 64)
		if err != nil {
			continue
		}
		r.process(ctx, taskID)
	}
}

func (r *TaskRunner) process(parent context.Context, taskID uint64) {
	claimed, err := r.repo.ClaimTask(taskID)
	if err != nil || !claimed {
		return
	}
	task, err := r.repo.GetTaskByID(taskID)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	r.mu.Lock()
	r.cancelFuncs[taskID] = cancel
	r.mu.Unlock()
	defer func() {
		cancel()
		r.mu.Lock()
		delete(r.cancelFuncs, taskID)
		r.mu.Unlock()
	}()

	task.Progress = 5
	task.Message = "正在准备生成参数"
	r.svc.clearTaskGeneratedPayload(task)
	_ = r.repo.UpdateTask(task)
	r.Publish(*task)

	update := func(progress int, message string) {
		if ctx.Err() != nil {
			return
		}
		task.Progress = progress
		task.Message = message
		_ = r.repo.UpdateTask(task)
		r.Publish(*task)
	}
	err = r.svc.executeGenerationTask(ctx, task, update)
	if latest, latestErr := r.repo.GetTaskByID(task.ID); latestErr == nil {
		task = latest
	}
	now := time.Now()
	task.FinishedAt = &now
	if ctx.Err() != nil {
		task.Status = "canceled"
		task.Message = "任务已取消"
	} else if err != nil {
		task.Status = "failed"
		task.Message = err.Error()
	} else {
		task.Status = "done"
		task.Progress = 100
		task.Message = "生成完成，结果已保存到云盘"
	}
	_ = r.repo.UpdateTask(task)
	r.Publish(*task)
}
