package repositories

import (
	"context"
	"gorm.io/gorm"
	"tarot/app/models/reading"
	"tarot/pkg/database"
)

// ReadingRepository 塔罗牌阅读记录仓库
type ReadingRepository struct {
	db *gorm.DB
}

// NewReadingRepository 创建仓库实例
func NewReadingRepository() *ReadingRepository {
	return &ReadingRepository{
		db: database.DB,
	}
}

// Create 创建阅读记录
func (r *ReadingRepository) Create(ctx context.Context, reading *reading.Reading) error {
	return r.db.WithContext(ctx).Create(reading).Error
}

// GetByUserID 获取用户的历史记录
func (r *ReadingRepository) GetByUserID(ctx context.Context, userID string, page, pageSize int) ([]reading.Reading, int64, error) {
	var readings []reading.Reading
	var total int64
	
	// 使用预加载和索引优化查询
	query := r.db.WithContext(ctx).Model(&reading.Reading{}).Where("user_id = ?", userID)
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&readings).Error
	
	return readings, total, err
}

// GetByTaskID 获取单次测算结果
func (r *ReadingRepository) GetByTaskID(ctx context.Context, userID, taskID string) (*reading.Reading, error) {
	var reading reading.Reading
	
	// 使用复合条件确保安全性
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND task_id = ?", userID, taskID).
		First(&reading).Error
	
	if err != nil {
		return nil, err
	}
	
	return &reading, nil
} 