package bootstrap

import (
	"fmt"
	"tarot/pkg/config"
	"tarot/pkg/redis"
)

// SetupRedis 初始化 Redis
func SetupRedis() {
	// 初始化 Redis 连接
	redis.InitRedis(
		fmt.Sprintf("%v:%v", config.GetString("redis.host"), config.GetString("redis.port")),
		config.GetString("redis.username"),
		config.GetString("redis.password"),
		config.GetInt("redis.database"),
		config.GetInt("redis.queue_database"),
	)
}
