package bootstrap

import (
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
	// 获取数据库连接类型
	dbConnection := config.Get("database.connection")
	logger.InfoString("数据库", "连接类型", fmt.Sprintf("使用 %s 数据库", dbConnection))

	// 根据配置文件选择数据库类型
	var dbConfig gorm.Dialector
	switch dbConnection {
	case "postgresql":
		dbConfig = setupPostgreSQL()
	case "sqlite":
		dbConfig = setupSQLite()
	default:
		panic(fmt.Errorf("不支持的数据库类型: %s", dbConnection))
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
	host := config.Get("database.postgresql.host")
	port := config.Get("database.postgresql.port")
	username := config.Get("database.postgresql.username")
	password := config.Get("database.postgresql.password")
	dbname := config.Get("database.postgresql.database")

	// 打印连接信息（不包含密码）
	logger.InfoString("数据库", "PostgreSQL", fmt.Sprintf("正在连接到 %s:%s/%s", host, port, dbname))

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		host, port, username, password, dbname)
	return postgres.New(postgres.Config{
		DSN: dsn,
	})
}

// setupSQLite 配置 SQLite 连接
func setupSQLite() gorm.Dialector {
	database := config.Get("database.sqlite.database")
	logger.InfoString("数据库", "SQLite", fmt.Sprintf("正在使用数据库文件: %s", database))
	return sqlite.Open(database)
}

// setupDBPool 配置数据库连接池
func setupDBPool() {
	maxOpenConns := config.GetInt("database.postgresql.max_open_connections")
	maxIdleConns := config.GetInt("database.postgresql.max_idle_connections")
	maxLifeSeconds := config.GetInt("database.postgresql.max_life_seconds")

	database.SQLDB.SetMaxOpenConns(maxOpenConns)
	database.SQLDB.SetMaxIdleConns(maxIdleConns)
	database.SQLDB.SetConnMaxLifetime(time.Duration(maxLifeSeconds) * time.Second)

	logger.InfoString("数据库", "连接池", fmt.Sprintf(
		"最大连接数: %d, 空闲连接数: %d, 连接最大生命周期: %d秒",
		maxOpenConns, maxIdleConns, maxLifeSeconds,
	))
}
