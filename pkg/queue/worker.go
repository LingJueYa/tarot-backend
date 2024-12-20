package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"tarot/pkg/dify"
	"tarot/pkg/logger"
	"tarot/app/models/reading"
	"tarot/app/repositories"
)

// contextKey 自定义上下文键类型
type contextKey string

// 预定义上下文键
const (
	taskIDKey contextKey = "task_id"
)

// Worker 队列工作器
type Worker struct {
	queueService *QueueService
	difyService  *dify.DifyService
	stopChan     chan struct{}
	workerCount  int           // 工作器数量
	metrics      *QueueMetrics // 性能指标
	wg           sync.WaitGroup
	config       WorkerConfig
}

// WorkerConfig 工作器配置
type WorkerConfig struct {
	WorkerCount     int           // 并发工作器数量
	MaxRetries      int           // 最大重试次数
	RetryInterval   time.Duration // 重试间隔
	ShutdownTimeout time.Duration // 关闭超时时间
	BatchSize       int           // 批处理大小
	MaxQueueSize    int          // 最大队列长度
}

// NewWorker 创建新的工作器组
func NewWorker(qs *QueueService, ds *dify.DifyService, config WorkerConfig) *Worker {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 10 // 默认工作器数量
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 10 // 默认批处理大小
	}
	if config.MaxQueueSize <= 0 {
		config.MaxQueueSize = 10000 // 默认最大队列长度
	}

	return &Worker{
		queueService: qs,
		difyService:  ds,
		stopChan:     make(chan struct{}),
		workerCount:  config.WorkerCount,
		metrics:      NewQueueMetrics(),
		config:      config,
	}
}

// Start 启动工作器组
func (w *Worker) Start() {
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.startWorker(i)
	}
}

// startWorker 启动单个工作器
func (w *Worker) startWorker(id int) {
	defer w.wg.Done()

	logger.InfoString("Worker", "Start", fmt.Sprintf("Worker %d started", id))

	// 创建带缓冲的错误通道
	errorChan := make(chan error, 100)

	for {
		select {
		case <-w.stopChan:
			logger.InfoString("Worker", "Stop", fmt.Sprintf("Worker %d stopping", id))
			return

		case err := <-errorChan:
			logger.ErrorString("Worker", "Error", fmt.Sprintf("Worker %d error: %v", id, err))
			time.Sleep(time.Second) // 错误恢复延迟

		default:
			if err := w.processNextTask(); err != nil {
				select {
				case errorChan <- err:
				default:
					logger.ErrorString("Worker", "ErrorBuffer", "Error buffer full")
				}
			}
		}
	}
}

// processNextTask 优化任务处理逻辑
func (w *Worker) processNextTask() error {
	start := time.Now()
	defer func() {
		w.metrics.RecordProcessingTime(time.Since(start))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 使用管道缓冲任务
	taskChan := make(chan *TarotTask, w.config.BatchSize)
	errChan := make(chan error, 1)

	// 异步获取任务
	go func() {
		task, err := w.queueService.PopTask(ctx)
		if err != nil {
			if err != redis.Nil {
				errChan <- fmt.Errorf("pop task error: %w", err)
			}
			close(taskChan)
			return
		}
		taskChan <- task
		close(taskChan)
	}()

	// 等待任务或错误
	select {
	case err := <-errChan:
		return err
	case task, ok := <-taskChan:
		if !ok {
			time.Sleep(100 * time.Millisecond) // 避免空队列时的忙等
			return nil
		}

		// 处理任务
		return w.handleTask(ctx, task)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// handleTask 处理单个任务
func (w *Worker) handleTask(ctx context.Context, task *TarotTask) error {
	w.metrics.EndWaitTime(TaskID(task.ID))
	
	// 使用带超时的上下文
	taskCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	
	// 更新状态为处理中
	if err := w.queueService.UpdateTaskStatus(ctx, task.ID, TaskRunning, ""); err != nil {
		return fmt.Errorf("update task status error: %w", err)
	}
	
	// 处理任务
	result, err := w.processTask(taskCtx, task)
	if err != nil {
		w.metrics.RecordError(OpProcess)
		if updateErr := w.queueService.UpdateTaskStatus(ctx, task.ID, TaskFailed, err.Error()); updateErr != nil {
			logger.ErrorString("Worker", "UpdateStatus", updateErr.Error())
		}
		return fmt.Errorf("process task error: %w", err)
	}
	
	// 创建数据库记录
	readingRepo := repositories.NewReadingRepository()
	reading := &reading.Reading{
		TaskID:         task.ID,
		UserID:         task.UserID,
		Question:       task.Question,
		Cards:          task.Cards,
		Interpretation: result,
		Status:         string(TaskCompleted),
	}
	
	if err := readingRepo.Create(ctx, reading); err != nil {
		logger.ErrorString("Worker", "SaveReading", err.Error())
		// 不要因为保存失败而影响任务状态
	}
	
	// 更新任务状态
	if err := w.queueService.UpdateTaskStatus(ctx, task.ID, TaskCompleted, result); err != nil {
		return fmt.Errorf("update task result error: %w", err)
	}
	
	w.metrics.RecordSuccess(OpProcess)
	return nil
}

// processTask 处理单个任务
func (w *Worker) processTask(ctx context.Context, task *TarotTask) (string, error) {
	// 添加追踪信息 - 使用自定义类型的键
	ctx = context.WithValue(ctx, taskIDKey, task.ID)

	// 调用 Dify API
	result, err := w.difyService.ProcessTarotReading(ctx, task.Question, task.Cards)
	if err != nil {
		return "", fmt.Errorf("dify processing error: %w", err)
	}

	return result, nil
}

// Stop 优雅关闭工作器组
func (w *Worker) Stop() {
	close(w.stopChan)

	// 等待所有工作器完成
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	// 设置超时时间
	timeout := time.After(30 * time.Second)

	select {
	case <-done:
		logger.InfoString("Worker", "Stop", "All workers stopped gracefully")
	case <-timeout:
		logger.WarnString("Worker", "Stop", "Worker shutdown timed out")
	}
}
