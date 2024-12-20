// Package app 提供应用程序相关的辅助函数
package app

import (
	"tarot/pkg/config"
	"time"
)

// IsLocal 判断当前是否运行在本地环境
// 返回值：
// - true：当前是本地开发环境
// - false：当前不是本地环境
func IsLocal() bool {
	return config.Get("app.env") == "local"
}

// IsProduction 判断当前是否运行在生产环境
// 返回值：
// - true：当前是生产环境
// - false：当前不是生产环境
func IsProduction() bool {
	return config.Get("app.env") == "production"
}

// IsTesting 判断当前是否运行在测试环境
// 返回值：
// - true：当前是测试环境
// - false：当前不是测试环境
func IsTesting() bool {
	return config.Get("app.env") == "testing"
}

// TimenowInTimezone 获取当前时间（支持时区设置）
// 从配置文件读取 app.timezone 配置项来确定时区
// 返回值：
// - time.Time：返回对应时区的当前时间
// 使用示例：
// currentTime := TimenowInTimezone() // 获取配置的时区的当前时间
func TimenowInTimezone() time.Time {
	// 加载配置文件中指定的时区
	chinaTimezone, _ := time.LoadLocation(config.GetString("app.timezone"))
	// 返回指定时区的当前时间
	return time.Now().In(chinaTimezone)
}
