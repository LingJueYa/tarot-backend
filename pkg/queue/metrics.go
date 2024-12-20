package queue

import (
	"sync"
	"sync/atomic"
	"time"
)

// TaskID 任务ID的类型别名
type TaskID string

// MetricOperation 定义指标操作类型
type MetricOperation string

const (
	OpPush    MetricOperation = "push"
	OpPop     MetricOperation = "pop"
	OpProcess MetricOperation = "process"
)

// LatencyStats 延迟统计
type LatencyStats struct {
	count    atomic.Int64
	totalMs  atomic.Int64
	maxMs    atomic.Int64
	recentMs []int64 // 最近的延迟样本
}

// QueueMetrics 增强版性能指标收集器
type QueueMetrics struct {
	totalTasks      atomic.Int64
	successfulTasks atomic.Int64
	failedTasks     atomic.Int64
	processingTimes *sync.Map // 处理时间统计
	errorRates      *sync.Map // 错误率统计

	// 延迟统计
	pushLatency LatencyStats
	popLatency  LatencyStats

	// 队列状态
	queueLength     atomic.Int64
	avgWaitTime     atomic.Int64 // 平均等待时间(毫秒)
	peakQueueLength atomic.Int64

	// 等待时间计算
	waitTimeStart *sync.Map // map[TaskID]time.Time
}

// NewQueueMetrics 创建新的指标收集器
func NewQueueMetrics() *QueueMetrics {
	return &QueueMetrics{
		processingTimes: &sync.Map{},
		errorRates:      &sync.Map{},
		waitTimeStart:   &sync.Map{},
	}
}

// RecordSuccess 记录成功操作
func (m *QueueMetrics) RecordSuccess(op MetricOperation) {
	m.successfulTasks.Add(1)
	m.totalTasks.Add(1)
}

// RecordError 记录失败操作
func (m *QueueMetrics) RecordError(op MetricOperation) {
	m.failedTasks.Add(1)
	m.totalTasks.Add(1)
}

// StartWaitTime 记录任务开始等待的时间
func (m *QueueMetrics) StartWaitTime(taskID TaskID) {
	m.waitTimeStart.Store(taskID, time.Now())
}

// EndWaitTime 计算并更新平均等待时间
func (m *QueueMetrics) EndWaitTime(taskID TaskID) {
	if startTime, ok := m.waitTimeStart.LoadAndDelete(taskID); ok {
		waitDuration := time.Since(startTime.(time.Time))

		// 更新平均等待时间
		currentAvg := m.avgWaitTime.Load()
		totalTasks := m.totalTasks.Load()

		// 计算新的平均值
		newAvg := (currentAvg*totalTasks + waitDuration.Milliseconds()) / (totalTasks + 1)
		m.avgWaitTime.Store(newAvg)
	}
}

// RecordProcessingTime 记录任务处理时间
func (m *QueueMetrics) RecordProcessingTime(duration time.Duration) {
	m.processingTimes.Store(time.Now().Unix(), duration.Milliseconds())

	// 更新队列长度
	currentLength := m.queueLength.Load()
	if currentLength > m.peakQueueLength.Load() {
		m.peakQueueLength.Store(currentLength)
	}
}

// RecordPushLatency 记录推送延迟
func (m *QueueMetrics) RecordPushLatency(d time.Duration) {
	ms := d.Milliseconds()
	m.pushLatency.record(ms)
}

// RecordPopLatency 记录获取延迟
func (m *QueueMetrics) RecordPopLatency(d time.Duration) {
	ms := d.Milliseconds()
	m.popLatency.record(ms)
}

// record 记录延迟数据
func (s *LatencyStats) record(ms int64) {
	s.count.Add(1)
	s.totalMs.Add(ms)

	// 更新最大延迟
	for {
		current := s.maxMs.Load()
		if ms <= current {
			break
		}
		if s.maxMs.CompareAndSwap(current, ms) {
			break
		}
	}

	// 记录最近的样本
	idx := s.count.Load() % int64(len(s.recentMs))
	s.recentMs[idx] = ms
}
