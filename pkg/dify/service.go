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
	instances  []*Instance    // Dify API 实例列表
	numRetries int           // 重试次数
	timeout    time.Duration // 请求超时时间
	mu         sync.RWMutex  // 保护实例状态的互斥锁
	
	// 新增字段用于健康检查
	healthCheckInterval time.Duration
	recoveryThreshold  time.Duration
}

// Instance Dify 实例
type Instance struct {
	URL     string
	APIKey  string
	Health  bool
	Client  *resty.Client
	LastErr error
}

// GetConfig 获取配置切片
func GetConfig(key string) []string {
	value := config.GetString(key)
	if value == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

// NewDifyService 创建并初始化 DifyService
// 包含自动的健康检查和实例恢复机制
func NewDifyService() *DifyService {
	urls := GetConfig("dify.urls")
	apiKeys := GetConfig("dify.api_keys")
	timeout := time.Duration(config.GetInt("dify.timeout")) * time.Second

	if len(urls) != len(apiKeys) {
		logger.ErrorString("Dify", "Config", "URLs and API keys count mismatch")
		return nil
	}

	instances := make([]*Instance, len(urls))
	for i := range urls {
		client := resty.New().
			SetTimeout(timeout).
			SetRetryCount(3).
			SetRetryWaitTime(1 * time.Second).
			SetRetryAfter(func(client *resty.Client, resp *resty.Response) (time.Duration, error) {
				// 智能退避策略
				return time.Duration(resp.Request.Attempt) * time.Second, nil
			})

		instances[i] = &Instance{
			URL:     urls[i],
			APIKey:  apiKeys[i],
			Health:  true,
			Client:  client,
		}
	}

	service := &DifyService{
		instances:           instances,
		numRetries:         config.GetInt("dify.max_retries"),
		timeout:            timeout,
		healthCheckInterval: time.Duration(config.GetInt("dify.health_check_interval", 30)) * time.Second,
		recoveryThreshold:  time.Duration(config.GetInt("dify.recovery_threshold", 300)) * time.Second,
	}

	// 启动健康检查
	go service.startHealthCheck()

	return service
}

// startHealthCheck 定期检查不健康实例的状态
func (s *DifyService) startHealthCheck() {
	ticker := time.NewTicker(s.healthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		instances := make([]*Instance, len(s.instances))
		copy(instances, s.instances)
		s.mu.RUnlock()

		for _, instance := range instances {
			if !instance.Health {
				go s.checkInstanceHealth(instance)
			}
		}
	}
}

// checkInstanceHealth 检查单个实例的健康状态
func (s *DifyService) checkInstanceHealth(instance *Instance) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// 发送健康检查请求
	resp, err := instance.Client.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", instance.APIKey)).
		Get(fmt.Sprintf("%s/health", instance.URL))

	s.mu.Lock()
	defer s.mu.Unlock()

	if err == nil && resp.StatusCode() == 200 {
		instance.Health = true
		instance.LastErr = nil
		logger.InfoString("Dify", "Instance Recovery", fmt.Sprintf("Instance %s recovered", instance.URL))
	}
}

// GetHealthyInstance 获取健康的 Dify 实例
func (s *DifyService) GetHealthyInstance() (*Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 使用简单的轮询策略
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

// ProcessTarotReading 处理塔罗牌解读请求
func (s *DifyService) ProcessTarotReading(ctx context.Context, question string, cards []int) (string, error) {
	var lastErr error

	for i := 0; i < s.numRetries; i++ {
		instance, err := s.GetHealthyInstance()
		if err != nil {
			return "", fmt.Errorf("no available dify instance: %w", err)
		}

		result, err := s.callDifyAPI(ctx, instance, question, cards)
		if err != nil {
			s.MarkInstanceUnhealthy(instance, err)
			lastErr = err
			continue
		}

		return result, nil
	}

	return "", fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// callDifyAPI 调用 Dify API
func (s *DifyService) callDifyAPI(ctx context.Context, instance *Instance, question string, cards []int) (string, error) {
	// 构建请求体
	reqBody := DifyRequest{
		Inputs: map[string]string{
			"question": question,
			"cards":    formatCards(cards),
		},
		ResponseMode: "blocking",
		User:        "tarot-user",
	}

	// 发送请求
	resp, err := instance.Client.R().
		SetContext(ctx).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", instance.APIKey)).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post(fmt.Sprintf("%s/chat/completions", instance.URL))

	if err != nil {
		return "", fmt.Errorf("failed to call dify api: %w", err)
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("dify api returned non-200 status: %d, body: %s", resp.StatusCode(), resp.String())
	}

	// 解析响应
	var difyResp DifyResponse
	if err := json.Unmarshal(resp.Body(), &difyResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal dify response: %w", err)
	}

	return difyResp.Data.Answer, nil
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