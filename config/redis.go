package config

import (
	"tarot/pkg/config"
)

func init() {
	config.Add("redis", func() map[string]interface{} {
		return map[string]interface{}{
			"host":     config.Env("REDIS_HOST", "127.0.0.1"),
			"port":     config.Env("REDIS_PORT", "6379"),
			"password": config.Env("REDIS_PASSWORD", ""),

			// 业务类存储使用 1 号库（包括限流）
			"database": config.Env("REDIS_MAIN_DB", 1),

			// 队列专用 2 号库
			"queue_database": config.Env("REDIS_QUEUE_DB", 2),
			"queue_prefix":   config.Env("REDIS_QUEUE_PREFIX", "tarot:queue"),
			"queue_timeout":  config.Env("REDIS_QUEUE_TIMEOUT", 300),
		}
	})
}
