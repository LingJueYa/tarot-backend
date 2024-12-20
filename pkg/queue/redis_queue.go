package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	
	"tarot/pkg/config"
	"tarot/pkg/redis"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// TarotTask 塔罗牌解读任务
type TarotTask struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Question  string     `json:"question"`
	Cards     []int      `json:"cards"`
	Status    TaskStatus `json:"status"`
	Result    string     `json:"result"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// QueueService Redis 队列服务
// 支持高并发操作和可靠的任务处理
type QueueService struct {
	client      *redis.RedisClient
	prefix      string
	timeout     time.Duration
	rateLimiter *rate.Limiter
	metrics     *QueueMetrics
}

// NewQueueService 创建新的队列服务实例
func NewQueueService() *QueueService {
	rateLimit := config.GetInt("queue.rate_limit", 1000)
	burst := config.GetInt("queue.rate_burst", rateLimit)
	
	return &QueueService{
		client:      redis.GetRedis(redis.QueueDB),
		prefix:      config.GetString("redis.queue_prefix", "tarot"),
		timeout:     time.Duration(config.GetInt("redis.queue_timeout", 3600)) * time.Second,
		rateLimiter: rate.NewLimiter(rate.Limit(rateLimit), burst),
		metrics:     NewQueueMetrics(),
	}
}

// PushTask 将任务推送到队列
// 支持限流和监控指标收集
func (q *QueueService) PushTask(ctx context.Context, task *TarotTask) error {
	// 应用限流
	if err := q.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit exceeded: %w", err)
	}

	// 开始计时
	start := time.Now()
	defer func() {
		if q.metrics != nil {
			q.metrics.RecordPushLatency(time.Since(start))
		}
	}()

	// 序列化任务
	taskJSON, err := json.Marshal(task)
	if err != nil {
		q.metrics.RecordError(OpPush)
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// 使用事务确保原子性
	key := fmt.Sprintf("%s:tasks", q.prefix)
	statusKey := fmt.Sprintf("%s:status:%s", q.prefix, task.ID)

	pipe := q.client.Client.Pipeline()
	pipe.LPush(ctx, key, taskJSON)
	pipe.Set(ctx, statusKey, string(TaskPending), q.timeout)
	
	_, err = pipe.Exec(ctx)
	if err != nil {
		q.metrics.RecordError(OpPush)
		return fmt.Errorf("failed to push task: %w", err)
	}

	q.metrics.RecordSuccess(OpPush)
	return nil
}

// PopTask 从队列中获取任务
func (q *QueueService) PopTask(ctx context.Context) (*TarotTask, error) {
	key := fmt.Sprintf("%s:tasks", q.prefix)
	result, err := q.client.Client.BRPop(ctx, 0, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to pop task from queue: %w", err)
	}

	var task TarotTask
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// UpdateTaskStatus 更新任务状态
func (q *QueueService) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, result string) error {
	statusKey := fmt.Sprintf("%s:status:%s", q.prefix, taskID)
	if err := q.client.Client.Set(ctx, statusKey, string(status), q.timeout).Err(); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	if result != "" {
		resultKey := fmt.Sprintf("%s:result:%s", q.prefix, taskID)
		if err := q.client.Client.Set(ctx, resultKey, result, q.timeout).Err(); err != nil {
			return fmt.Errorf("failed to save task result: %w", err)
		}
	}

	return nil
}

// GetTaskResult 获取任务结果
func (q *QueueService) GetTaskResult(ctx context.Context, taskID string) (*TarotTask, error) {
	// 1. 获取任务状态
	statusKey := fmt.Sprintf("%s:status:%s", q.prefix, taskID)
	status, err := q.client.Client.Get(ctx, statusKey).Result()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil // 任务不存在
		}
		return nil, fmt.Errorf("failed to get task status: %w", err)
	}

	// 2. 获取任务结果
	resultKey := fmt.Sprintf("%s:result:%s", q.prefix, taskID)
	result, err := q.client.Client.Get(ctx, resultKey).Result()
	if err != nil && err != goredis.Nil {
		return nil, fmt.Errorf("failed to get task result: %w", err)
	}

	// 3. 构建任务对象
	task := &TarotTask{
		ID:     taskID,
		Status: TaskStatus(status),
		Result: result,
	}

	// 4. 如果任务未完成，返回 nil
	if task.Status != TaskCompleted {
		return nil, nil
	}

	return task, nil
}

// GetTaskStatus 获取任务状态
func (q *QueueService) GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error) {
	statusKey := fmt.Sprintf("%s:status:%s", q.prefix, taskID)
	status, err := q.client.Client.Get(ctx, statusKey).Result()
	if err != nil {
		if err == goredis.Nil {
			return "", nil // ���务不存在
		}
		return "", fmt.Errorf("failed to get task status: %w", err)
	}

	return TaskStatus(status), nil
}

// GetTaskProgress 获取任务进度信息
func (q *QueueService) GetTaskProgress(ctx context.Context, taskID string) (*TaskProgress, error) {
	// 1. 获取任务状态
	status, err := q.GetTaskStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 2. 构建进度信息
	progress := &TaskProgress{
		TaskID: taskID,
		Status: status,
	}

	// 3. 如果任务已完成，获取结果
	if status == TaskCompleted {
		resultKey := fmt.Sprintf("%s:result:%s", q.prefix, taskID)
		result, err := q.client.Client.Get(ctx, resultKey).Result()
		if err != nil && err != goredis.Nil {
			return nil, fmt.Errorf("failed to get task result: %w", err)
		}
		progress.Result = result
	}

	return progress, nil
}

// TaskProgress 任务进度信息
type TaskProgress struct {
	TaskID string     `json:"task_id"`
	Status TaskStatus `json:"status"`
	Result string     `json:"result,omitempty"`
}

// Ping 检查队列服务健康状态
func (q *QueueService) Ping(ctx context.Context) error {
	return q.client.Ping()
}

// DequeueTask 从队列中获取任务
func (q *QueueService) DequeueTask(ctx context.Context) (*TarotTask, error) {
	key := fmt.Sprintf("%s:tasks", q.prefix)
	
	// 使用 Client.BRPop 而不是直接使用 BRPop
	result, err := q.client.Client.BRPop(ctx, 0, key).Result()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}
		if err == context.DeadlineExceeded {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to pop task from queue: %v", err)
	}
	
	if len(result) != 2 {
		return nil, fmt.Errorf("invalid result from queue")
	}
	
	var task TarotTask
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %v", err)
	}
	
	return &task, nil
}
 