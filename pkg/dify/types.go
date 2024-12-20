package dify

// DifyRequest Dify API 请求结构
type DifyRequest struct {
	Inputs       map[string]string `json:"inputs"`
	ResponseMode string            `json:"response_mode"`
	User         string            `json:"user"`
}

// DifyResponse Dify API 响应结构
type DifyResponse struct {
	Data struct {
		Answer string `json:"answer"`  // 直接使用 answer 字段接收解读结果
	} `json:"data"`
} 