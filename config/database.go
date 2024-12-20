package config

import (
	"tarot/pkg/config"
)

func init() {
	config.Add("database", func() map[string]interface{} {
		return map[string]interface{}{
			// 默认连接
			"connection": config.Env("DB_CONNECTION", "postgresql"),

			// PostgreSQL 数据库配置
			"postgresql": map[string]interface{}{
				// 数据库连接信息
				"host":     config.Env("DB_HOST", "127.0.0.1"),
				"port":     config.Env("DB_PORT", "5432"),        // PostgreSQL 默认端口为 5432
				"database": config.Env("DB_DATABASE", "tarot"),
				"username": config.Env("DB_USERNAME", ""),
				"password": config.Env("DB_PASSWORD", ""),

				// 数据库连接池配置
				"max_idle_connections": config.Env("DB_MAX_IDLE_CONNECTIONS", 100),
				"max_open_connections": config.Env("DB_MAX_OPEN_CONNECTIONS", 25),
				"max_life_seconds":     config.Env("DB_MAX_LIFE_SECONDS", 5*60),
			},

			// SQLite 配置
			"sqlite": map[string]interface{}{
				"database": config.Env("DB_SQL_FILE", "database/database.db"),
			},
		}
	})
}
