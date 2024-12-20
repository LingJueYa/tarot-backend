package payment

import (
	"github.com/gin-gonic/gin"

	"tarot/pkg/payment"
	"tarot/pkg/response"
)

type PaymentController struct {
	paymentService payment.Service
}

// NewPaymentController 创建支付控制器
func NewPaymentController(service payment.Service) *PaymentController {
	return &PaymentController{
		paymentService: service,
	}
}

// CreatePayment 创建支付
func (pc *PaymentController) CreatePayment(c *gin.Context) {
	var req struct {
		ReadingID uint64           `json:"reading_id" binding:"required"`
		Provider  payment.Provider `json:"provider" binding:"required,oneof=wechat alipay"`
		ReturnURL string           `json:"return_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err, "invalid request")
		return
	}

	// 获取用户ID
	userID := c.GetString("user_id")

	// 创建支付请求
	payReq := &payment.Request{
		UserID:      userID,
		ReadingID:   req.ReadingID,
		Amount:      2000, // 20元
		Provider:    req.Provider,
		ReturnURL:   req.ReturnURL,
		Description: "塔罗牌解读服务",
	}

	// 创建支付
	result, err := pc.paymentService.CreatePayment(c.Request.Context(), payReq)
	if err != nil {
		response.Abort500(c, "create payment failed")
		return
	}

	response.Data(c, result)
}
