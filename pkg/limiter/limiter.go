// Package limiter 处理限流逻辑
package limiter

import (
	"fmt"
	"strconv"
	"strings"
	"tarot/pkg/config"
	"tarot/pkg/logger"
	"tarot/pkg/redis"

	"github.com/gin-gonic/gin"
	limiterlib "github.com/ulule/limiter/v3"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

// Rate 定义限流速率
type Rate struct {
	Rate float64
}

// ParseLimit 解析限流配置字符串
// 支持的格式: "5-S"、"10-M"、"1000-H"、"2000-D"
func ParseLimit(limit string) (*Rate, error) {
	// 将 "5-S" 格式转换为 "5/S" 格式
	formatted := strings.ReplaceAll(limit, "-", "/")

	// 使用 limiterlib 解析
	_, err := limiterlib.NewRateFromFormatted(formatted)
	if err != nil {
		return nil, fmt.Errorf("invalid limit format: %w", err)
	}

	// 获取数值部分
	parts := strings.Split(limit, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid limit format: %s", limit)
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid rate value: %s", parts[0])
	}

	// 根据时间单位转换为每秒的速率
	var ratePerSecond float64
	switch strings.ToUpper(parts[1]) {
	case "S":
		ratePerSecond = value
	case "M":
		ratePerSecond = value / 60.0
	case "H":
		ratePerSecond = value / 3600.0
	case "D":
		ratePerSecond = value / 86400.0
	default:
		return nil, fmt.Errorf("invalid time unit: %s", parts[1])
	}

	return &Rate{Rate: ratePerSecond}, nil
}

// GetKeyIP 获取 Limitor 的 Key，IP
func GetKeyIP(c *gin.Context) string {
	return c.ClientIP()
}

// GetKeyRouteWithIP Limitor 的 Key，路由+IP，针对单个路由做限流
func GetKeyRouteWithIP(c *gin.Context) string {
	return routeToKeyString(c.FullPath()) + c.ClientIP()
}

// CheckRate 检测请求是否超额
func CheckRate(c *gin.Context, key string, formatted string) (limiterlib.Context, error) {

	// 实例化依赖的 limiter 包的 limiter.Rate 对象
	var context limiterlib.Context
	rate, err := limiterlib.NewRateFromFormatted(formatted)
	if err != nil {
		logger.LogIf(err)
		return context, err
	}

	// 初始化存储，使用我们程序里共用的 redis.Redis 对象
	store, err := sredis.NewStoreWithOptions(redis.Redis.Client, limiterlib.StoreOptions{
		// 为 limiter 设置前缀，保持 redis 里数据的整洁
		Prefix: config.GetString("app.name") + ":limiter",
	})
	if err != nil {
		logger.LogIf(err)
		return context, err
	}

	// 使用上面的初始化的 limiter.Rate 对象和存储对象
	limiterObj := limiterlib.New(store, rate)

	// 获取限流的结果
	if c.GetBool("limiter-once") {
		// Peek() 取结果，不增加访问次数
		return limiterObj.Peek(c, key)
	} else {

		// 确保多个路由组里调用 LimitIP 进行限流时，只增加一次访问次数。
		c.Set("limiter-once", true)

		// Get() 取结果且增加访问次数
		return limiterObj.Get(c, key)
	}
}

// routeToKeyString 辅助方法，将 URL 中的 / 格式为 -
func routeToKeyString(routeName string) string {
	routeName = strings.ReplaceAll(routeName, "/", "-")
	routeName = strings.ReplaceAll(routeName, ":", "_")
	return routeName
}
