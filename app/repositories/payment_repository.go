package repositories

import (
	"context"
	"gorm.io/gorm"
	"tarot/app/models/payment"
	"tarot/pkg/database"
)

// PaymentRepository 支付记录仓库
type PaymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository 创建仓库实例
func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{
		db: database.DB,
	}
}

// Create 创建支付记录
func (r *PaymentRepository) Create(ctx context.Context, payment *payment.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

// Update 更新支付记录
func (r *PaymentRepository) Update(ctx context.Context, payment *payment.Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}

// GetByOrderNo 根据订单号获取支付记录
func (r *PaymentRepository) GetByOrderNo(ctx context.Context, orderNo string) (*payment.Payment, error) {
	var payment payment.Payment
	err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetByTransactionID 根据交易ID获取支付记录
func (r *PaymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*payment.Payment, error) {
	var payment payment.Payment
	err := r.db.WithContext(ctx).Where("transaction_id = ?", transactionID).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
} 