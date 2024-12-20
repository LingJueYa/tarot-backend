// 游客模型
package guest

import (
	"tarot/app/models"
)

// Guest 游客模型
type Guest struct {
	ID           string `gorm:"primaryKey;type:varchar(36);index"` // UUID
	FreeReadings int    `gorm:"default:1"`                         // 免费测算次数，默认1次
	PaidReadings int    `gorm:"default:0"`                         // 付费测算次数，默认0次

	models.CommonTimestampsField
	models.SoftDeletes // 软删除
}

// TableName 表名
func (Guest) TableName() string {
	return "guests"
}
