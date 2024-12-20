// Package database 数据库操作
package database

import (
	"database/sql"
	"tarot/pkg/logger"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB 对象
var DB *gorm.DB
var SQLDB *sql.DB

// Connect 连接数据库
func Connect(dbConfig gorm.Dialector, _logger gormlogger.Interface) {
	// 使用 gorm.Open 连接数据库
	var err error
	DB, err = gorm.Open(dbConfig, &gorm.Config{
		Logger: _logger,
	})
	// 处理错误
	if err != nil {
		logger.ErrorString("数据库", "连接", err.Error())
		panic(err)
	}

	// 获取底层的 sqlDB
	SQLDB, err = DB.DB()
	if err != nil {
		logger.ErrorString("数据库", "获取底层SQL", err.Error())
		panic(err)
	}
}

// AutoMigrate 自动迁移所有数据表
func AutoMigrate(tables []interface{}) error {
	return DB.AutoMigrate(tables...)
}
