package factory

import (
	"fmt"
	
	"tarot/config"
	"tarot/pkg/payment/alipay"
	"tarot/pkg/payment/types"
	"tarot/pkg/payment/wechat"
)

// NewPaymentService 创建支付服务
func NewPaymentService(provider types.Provider, repo types.Repository, cfg interface{}) (types.Service, error) {
	switch provider {
	case types.ProviderWechat:
		wcfg, ok := cfg.(config.WechatConfig)
		if !ok {
			return nil, fmt.Errorf("invalid wechat config type")
		}
		return wechat.NewWechatPayService(wcfg, repo)
		
	case types.ProviderAlipay:
		acfg, ok := cfg.(config.AlipayConfig)
		if !ok {
			return nil, fmt.Errorf("invalid alipay config type")
		}
		return alipay.NewAlipayService(acfg, repo)
		
	default:
		return nil, fmt.Errorf("unsupported payment provider: %s", provider)
	}
} 