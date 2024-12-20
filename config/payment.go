package config

// PaymentConfig 支付配置
type PaymentConfig struct {
	Wechat  WechatConfig
	Alipay  AlipayConfig
}

// WechatConfig 微信支付配置
type WechatConfig struct {
	AppID      string
	MchID      string
	SerialNo   string
	PrivateKey string
	APIv3Key   string
	NotifyURL  string
	ReturnURL  string
}

// AlipayConfig 支付宝配置
type AlipayConfig struct {
	AppID        string
	PrivateKey   string
	PublicKey    string
	NotifyURL    string
	ReturnURL    string
	IsProduction bool
} 