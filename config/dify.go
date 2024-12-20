package config

import (
	"tarot/pkg/config"
)

func init() {
	config.Add("dify", func() map[string]interface{} {
		urls := config.Env("DIFY_API_URLS", "")
		if urls == "" {
			urls = config.Env("DIFY_URL", "")
		}

		apiKeys := config.Env("DIFY_API_KEYS", "")
		if apiKeys == "" {
			apiKeys = config.Env("DIFY_API_KEY", "")
		}

		return map[string]interface{}{
			"urls":        urls,
			"api_keys":    apiKeys,
			"timeout":     config.Env("DIFY_TIMEOUT", 90),
			"max_retries": config.Env("DIFY_MAX_RETRIES", 3),
		}
	})
} 