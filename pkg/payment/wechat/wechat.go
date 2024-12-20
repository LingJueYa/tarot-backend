package wechat

import (
	"context"
	"fmt"
	"time"
	
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	
	"tarot/app/models/payment"
	"tarot/config"
	"tarot/pkg/payment/types"
)

// WechatPayService 微信支付服务
type WechatPayService struct {
	client     *core.Client
	appID      string
	mchID      string
	notifyURL  string
	repository types.Repository
}

// NewWechatPayService 创建微信支付服务
func NewWechatPayService(config config.WechatConfig, repo types.Repository) (*WechatPayService, error) {
	// 1. 加载商户私钥
	mchPrivateKey, err := utils.LoadPrivateKey(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("load merchant private key error: %w", err)
	}
	
	// 2. 创建证书管理器
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(
			config.MchID,
			config.SerialNo,
			mchPrivateKey,
			config.APIv3Key,
		),
	}
	
	// 3. 创建客户端
	client, err := core.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create wechat pay client error: %w", err)
	}
	
	return &WechatPayService{
		client:     client,
		appID:      config.AppID,
		mchID:      config.MchID,
		
		notifyURL:  config.NotifyURL,
		repository: repo,
	}, nil
}

// CreatePayment 创建支付
func (s *WechatPayService) CreatePayment(ctx context.Context, req *types.Request) (*types.Result, error) {
	orderNo := GenerateOrderNo()
	expireAt := time.Now().Add(30 * time.Minute)
	
	p := &payment.Payment{
		OrderNo:   orderNo,
		UserID:    req.UserID,
		
		ReadingID: req.ReadingID,
		Provider:  string(types.ProviderWechat),
		Amount:    req.Amount,
		Status:    string(types.StatusPending),
		ExpireAt:  &expireAt,
	}
	
	if err := s.repository.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create payment record error: %w", err)
	}
	
	// 2. 调用微信支付API
	svc := jsapi.JsapiApiService{Client: s.client}
	prepayResp, result, err := svc.Prepay(ctx, jsapi.PrepayRequest{
		Appid:       core.String(s.appID),
		Mchid:       core.String(s.mchID),
		Description: core.String(req.Description),
		OutTradeNo:  core.String(orderNo),
		NotifyUrl:   core.String(s.notifyURL),
		Amount: &jsapi.Amount{
			Total:    core.Int64(req.Amount),
			Currency: core.String("CNY"),
		},
	})
	
	if err != nil {
		return nil, fmt.Errorf("create wechat payment error: %w", err)
	}
	
	if result != nil && result.Response.StatusCode != 200 {
		return nil, fmt.Errorf("create wechat payment failed with status code: %d", result.Response.StatusCode)
	}
	
	// 生成支付参数
	timestamp := time.Now().Unix()
	nonceStr := GenerateNonceStr()
	packageStr := fmt.Sprintf("prepay_id=%s", *prepayResp.PrepayId)
	
	// 计算签名
	paySign := CalculateWechatPaySign(s.appID, timestamp, nonceStr, packageStr)
	
	return &types.Result{
		OrderNo:   orderNo,
		PrepayID:  *prepayResp.PrepayId,
		ExtraData: map[string]interface{}{
			"appId":     s.appID,
			"timeStamp": timestamp,
			"nonceStr":  nonceStr,
			"package":   packageStr,
			"signType":  "RSA",
			"paySign":   paySign,
		},
		ExpireAt: expireAt,
	}, nil
}

// GenerateOrderNo 生成订单号
func GenerateOrderNo() string {
	return fmt.Sprintf("%d%06d", time.Now().Unix(), time.Now().Nanosecond()/1000)
}

// GenerateNonceStr 生成随机字符串
func GenerateNonceStr() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// CalculateWechatPaySign 计算微信支付签名
func CalculateWechatPaySign(appID string, timestamp int64, nonceStr, packageStr string) string {
	// 实现签名逻辑
	return ""
}

// 实现所有接口方法
func (s *WechatPayService) CancelPayment(ctx context.Context, orderNo string) error {
	// 实现取消支付逻辑
	return nil
}

func (s *WechatPayService) QueryPayment(ctx context.Context, orderNo string) (*payment.Payment, error) {
	return s.repository.GetByOrderNo(ctx, orderNo)
}

func (s *WechatPayService) HandleNotify(ctx context.Context, data []byte) error {
	// 实现支付通知处理逻辑
	return nil
}

func (s *WechatPayService) RefundPayment(ctx context.Context, orderNo string, amount int64, reason string) error {
	// 实现退款逻辑
	return nil
} 