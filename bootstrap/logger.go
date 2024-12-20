package bootstrap

import (
	"tarot/pkg/config"
	"tarot/pkg/logger"
)

// SetupLogger 初始化 Logger
// 该方法用于设置项目的日志系统，从配置文件中读取相关配置
// 参数说明：
// - filename: 日志文件路径
// - max_size: 每个日志文件保存的最大尺寸，单位：MB
// - max_backup: 日志文件最多保存多少个备份
// - max_age: 文件最多保存多少天
// - compress: 是否压缩归档的日志文件
// - type: 日志记录类型 可选：daily（按天）, single（单文件）
// - level: 日志级别，可选：debug, info, warn, error, fatal
func SetupLogger() {
	logger.InitLogger(
		config.GetString("log.filename"), // 日志文件路径
		config.GetInt("log.max_size"),    // 日志文件大小
		config.GetInt("log.max_backup"),  // 最多保存备份数
		config.GetInt("log.max_age"),     // 日志文件保存天数
		config.GetBool("log.compress"),   // 是否压缩
		config.GetString("log.type"),     // 日志记录类型
		config.GetString("log.level"),    // 日志级别
	)
}
