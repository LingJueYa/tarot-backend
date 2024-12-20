package dify

import "time"

// DifyRequest 请求结构体
type DifyRequest struct {
	Inputs        map[string]interface{} `json:"inputs"`        // 改为 interface{} 类型以支持更灵活的输入
	ResponseMode  string                 `json:"response_mode"` 
	User          string                 `json:"user"`
}

// DifyResponse 响应结构体
type DifyResponse struct {
	EventType string `json:"event"` // 事件类型
	Task      struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"task"`
	Answer string `json:"answer"` // 对于非流式响应
}

// Config Dify 服务配置
type Config struct {
	URLs       []string      // Dify 服务地址列表
	APIKeys    []string      // API 密钥列表
	Timeout    time.Duration // 请求超时时间
	MaxRetries int          // 最大重试次数
} 