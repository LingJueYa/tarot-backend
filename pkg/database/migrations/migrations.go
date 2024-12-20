package migrations

import (
	"tarot/app/models/user"
	"tarot/app/models/reading"
)

// RegisterTables 返回需要迁移的表的模型列表
func RegisterTables() []interface{} {
	return []interface{}{
		&user.User{},
		&reading.Reading{},
	}
} 