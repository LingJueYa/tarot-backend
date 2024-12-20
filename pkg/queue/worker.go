package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tarot/pkg/dify"
	"tarot/pkg/logger"
)

// 错误常量定义
var (
	ErrQueueEmpty = errors.New("queue is empty")
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
	workerCount  int
	metrics      *QueueMetrics
	wg           sync.WaitGroup
	config       WorkerConfig
	cancel       context.CancelFunc
	ctx          context.Context
	timeout      time.Duration
	retryConfig  RetryConfig
}

// WorkerConfig 工作器配置
type WorkerConfig struct {
	WorkerCount     int           // 并发工作器数量
	MaxRetries      int           // 最大重试次数
	RetryInterval   time.Duration // 重试间隔
	ShutdownTimeout time.Duration // 关闭超时时间
	BatchSize       int           // 批处理大小
	MaxQueueSize    int           // 最大队列长度
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int
	RetryInterval time.Duration
	Timeout       time.Duration
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

	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		queueService: qs,
		difyService:  ds,
		stopChan:     make(chan struct{}),
		workerCount:  config.WorkerCount,
		metrics:      NewQueueMetrics(),
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
		timeout:      30 * time.Second,
		retryConfig: RetryConfig{
			MaxRetries:    3,
			RetryInterval: 5 * time.Second,
			Timeout:       30 * time.Second,
		},
	}
}

// Start 启动工作器组
func (w *Worker) Start() {
	logger.InfoString("Worker", "Start", fmt.Sprintf("Starting %d workers", w.workerCount))

	w.wg.Add(w.workerCount)
	for i := 0; i < w.workerCount; i++ {
		go func(id int) {
			defer w.wg.Done()
			if err := w.startWorker(id); err != nil {
				logger.ErrorString("Worker", "Error",
					fmt.Sprintf("Worker %d error: %v", id, err))
			}
		}(i)
	}
}

// startWorker 启动单个工作器
func (w *Worker) startWorker(id int) error {
	logger.InfoString("Worker", "Start", fmt.Sprintf("Worker %d started", id))

	for {
		select {
		case <-w.ctx.Done():
			logger.InfoString("Worker", "Stop", fmt.Sprintf("Worker %d stopping", id))
			return nil
		default:
			// 尝试获取任务
			task, err := w.queueService.DequeueTask(w.ctx)
			if err != nil {
				if err == ErrQueueEmpty {
					// 队列为空，等待一段时间后重试
					time.Sleep(1 * time.Second)
					continue
				}
				// 记录错误并继续
				logger.ErrorString("Worker", "Error",
					fmt.Sprintf("Worker %d dequeue error: %v", id, err))
				continue
			}

			// 执行任务
			if err := w.executeTask(w.ctx, task, id); err != nil {
				logger.ErrorString("Worker", "Error",
					fmt.Sprintf("Worker %d execution error: %v", id, err))
			}
		}
	}
}

// executeTask 执行单个任务
func (w *Worker) executeTask(ctx context.Context, task *TarotTask, workerID int) error {
	start := time.Now()
	defer func() {
		w.metrics.RecordProcessingTime(time.Since(start))
	}()

	// 更新状态���中
	if err := w.queueService.UpdateTaskStatus(ctx, task.ID, TaskRunning, ""); err != nil {
		return fmt.Errorf("update task status error: %w", err)
	}

	// 处理任务
	err := w.processTask(ctx, task)
	if err != nil {
		w.metrics.RecordError(OpProcess)
		if updateErr := w.queueService.UpdateTaskStatus(ctx, task.ID, TaskFailed, err.Error()); updateErr != nil {
			logger.ErrorString("Worker", "UpdateStatus", updateErr.Error())
		}
		return fmt.Errorf("process task error: %w", err)
	}

	w.metrics.RecordSuccess(OpProcess)
	logger.InfoString("Worker", "Success",
		fmt.Sprintf("Worker %d completed task %s", workerID, task.ID))
	return nil
}

// processTask 处理任务的核心逻辑
func (w *Worker) processTask(ctx context.Context, task *TarotTask) error {
	var lastErr error

	// 重试循环
	for attempt := 0; attempt <= w.retryConfig.MaxRetries; attempt++ {
		// 如果不是第一次尝试，记录重试信息
		if attempt > 0 {
			logger.InfoString("Worker", "Retry",
				fmt.Sprintf("Retrying task %s, attempt %d of %d",
					task.ID, attempt, w.retryConfig.MaxRetries))

			// 添加重试延迟
			select {
			case <-ctx.Done():
				return fmt.Errorf("task cancelled during retry wait: %w", ctx.Err())
			case <-time.After(w.retryConfig.RetryInterval):
			}
		}

		// 执行任务
		err := w.executeTaskWithTimeout(ctx, task)
		if err == nil {
			return nil // 任务成功完成
		}

		lastErr = err
		logger.WarnString("Worker", "TaskError",
			fmt.Sprintf("Task %s failed attempt %d: %v", task.ID, attempt, err))

		// 检查是否是致命错误（不需要重试）
		if isFatalError(err) {
			return fmt.Errorf("fatal error occurred: %w", err)
		}
	}

	// 所有重试都失败后返回错误
	if lastErr != nil {
		return fmt.Errorf("task %s failed after %d attempts: %w",
			task.ID, w.retryConfig.MaxRetries+1, lastErr)
	}
	return errors.New("task failed with unknown error")
}

// executeTaskWithTimeout 在超时限制内执行任务
func (w *Worker) executeTaskWithTimeout(ctx context.Context, task *TarotTask) error {
	taskCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// 获取可用的 Dify 实例
	instance, err := w.difyService.GetHealthyInstance()
	if err != nil {
		return fmt.Errorf("failed to get healthy instance: %w", err)
	}

	// 将卡牌数组转换为字符串
	cardsStr := fmt.Sprintf("%v", task.Cards)

	// 构建请求体
	requestBody := map[string]interface{}{
		"inputs": map[string]interface{}{
			"question": task.Question,
			"cards":    cardsStr, // 转换为字符串
		},
		"response_mode": "blocking",
		"user":          task.ID,
	}

	// 使用选定的实例执行任务
	result, err := instance.Client.R().
		SetContext(taskCtx).
		SetHeader("Authorization", "Bearer "+instance.APIKey).
		SetHeader("Content-Type", "application/json").
		SetBody(requestBody).
		Post(instance.URL + "/workflows/run")

	if err != nil {
		w.difyService.MarkInstanceUnhealthy(instance, err)
		return fmt.Errorf("failed to process task: %w", err)
	}

	// 更新任务状态和结果
	if err := w.queueService.UpdateTaskStatus(taskCtx, task.ID, TaskCompleted, result.String()); err != nil {
		return fmt.Errorf("failed to update task result: %w", err)
	}

	// 记录实例成功使用
	instance.LastUsed = time.Now()
	instance.RequestCount.AddRequest()

	return nil
}

// isFatalError 判断是否是致命错误
func isFatalError(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

// Stop 优雅关闭工作器组
func (w *Worker) Stop() {
	logger.InfoString("Worker", "Stop", "Stopping all workers...")

	// 发送停止信号
	w.cancel()

	// 等待所有工作器完成
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	// 设置超时
	select {
	case <-done:
		logger.InfoString("Worker", "Stop", "All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		logger.WarnString("Worker", "Stop", "Worker shutdown timed out")
	}
}
