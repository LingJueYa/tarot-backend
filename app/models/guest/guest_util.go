// 游客模型操作函数
package guest

import (
	"errors"
	"fmt"
	"tarot/app/models/reading"
	"tarot/app/models/user"
	"tarot/pkg/database"
	"time"

	"gorm.io/gorm"
)

var (
	// ErrGuestNotFound 游客记录未找到
	ErrGuestNotFound = errors.New("guest not found")
)

// ReadingData 定义从前端接收的测算数据结构
type ReadingData struct {
	Type           reading.ReadingType `json:"type" binding:"required,oneof=free premium"`
	Question       string              `json:"question" binding:"required,min:10,max:500"`
	Cards          reading.Cards       `json:"cards" binding:"required,min=1,max=3"`
	Interpretation string              `json:"interpretation" binding:"required"`
}

// MigrateToUser 将游客数据迁移到注册用户账号
//
// 业务逻辑：
// 1. 无效的游客ID：
//   - 如果有用户ID和测算记录，则直接创建用户记录
//   - 如果缺少任一项，则静默返回
//
// 2. 无效的用户ID：静默返回
// 3. 空的测算记录：静默返回
//
// 参数:
//   - guestID: 游客UUID（可选）
//   - userID: 用户UUID
//   - readingData: 需要迁移的测算记录数组
//
// 返回:
//   - error: 仅在数据库操作失败时返回错误
func MigrateToUser(guestID string, userID string, readingData []ReadingData) error {
	// 1. 如果用户ID为空，静默返回
	if userID == "" {
		return nil
	}

	// 2. 如果测算记录为空，静默返回
	if len(readingData) == 0 {
		return nil
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 3. 如果提供了游客ID，则进行游客相关操作
		if guestID != "" {
			var guestExists int64
			if err := tx.Model(&Guest{}).
				Where("id = ? AND deleted_at IS NULL", guestID).
				Count(&guestExists).Error; err != nil {
				return fmt.Errorf("failed to check guest existence: %w", err)
			}

			// 如果游客存在，则进行关联和软删除
			if guestExists > 0 {
				// 更新用户表的 guest_id
				if err := tx.Model(&user.User{}).
					Where("id = ?", userID).
					Update("guest_id", guestID).Error; err != nil {
					return fmt.Errorf("failed to update user guest_id: %w", err)
				}

				// 软删除游客记录
				if err := tx.Model(&Guest{}).
					Where("id = ?", guestID).
					Update("deleted_at", time.Now().UTC()).Error; err != nil {
					return fmt.Errorf("failed to soft delete guest: %w", err)
				}
			}
		}

		// 4. 批量创建用户的测算记录
		readings := make([]reading.Reading, len(readingData))
		for i, data := range readingData {
			readings[i] = reading.Reading{
				UserID:         userID,
				Type:           data.Type,
				Question:       data.Question,
				Cards:          data.Cards,
				Interpretation: data.Interpretation,
				Status:         "completed",
			}
		}

		// 使用批量插入提高性能
		if err := tx.Table("tarot_readings").CreateInBatches(readings, 100).Error; err != nil {
			return fmt.Errorf("failed to create reading records: %w", err)
		}

		// 5. 更新用户的测算次数（使用原子操作）
		if err := tx.Model(&user.User{}).
			Where("id = ?", userID).
			Update("readings_count", gorm.Expr("readings_count + ?", len(readingData))).Error; err != nil {
			return fmt.Errorf("failed to update user readings count: %w", err)
		}

		return nil
	})
}
