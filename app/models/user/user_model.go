// Package user 存放用户 Model 相关逻辑
package user

import (
	"tarot/app/models"
)

// User 用户模型
type User struct {
	ID        string `gorm:"primaryKey;type:varchar(36)"`
	Email     string `gorm:"unique;type:varchar(255)"`
	ClerkID   string `gorm:"unique;type:varchar(255);index"`
	Nickname  string `gorm:"type:varchar(50)"`
	AvatarURL string `gorm:"type:text"`
	Credits   int    `gorm:"default:0;index"`                     // 用户积分/次数
	GuestID   string `gorm:"type:varchar(36);index;default:null"` // 关联之前的游客ID

	models.CommonTimestampsField
}

// TableName 表名
func (User) TableName() string {
	return "users"
}
