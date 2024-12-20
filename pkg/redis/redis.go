/*
	Package redis 提供 Redis 连接和操作的工具包

	1. 连接池管理
	2. 自动重连
	3. 故障转移
	4. 性能优化
	5. 并发安全
*/
package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tarot/pkg/logger"

	redis "github.com/redis/go-redis/v9"
)

// 关键配置常量
const (
	// DefaultPoolSize Redis 连接池大小
	DefaultPoolSize = 100
	// DefaultTimeout 默认操作超时时间
	DefaultTimeout = 5 * time.Second
	// DefaultRetryTimes 重试次数
	DefaultRetryTimes = 3
	// DefaultMinIdleConns 最小空闲连接数
	DefaultMinIdleConns = 10
	// DefaultMaxRetries 最大重试次数
	DefaultMaxRetries = 3
	// DefaultIdleTimeout 空闲超时
	DefaultIdleTimeout = 5 * time.Minute
)

// RedisInstance Redis 实例类型
type RedisInstance string

const (
	MainDB   RedisInstance = "main"   // 主数据库实例（用于限流等）
	QueueDB  RedisInstance = "queue"  // 队列数据库实例
)

// RedisClient Redis 客户端封装
type RedisClient struct {
	Client  *redis.Client
	Context context.Context
	mutex   sync.RWMutex // 用于并发安全的操作
}

// RedisConfig Redis 配置结构
type RedisConfig struct {
	Address      string
	Username     string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	Timeout      time.Duration
}

type RedisManager struct {
	instances map[RedisInstance]*RedisClient
	mutex     sync.RWMutex
}

var (
	once     sync.Once
	Manager  *RedisManager
	Redis    *RedisClient  // 保持向后兼容
)

/* 🔄 连接管理相关方法 */

// ConnectRedis 初始化 Redis 连接
func ConnectRedis(address, username, password string, db int) {
	once.Do(func() {
		config := RedisConfig{
			Address:      address,
			Username:     username,
			Password:     password,
			DB:           db,
			PoolSize:     DefaultPoolSize,
			MinIdleConns: DefaultMinIdleConns,
			Timeout:      DefaultTimeout,
		}
		Redis = NewClient(config)
	})
}

// NewClient 创建新的 Redis 客户端
func NewClient(config RedisConfig) *RedisClient {
	rds := &RedisClient{
		Context: context.Background(),
	}

	// 优化的 Redis 客户端配置
	rds.Client = redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Username:     config.Username,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,     // 连接池大小
		MinIdleConns: config.MinIdleConns, // 最小空闲连接数
		
		// 连接池配置
		PoolTimeout:     config.Timeout,
		ConnMaxIdleTime: DefaultIdleTimeout,
		ConnMaxLifetime: 24 * time.Hour,
		
		// 读写超时
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		
		// 重试策略
		MaxRetries:      DefaultMaxRetries,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})

	// 测试连接
	if err := rds.Ping(); err != nil {
		panic(fmt.Sprintf("Redis 连接失败: %v", err))
	}

	return rds
}

/* 🔍 健康检查方法 */

// Ping 测试 Redis 连接
func (rds *RedisClient) Ping() error {
	ctx, cancel := context.WithTimeout(rds.Context, DefaultTimeout)
	defer cancel()

	_, err := rds.Client.Ping(ctx).Result()
	return err
}

/* 📝 数据操作方法 */

// Set 存储键值对
func (rds *RedisClient) Set(key string, value interface{}, expiration time.Duration) bool {
	ctx, cancel := context.WithTimeout(rds.Context, DefaultTimeout)
	defer cancel()

	rds.mutex.Lock()
	defer rds.mutex.Unlock()

	if err := rds.Client.Set(ctx, key, value, expiration).Err(); err != nil {
		logger.ErrorString("Redis", "Set", err.Error())
		return false
	}
	return true
}

// Get 获取键值
func (rds *RedisClient) Get(key string) string {
	ctx, cancel := context.WithTimeout(rds.Context, DefaultTimeout)
	defer cancel()

	rds.mutex.RLock()
	defer rds.mutex.RUnlock()

	result, err := rds.Client.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			logger.ErrorString("Redis", "Get", err.Error())
		}
		return ""
	}
	return result
}

// Has 检查键是否存在
func (rds *RedisClient) Has(key string) bool {
	ctx, cancel := context.WithTimeout(rds.Context, DefaultTimeout)
	defer cancel()

	rds.mutex.RLock()
	defer rds.mutex.RUnlock()

	n, err := rds.Client.Exists(ctx, key).Result()
	if err != nil {
		logger.ErrorString("Redis", "Has", err.Error())
		return false
	}
	return n > 0
}

// Del 删除键
func (rds *RedisClient) Del(keys ...string) bool {
	ctx, cancel := context.WithTimeout(rds.Context, DefaultTimeout)
	defer cancel()

	rds.mutex.Lock()
	defer rds.mutex.Unlock()

	if err := rds.Client.Del(ctx, keys...).Err(); err != nil {
		logger.ErrorString("Redis", "Del", err.Error())
		return false
	}
	return true
}

/* 🔢 计数器相关方法 */

// Increment 增加计数
func (rds *RedisClient) Increment(parameters ...interface{}) bool {
	ctx, cancel := context.WithTimeout(rds.Context, DefaultTimeout)
	defer cancel()

	rds.mutex.Lock()
	defer rds.mutex.Unlock()

	switch len(parameters) {
	case 1:
		key := parameters[0].(string)
		if err := rds.Client.Incr(ctx, key).Err(); err != nil {
			logger.ErrorString("Redis", "Increment", err.Error())
			return false
		}
	case 2:
		key := parameters[0].(string)
		value := parameters[1].(int64)
		if err := rds.Client.IncrBy(ctx, key, value).Err(); err != nil {
			logger.ErrorString("Redis", "Increment", err.Error())
			return false
		}
	default:
		logger.ErrorString("Redis", "Increment", "参数数量错误")
		return false
	}
	return true
}

// InitRedis 初始化 Redis 管理器
func InitRedis(address, username, password string, mainDB, queueDB int) {
	once.Do(func() {
		Manager = &RedisManager{
			instances: make(map[RedisInstance]*RedisClient),
		}

		// 初始化主数据库实例
		mainConfig := RedisConfig{
			Address:      address,
			Username:     username,
			Password:     password,
			DB:          mainDB,
			PoolSize:    DefaultPoolSize,
			MinIdleConns: DefaultMinIdleConns,
			Timeout:     DefaultTimeout,
		}
		Manager.instances[MainDB] = NewClient(mainConfig)

		// 初始化队列数据库实例
		queueConfig := RedisConfig{
			Address:      address,
			Username:     username,
			Password:     password,
			DB:          queueDB,
			PoolSize:    DefaultPoolSize,
			MinIdleConns: DefaultMinIdleConns,
			Timeout:     DefaultTimeout,
		}
		Manager.instances[QueueDB] = NewClient(queueConfig)

		// 保持向后兼容
		Redis = Manager.instances[MainDB]
	})
}

// GetRedis 获取指定的 Redis 实例
func GetRedis(instance RedisInstance) *RedisClient {
	Manager.mutex.RLock()
	defer Manager.mutex.RUnlock()
	
	if client, ok := Manager.instances[instance]; ok {
		return client
	}
	return Redis // 默认返回主实例
}
