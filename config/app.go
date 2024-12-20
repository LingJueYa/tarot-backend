// Package config 站点配置信息
package config

import "tarot/pkg/config"

func init() {
	config.Add("app", func() map[string]interface{} {
		return map[string]interface{}{

			// 应用名称
			"name": config.Env("APP_NAME", "Tarot"),

			// 当前环境，用以区分多环境，一般为 local, stage, production, test
			"env": config.Env("APP_ENV", "production"),

			// 是否进入调试模式
			"debug": config.Env("APP_DEBUG", false),

			// 应用服务端口
			"port": config.Env("APP_PORT", "3000"),

			// 设置时区，日志记录里会使用到
			"timezone": config.Env("TIMEZONE", "Asia/Shanghai"),

			// 修改限流格式为每小时请求数
			"api_rate_limit": config.Env("API_RATE_LIMIT", "100"),  // 每小时100次
			"queue_rate_limit": config.Env("QUEUE_RATE_LIMIT", "30000"), // 每小时30000次
		}
	})
}
