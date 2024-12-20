package payment

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Provider 支付提供商类型
type Provider string

const (
	ProviderWechat Provider = "wechat" // 微信支付
	ProviderAlipay Provider = "alipay" // 支付宝
)

// Status 支付状态
type Status string

const (
	StatusPending  Status = "pending"  // 待支付
	StatusPaid     Status = "paid"     // 已支付
	StatusFailed   Status = "failed"   // 支付失败
	StatusCanceled Status = "canceled" // 已取消
	StatusRefunded Status = "refunded" // 已退款
)

// JSON 自定义JSON类型
type JSON map[string]interface{}

// Value 实现 driver.Valuer 接口
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSON)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid scan source")
	}
	return json.Unmarshal(bytes, j)
}

// Validate 验证支付记录
func (p *Payment) Validate() error {
	if p.UserID == "" {
		return errors.New("user_id is required")
	}
	if p.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if !p.ValidateProvider() {
		return errors.New("invalid payment provider")
	}
	return nil
}

// ValidateProvider 验证支付提供商
func (p *Payment) ValidateProvider() bool {
	return p.Provider == string(ProviderWechat) || p.Provider == string(ProviderAlipay)
}

// IsSuccess 检查支付是否成功
func (p *Payment) IsSuccess() bool {
	return p.Status == string(StatusPaid)
}

// IsPending 检查是否待支付
func (p *Payment) IsPending() bool {
	return p.Status == string(StatusPending)
}

// IsRefunded 检查是否已退款
func (p *Payment) IsRefunded() bool {
	return p.Status == string(StatusRefunded)
}

// IsCanceled 检查是否已取消
func (p *Payment) IsCanceled() bool {
	return p.Status == string(StatusCanceled)
}
