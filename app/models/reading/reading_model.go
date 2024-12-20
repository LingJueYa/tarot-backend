// 塔罗牌阅读记录
package reading

import (
	"tarot/app/models"
)

// Reading 塔罗牌阅读记录模型
type Reading struct {
	ID             uint64      `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID         string      `gorm:"type:varchar(36);uniqueIndex" json:"task_id"`      // 任务ID，唯一索引
	UserID         string      `gorm:"type:varchar(36);index" json:"user_id"`            // 用户ID，普通索引
	Type           ReadingType `gorm:"type:varchar(20);index" json:"type"`               // 解读类型（免费/付费）
	Question       string      `gorm:"type:text" json:"question"`                        // 问题
	Cards          Cards       `gorm:"type:json" json:"cards"`                          // 卡牌数组
	Interpretation string      `gorm:"type:text" json:"interpretation"`                  // 解读结果
	Status         string      `gorm:"type:varchar(20);index" json:"status"`            // 状态
	
	models.CommonTimestampsField // 包含 created_at 和 updated_at
}

// TableName 指定表名
func (Reading) TableName() string {
	return "tarot_readings"
}

// BeforeSave GORM 钩子
func (r *Reading) BeforeSave() error {
	return r.Validate()
}
