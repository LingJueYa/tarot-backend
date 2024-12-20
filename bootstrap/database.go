package bootstrap

import (
	"errors"
	"fmt"
	"tarot/pkg/config"
	"tarot/pkg/database"
	"tarot/pkg/database/migrations"
	"tarot/pkg/logger"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupDB 初始化数据库和 ORM
func SetupDB() {
	// 根据配置文件选择数据库类型
	var dbConfig gorm.Dialector
	switch config.Get("database.connection") {
	case "postgresql":
		dbConfig = setupPostgreSQL()
	case "sqlite":
		dbConfig = setupSQLite()
	default:
		panic(errors.New("暂不支持该数据库类型"))
	}

	// 连接数据库，并设置 GORM 的日志模式
	database.Connect(dbConfig, logger.NewGormLogger())

	// 设置连接池
	setupDBPool()

	// 自动迁移数据库结构
	if err := database.AutoMigrate(migrations.RegisterTables()); err != nil {
		logger.ErrorString("数据库", "自动迁移", "数据表结构迁移失败："+err.Error())
		return
	}
	logger.InfoString("数据库", "自动迁移", "数据表结构迁移成功")
}

// setupPostgreSQL 配置 PostgreSQL 连接
func setupPostgreSQL() gorm.Dialector {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		config.Get("database.postgresql.host"),
		config.Get("database.postgresql.port"),
		config.Get("database.postgresql.username"),
		config.Get("database.postgresql.password"),
		config.Get("database.postgresql.database"),
	)
	return postgres.New(postgres.Config{
		DSN: dsn,
	})
}

// setupSQLite 配置 SQLite 连接
func setupSQLite() gorm.Dialector {
	database := config.Get("database.sqlite.database")
	return sqlite.Open(database)
}

// setupDBPool 配置数据库连接池
func setupDBPool() {
	database.SQLDB.SetMaxOpenConns(config.GetInt("database.postgresql.max_open_connections"))
	database.SQLDB.SetMaxIdleConns(config.GetInt("database.postgresql.max_idle_connections"))
	database.SQLDB.SetConnMaxLifetime(time.Duration(config.GetInt("database.postgresql.max_life_seconds")) * time.Second)
}
