package config

import "tarot/pkg/config"

func init() {
	config.Add("queue", func() map[string]interface{} {
		return map[string]interface{}{
			"rate_limit":    config.Env("QUEUE_RATE_LIMIT", 12),
			"rate_burst":    config.Env("QUEUE_RATE_BURST", 50),
			"worker_count":  config.Env("QUEUE_WORKER_COUNT", 10),
			"metrics_size":  config.Env("QUEUE_METRICS_SIZE", 1000),
			"retry_times":   config.Env("QUEUE_RETRY_TIMES", 3),
			"retry_delay":   config.Env("QUEUE_RETRY_DELAY", 1),
			"pool_size":     config.Env("QUEUE_POOL_SIZE", 100),
			"min_idle":      config.Env("QUEUE_MIN_IDLE", 10),
		}
	})
} 