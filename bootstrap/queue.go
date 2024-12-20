package bootstrap

import (
	"strings"
	"time"

	"tarot/pkg/config"
	"tarot/pkg/dify"
	"tarot/pkg/queue"
	"tarot/pkg/logger"
	"tarot/pkg/redis"
)

func SetupQueue() {
	if redis.Manager == nil {
		logger.ErrorString("Queue", "Setup", "Redis manager not initialized")
		return
	}

	queueService := queue.NewQueueService()
	
	// 创建 Dify 配置
	difyConfig := &dify.Config{
		URLs:       strings.Split(config.GetString("dify.urls"), ","),
		APIKeys:    strings.Split(config.GetString("dify.api_keys"), ","),
		Timeout:    time.Duration(config.GetInt("dify.timeout")) * time.Second,
		MaxRetries: config.GetInt("dify.max_retries"),
	}
	
	difyService := dify.NewDifyService(difyConfig)
	if difyService == nil {
		logger.ErrorString("Queue", "Setup", "Dify service initialization failed")
		return
	}
	
	worker := queue.NewWorker(queueService, difyService, queue.WorkerConfig{
		WorkerCount:     config.GetInt("queue.worker_count", 10),
		MaxRetries:      config.GetInt("queue.retry_times", 3),
		RetryInterval:   time.Duration(config.GetInt("queue.retry_delay", 1)) * time.Second,
		ShutdownTimeout: 30 * time.Second,
		BatchSize:       10,
		MaxQueueSize:    10000,
	})
	
	go worker.Start()
	
	logger.InfoString("Queue", "Setup", "队列服务启动成功")
} 