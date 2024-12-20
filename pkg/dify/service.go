package dify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"

	"tarot/pkg/config"
	"tarot/pkg/logger"
)

// DifyService 实现了与 Dify API 的交互
// 支持多实例负载均衡、故障转移和自动恢复
type DifyService struct {
	instances  []*Instance   // Dify API 实例列表
	numRetries int           // 重试次数
	timeout    time.Duration // 请求超时时间
	mu         sync.RWMutex  // 保护实例状态的互斥锁
}

// Instance Dify 实例
type Instance struct {
	URL          string
	APIKey       string
	Health       bool
	Client       *resty.Client
	LastErr      error
	LastUsed     time.Time       // 记录最后一次成功使用时间
	ErrorCount   int             // 连续错误计数
	RequestCount *RequestCounter // 新增：请求计数器
}

// RequestCounter 请求计数器
type RequestCounter struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewRequestCounter 创建新的请求计数器
func NewRequestCounter() *RequestCounter {
	return &RequestCounter{
		requests: make([]time.Time, 0, 1000), // 预分配空间
	}
}

// AddRequest 记录新请求
func (rc *RequestCounter) AddRequest() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	// 清理超过1小时的旧记录
	for i, t := range rc.requests {
		if now.Sub(t) <= time.Hour {
			rc.requests = rc.requests[i:]
			break
		}
	}
	rc.requests = append(rc.requests, now)
}

// GetRecentCount 获取最近时间段内的请求数
func (rc *RequestCounter) GetRecentCount(duration time.Duration) int {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	count := 0
	for i := len(rc.requests) - 1; i >= 0; i-- {
		if now.Sub(rc.requests[i]) <= duration {
			count++
		} else {
			break
		}
	}
	return count
}

// GetConfig 获取配置切片
func GetConfig(key string) []string {
	value := config.GetString(key)
	if value == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

// NewDifyService 创建新的 Dify 服务实例
func NewDifyService(config *Config) *DifyService {
	if config == nil {
		return nil
	}

	if len(config.URLs) == 0 || len(config.APIKeys) == 0 {
		return nil
	}

	// 创建服务实例
	service := &DifyService{
		instances: make([]*Instance, 0, len(config.URLs)),
		timeout:   config.Timeout,
	}

	// 初始化所有实例
	for i := 0; i < len(config.URLs); i++ {
		url := config.URLs[i]
		apiKey := config.APIKeys[i]
		
		instance := NewInstance(url, apiKey, config.Timeout)
		if instance != nil {
			service.instances = append(service.instances, instance)
		}
	}

	// 检查是否有可用实例
	if len(service.instances) == 0 {
		return nil
	}

	return service
}

// GetInstances 获取所有实例列表
func (s *DifyService) GetInstances() []*Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 创建副本以避免外部修改
	instances := make([]*Instance, len(s.instances))
	copy(instances, s.instances)
	return instances
}

// GetHealthyInstanceCount 获取健康实例数量
func (s *DifyService) GetHealthyInstanceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, instance := range s.instances {
		if instance.Health {
			count++
		}
	}
	return count
}

// GetHealthyInstance 获取健康的 Dify 实例
func (s *DifyService) GetHealthyInstance() (*Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 使用简单轮询策略
	for _, instance := range s.instances {
		if instance.Health {
			return instance, nil
		}
	}

	return nil, errors.New("no healthy dify instance available")
}

// MarkInstanceUnhealthy 标记实例为不健康
func (s *DifyService) MarkInstanceUnhealthy(instance *Instance, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance.Health = false
	instance.LastErr = err
	logger.ErrorString("Dify", "Instance Unhealthy", fmt.Sprintf("URL: %s, Error: %v", instance.URL, err))
}

// ProcessTarotReading 处理塔罗牌解请求
func (s *DifyService) ProcessTarotReading(ctx context.Context, question string, cards []int) (string, error) {
	start := time.Now()
	var lastErr error

	for i := 0; i < s.numRetries; i++ {
		instance, err := s.getAvailableInstance()
		if err != nil {
			return "", fmt.Errorf("no available dify instance: %w", err)
		}

		// 记录请求开始
		logger.InfoString("Dify", "Request", fmt.Sprintf(
			"开始请求 实例:%s 问题:%s 卡牌:%v",
			shortenURL(instance.URL), question, cards))

		result, err := s.callDifyAPI(ctx, instance, question, cards)
		if err != nil {
			s.handleAPIError(instance, err)
			lastErr = err
			logger.ErrorString("Dify", "Error", fmt.Sprintf(
				"请求失败 实例:%s 错误:%v",
				shortenURL(instance.URL), err))
			continue
		}

		// 记录请求成功
		instance.RequestCount.AddRequest()
		duration := time.Since(start)
		logger.InfoString("Dify", "Success", fmt.Sprintf(
			"请求成功 实例:%s 耗时:%v 结果长度:%d",
			shortenURL(instance.URL), duration, len(result)))

		s.handleAPISuccess(instance)
		return result, nil
	}

	return "", fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// callDifyAPI 调用 Dify API
func (s *DifyService) callDifyAPI(ctx context.Context, instance *Instance, question string, cards []int) (string, error) {
	// 设置较长的超时时间
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	// 构建请求体
	reqBody := DifyRequest{
		Inputs: map[string]interface{}{
			"question": question,
			"cards":    formatCards(cards),
		},
		ResponseMode: "blocking",
		User:        "tarot-user",
	}

	// 发送请求前记录
	logger.InfoString("Dify", "Request", fmt.Sprintf(
		"开始请求 实例:%s URL:%s/v1/workflows/run",
		shortenURL(instance.URL), instance.URL))

	// 发送请求
	resp, err := instance.Client.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", instance.APIKey)).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post(fmt.Sprintf("%s/v1/workflows/run", instance.URL))

	if err != nil {
		logger.ErrorString("Dify", "Error", fmt.Sprintf(
			"请求失败 实例:%s 错误:%v", 
			shortenURL(instance.URL), err))
		return "", fmt.Errorf("failed to call dify api: %w", err)
	}

	// 记录响应结果
	logger.InfoString("Dify", "Response", fmt.Sprintf(
		"请求完成 实例:%s 状态:%d 响应长度:%d",
		shortenURL(instance.URL), resp.StatusCode(), len(resp.Body())))

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("dify api returned non-200 status: %d, body: %s",
			resp.StatusCode(), resp.String())
	}

	// 解析响应
	var difyResp DifyResponse
	if err := json.Unmarshal(resp.Body(), &difyResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal dify response: %w", err)
	}

	// 根据响应类型处理
	if difyResp.EventType == "message" {
		return difyResp.Answer, nil
	}

	return "", fmt.Errorf("unexpected response type: %s", difyResp.EventType)
}

// formatCards 格式化卡牌数组为字符串
func formatCards(cards []int) string {
	if len(cards) != 3 {
		return "" // 或者返回错误
	}
	cardStrs := make([]string, len(cards))
	for i, card := range cards {
		cardStrs[i] = fmt.Sprintf("%d", card)
	}
	return strings.Join(cardStrs, ",")
}

// HealthCheck 检查 Dify 服务健康状态
func (s *DifyService) HealthCheck(ctx context.Context) error {
	s.mu.RLock()
	instances := make([]*Instance, len(s.instances))
	copy(instances, s.instances)
	s.mu.RUnlock()

	// 检查是否有可用实例
	hasHealthy := false
	var lastErr error

	for _, instance := range instances {
		if instance.Health {
			hasHealthy = true
			break
		}
		if instance.LastErr != nil {
			lastErr = instance.LastErr
		}
	}

	if !hasHealthy {
		if lastErr != nil {
			return fmt.Errorf("no healthy dify instance available: %w", lastErr)
		}
		return errors.New("no healthy dify instance available")
	}

	return nil
}

// handleAPISuccess 处理 API 调用成功
func (s *DifyService) handleAPISuccess(instance *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance.Health = true
	instance.ErrorCount = 0
	instance.LastUsed = time.Now()
	instance.LastErr = nil
}

// handleAPIError 处理 API 调用错误
func (s *DifyService) handleAPIError(instance *Instance, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance.ErrorCount++
	instance.LastErr = err

	// 连续错误超过阈值才标记为不健康
	if instance.ErrorCount >= 3 {
		instance.Health = false
		logger.WarnString("Dify", "Instance", fmt.Sprintf(
			"实例 %s 被标记为不健康: 连续 %d 次错误, 最后错误: %v",
			instance.URL, instance.ErrorCount, err))
	}
}

// getAvailableInstance 获取可用的实例
func (s *DifyService) getAvailableInstance() (*Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var (
		selected *Instance
		minLoad  int
		statuses []string
	)

	// 记录当前有实例状态
	var healthyCount, totalCount int

	for i, instance := range s.instances {
		totalCount++
		if instance.Health {
			healthyCount++
			load := instance.RequestCount.GetRecentCount(5 * time.Minute)
			statuses = append(statuses, fmt.Sprintf(
				"实例#%d[%s] - 健康状态:✅ 最近负载:%d 上次使用:%s",
				i+1, shortenURL(instance.URL), load,
				formatDuration(instance.LastUsed)))

			if selected == nil || load < minLoad {
				selected = instance
				minLoad = load
			}
		} else {
			statuses = append(statuses, fmt.Sprintf(
				"实例#%d[%s] - 健康状态:❌ 错误计数:%d 最后错误:%v",
				i+1, shortenURL(instance.URL), instance.ErrorCount,
				instance.LastErr))
		}
	}

	// 记录负载均衡决策日志
	logger.InfoString("Dify", "LoadBalance", fmt.Sprintf(
		"实例状态统计 (健康:%d/总数:%d)\n%s",
		healthyCount, totalCount, strings.Join(statuses, "\n")))

	if selected != nil {
		logger.InfoString("Dify", "Selected", fmt.Sprintf(
			"选择实例 %s [负载:%d]", shortenURL(selected.URL), minLoad))
		return selected, nil
	}

	// 如果没有健康实例，重置所有实例状态
	if len(s.instances) > 0 {
		s.resetAllInstances()
		return s.instances[0], nil
	}

	return nil, errors.New("no dify instances available")
}

// resetAllInstances 重置所有实例状态
func (s *DifyService) resetAllInstances() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, instance := range s.instances {
		instance.Health = true
		instance.ErrorCount = 0
	}
	logger.InfoString("Dify", "Reset", "已重置所有实例状态")
}

// shortenURL 缩短 URL 用日志显示
func shortenURL(url string) string {
	if len(url) > 30 {
		return url[:15] + "..." + url[len(url)-12:]
	}
	return url
}

// 移除 humanize 包的引用，直接使用 time.Since 来格式化时间
func formatDuration(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return "刚刚"
	} else if duration < time.Hour {
		return fmt.Sprintf("%d分钟前", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(duration.Hours()))
	}
	return t.Format("01-02 15:04")
}

// NewInstance 创建新的 Dify 实例
func NewInstance(url string, apiKey string, timeout time.Duration) *Instance {
	if url == "" || apiKey == "" {
		return nil
	}

	client := resty.New().
		SetTimeout(timeout).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	return &Instance{
		URL:          url,
		APIKey:       apiKey,
		Health:       true,
		Client:       client,
		LastUsed:     time.Now(),
		ErrorCount:   0,
		RequestCount: NewRequestCounter(),
	}
}
