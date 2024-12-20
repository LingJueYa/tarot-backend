package types

import (
	"context"
	"tarot/app/models/payment"
	"time"
)

// Provider 支付提供商类型
type Provider string

const (
	ProviderWechat Provider = "wechat"
	ProviderAlipay Provider = "alipay"
)

// Status 支付状态
type Status string

const (
	StatusPending  Status = "pending"
	StatusPaid     Status = "paid"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
	StatusRefunded Status = "refunded"
)

// Request 支付请求参数
type Request struct {
	UserID      string   `json:"user_id"`
	ReadingID   uint64   `json:"reading_id"`
	Amount      int64    `json:"amount"`
	Provider    Provider `json:"provider"`
	ReturnURL   string   `json:"return_url"`
	NotifyURL   string   `json:"notify_url"`
	Description string   `json:"description"`
}

// Result 支付结果
type Result struct {
	OrderNo     string                 `json:"order_no"`
	PaymentURL  string                 `json:"payment_url,omitempty"`
	PrepayID    string                 `json:"prepay_id,omitempty"`
	ExtraData   map[string]interface{} `json:"extra_data,omitempty"`
	ExpireAt    time.Time             `json:"expire_at"`
}

// Service 支付服务接口
type Service interface {
	CreatePayment(ctx context.Context, req *Request) (*Result, error)
	QueryPayment(ctx context.Context, orderNo string) (*payment.Payment, error)
	HandleNotify(ctx context.Context, data []byte) error
	CancelPayment(ctx context.Context, orderNo string) error
	RefundPayment(ctx context.Context, orderNo string, amount int64, reason string) error
}

// Repository 支付仓储接口
type Repository interface {
	Create(ctx context.Context, payment *payment.Payment) error
	Update(ctx context.Context, payment *payment.Payment) error
	GetByOrderNo(ctx context.Context, orderNo string) (*payment.Payment, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*payment.Payment, error)
} 