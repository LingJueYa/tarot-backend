package alipay

import (
	"context"
	"fmt"
	"time"
	
	"github.com/smartwalle/alipay/v3"
	
	"tarot/app/models/payment"
	"tarot/config"
	"tarot/pkg/payment/types"
)

// AlipayService 支付宝支付服务
type AlipayService struct {
	client     *alipay.Client
	appID      string
	notifyURL  string
	returnURL  string
	repository types.Repository
}

// NewAlipayService 创建支付宝支付服务
func NewAlipayService(config config.AlipayConfig, repo types.Repository) (*AlipayService, error) {
	client, err := alipay.New(config.AppID, config.PrivateKey, config.IsProduction)
	if err != nil {
		return nil, fmt.Errorf("create alipay client error: %w", err)
	}
	
	if err := client.LoadAliPayPublicKey(config.PublicKey); err != nil {
		return nil, fmt.Errorf("load alipay public key error: %w", err)
	}
	
	return &AlipayService{
		client:     client,
		appID:      config.AppID,
		notifyURL:  config.NotifyURL,
		returnURL:  config.ReturnURL,
		repository: repo,
	}, nil
}

// CreatePayment 创建支付
func (s *AlipayService) CreatePayment(ctx context.Context, req *types.Request) (*types.Result, error) {
	orderNo := GenerateOrderNo()
	expireAt := time.Now().Add(30 * time.Minute)
	
	p := &payment.Payment{
		OrderNo:   orderNo,
			UserID:    req.UserID,
			ReadingID: req.ReadingID,
			Provider:  string(types.ProviderAlipay),
			Amount:    req.Amount,
			Status:    string(types.StatusPending),
			ExpireAt:  &expireAt,
	}
	
	if err := s.repository.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create payment record error: %w", err)
	}
	
	trade := alipay.TradePagePay{}
	trade.NotifyURL = s.notifyURL
	trade.ReturnURL = req.ReturnURL
	trade.Subject = req.Description
	trade.OutTradeNo = orderNo
	trade.TotalAmount = fmt.Sprintf("%.2f", float64(req.Amount)/100)
	trade.ProductCode = "FAST_INSTANT_TRADE_PAY"
	
	url, err := s.client.TradePagePay(trade)
	if err != nil {
		return nil, fmt.Errorf("create alipay payment error: %w", err)
	}
	
	return &types.Result{
		OrderNo:    orderNo,
		PaymentURL: url.String(),
		ExpireAt:   expireAt,
	}, nil
}

// GenerateOrderNo 生成订单号
func GenerateOrderNo() string {
	return fmt.Sprintf("%d%06d", time.Now().Unix(), time.Now().Nanosecond()/1000)
}

// 实现 Service 接口的所有方法
func (s *AlipayService) CancelPayment(ctx context.Context, orderNo string) error {
	// 实现取消支付逻辑
	return nil
}

func (s *AlipayService) QueryPayment(ctx context.Context, orderNo string) (*payment.Payment, error) {
	return s.repository.GetByOrderNo(ctx, orderNo)
}

func (s *AlipayService) HandleNotify(ctx context.Context, data []byte) error {
	// 实现支付通知处理逻辑
	return nil
}

func (s *AlipayService) RefundPayment(ctx context.Context, orderNo string, amount int64, reason string) error {
	// 实现退款逻辑
	return nil
} 