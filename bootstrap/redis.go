package bootstrap

import (
	"fmt"
	"tarot/pkg/config"
	"tarot/pkg/redis"
	"tarot/pkg/logger"
)

// SetupRedis 初始化 Redis
func SetupRedis() {
	// 添加日志
	logger.InfoString("Redis", "Setup", fmt.Sprintf(
		"正在连接 Redis: %v:%v, DB: %v, QueueDB: %v",
		config.GetString("redis.host"),
		config.GetString("redis.port"),
		config.GetInt("redis.database"),
		config.GetInt("redis.queue_database"),
	))
	
	// 初始化 Redis 连接
	redis.InitRedis(
		fmt.Sprintf("%v:%v", config.GetString("redis.host"), config.GetString("redis.port")),
		config.GetString("redis.username"),
		config.GetString("redis.password"),
		config.GetInt("redis.database"),
		config.GetInt("redis.queue_database"),
	)
	
	// 测试连接
	mainRedis := redis.GetRedis(redis.MainDB)
	if err := mainRedis.Ping(); err != nil {
		logger.ErrorString("Redis", "MainDB", fmt.Sprintf("连接失败: %v", err))
		panic(err)
	}
	
	queueRedis := redis.GetRedis(redis.QueueDB)
	if err := queueRedis.Ping(); err != nil {
		logger.ErrorString("Redis", "QueueDB", fmt.Sprintf("连接失败: %v", err))
		panic(err)
	}
	
	logger.InfoString("Redis", "Setup", "Redis 连接成功")
}
