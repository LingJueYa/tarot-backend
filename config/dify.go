package config

import (
	"strings"
	"tarot/pkg/config"
)

func init() {
	config.Add("dify", func() map[string]interface{} {
		return map[string]interface{}{
			"instances":   config.Env("DIFY_INSTANCES", 1),
			"urls":        strings.Split(config.Env("DIFY_API_URLS", "").(string), ","),
			"api_keys":    strings.Split(config.Env("DIFY_API_KEYS", "").(string), ","),
			"timeout":     config.Env("DIFY_TIMEOUT", 30),
			"max_retries": config.Env("DIFY_MAX_RETRIES", 3),
		}
	})
} 