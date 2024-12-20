package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateOrderNo 生成订单号
func GenerateOrderNo() string {
	return fmt.Sprintf("%s%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%1000)
}

// GenerateNonceStr 生成随机字符串
func GenerateNonceStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CalculateWechatPaySign 计算微信支付签名
func CalculateWechatPaySign(appID string, timestamp int64, nonceStr, packageStr string) string {
	// 实现签名计算逻辑
	return "calculated_sign"
} 