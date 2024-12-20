package payment

import (
	"time"
)

// Payment 支付记录模型
type Payment struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderNo       string         `gorm:"type:varchar(64);uniqueIndex" json:"order_no"`     
	UserID        string         `gorm:"type:varchar(36);index" json:"user_id"`           
	ReadingID     uint64         `gorm:"index" json:"reading_id"`                         
	Provider      string         `gorm:"type:varchar(20)" json:"provider"`                
	Amount        int64          `gorm:"" json:"amount"`                                  
	Status        string         `gorm:"type:varchar(20);index" json:"status"`           
	TransactionID string         `gorm:"type:varchar(64)" json:"transaction_id"`          
	PayAt         *time.Time     `gorm:"" json:"pay_at"`                                 
	ExpireAt      *time.Time     `gorm:"" json:"expire_at"`                             
	ExtraData     JSON           `gorm:"type:json" json:"extra_data"`                    
	CreatedAt     time.Time      `gorm:"" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"" json:"updated_at"`
}

// TableName 指定表名
func (Payment) TableName() string {
	return "payments"
} 