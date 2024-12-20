package bootstrap

import (
	"fmt"
	"strings"
	"time"

	"tarot/pkg/dify"
	"tarot/pkg/config"
	"tarot/pkg/logger"
)

// SetupDify 初始化 Dify 服务
func SetupDify() *dify.DifyService {
	logger.InfoString("Dify", "Setup", "正在初始化 Dify 服务...")

	// 获取配置
	urls := config.GetString("dify.urls")
	apiKeys := config.GetString("dify.api_keys")
	timeout := config.GetInt("dify.timeout")
	maxRetries := config.GetInt("dify.max_retries")

	// 记录当前配置值（用于调试）
	logger.DebugString("Dify", "Config", fmt.Sprintf(
		"当前配置: URLs=%s, APIKeys=%s, Timeout=%d, MaxRetries=%d",
		maskEmpty(urls),
		maskEmpty(apiKeys),
		timeout,
		maxRetries,
	))

	// 检查配置完整性
	if urls == "" {
		logger.ErrorString("Dify", "Config", "缺少必要的配置: DIFY_API_URLS 或 DIFY_URL 未设置")
		return nil
	}

	if apiKeys == "" {
		logger.ErrorString("Dify", "Config", "缺少必要的配置: DIFY_API_KEYS 或 DIFY_API_KEY 未设置")
		return nil
	}

	// 创建服务实例
	service := dify.NewDifyService(&dify.Config{
		URLs:       strings.Split(urls, ","),
			APIKeys:    strings.Split(apiKeys, ","),
			Timeout:    time.Duration(timeout) * time.Second,
			MaxRetries: maxRetries,
	})

	if service == nil {
		logger.ErrorString("Dify", "Setup", "Dify 服务初始化失败")
		return nil
	}

	logger.InfoString("Dify", "Setup", fmt.Sprintf(
		"Dify 服务初始化成功 [URLs: %d, APIKeys: %d]",
		len(strings.Split(urls, ",")),
		len(strings.Split(apiKeys, ",")),
	))
	return service
}

// maskEmpty 处理空字符串显示
func maskEmpty(s string) string {
	if s == "" {
		return "<空>"
	}
	return s
} 