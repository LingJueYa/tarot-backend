package reading

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// ReadingType 塔罗牌解读类型
type ReadingType string

const (
	TypeFree    ReadingType = "free"    // 免费解读
	TypePremium ReadingType = "premium"  // 付费解读
)

// Status 解读状态
type Status string

const (
	StatusPending    Status = "pending"    // 待解读
	StatusProcessing Status = "processing" // 解读中
	StatusCompleted  Status = "completed"  // 已完成
	StatusFailed     Status = "failed"     // 失败
)

// Cards 自定义类型用于处理卡牌数组的JSON序列化
type Cards []int

// Value 实现 driver.Valuer 接口
func (c Cards) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "[]", nil
	}
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner 接口
func (c *Cards) Scan(value interface{}) error {
	if value == nil {
		*c = Cards{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for cards")
	}

	return json.Unmarshal(bytes, c)
}

// Validate 验证记录
func (r *Reading) Validate() error {
	if r.UserID == "" {
		return errors.New("user_id is required")
	}
	if r.Type == "" {
		return errors.New("reading type is required")
	}
	if r.Type != TypeFree && r.Type != TypePremium {
		return errors.New("invalid reading type")
	}
	if len(r.Cards) == 0 {
		return errors.New("cards cannot be empty")
	}
	if len(r.Cards) > 3 {
		return errors.New("maximum 3 cards allowed")
	}
	return nil
}

// IsFree 检查是否为免费解读
func (r *Reading) IsFree() bool {
	return r.Type == TypeFree
}

// IsPremium 检查是否为付费解读
func (r *Reading) IsPremium() bool {
	return r.Type == TypePremium
}

// IsCompleted 检查是否已完成
func (r *Reading) IsCompleted() bool {
	return r.Status == string(StatusCompleted)
}

// IsPending 检查是否待解读
func (r *Reading) IsPending() bool {
	return r.Status == string(StatusPending)
}

// IsProcessing 检查是否解读中
func (r *Reading) IsProcessing() bool {
	return r.Status == string(StatusProcessing)
}

// IsFailed 检查是否失败
func (r *Reading) IsFailed() bool {
	return r.Status == string(StatusFailed)
} 