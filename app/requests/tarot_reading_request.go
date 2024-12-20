package requests

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/thedevsaddam/govalidator"
	"tarot/app/models/reading"
)

type TarotReadingRequest struct {
	UserID   string `json:"user_id" valid:"required"`
	Question string `json:"question" valid:"required"`
	Cards    []int  `json:"cards" valid:"required"`
	Type     reading.ReadingType `json:"type" valid:"required"`
}

func ValidateTarotReading(c *gin.Context) (*TarotReadingRequest, error) {
	var req TarotReadingRequest
	
	// 1. 首先绑定 JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	
	// 2. 验证规则
	rules := govalidator.MapData{
		"user_id":  []string{"required"},
		"question": []string{"required", "min:1"},
		"cards":    []string{"required"},
		"type":     []string{"required", "in:free,premium"},
	}
	
	// 3. 验证消息
	messages := govalidator.MapData{
		"user_id": []string{
			"required:用户 ID 不能为空",
		},
		"question": []string{
			"required:问题不能为空",
			"min:问题长度不能小于 1 个字符",
		},
		"cards": []string{
			"required:卡牌不能为空",
		},
		"type": []string{
			"required:解读类型不能为空",
			"in:解读类型必须是 free 或 premium",
		},
	}
	
	// 4. 开始验证
	opts := govalidator.Options{
		Data:     &req,
		Rules:    rules,
		Messages: messages,
	}
	
	validator := govalidator.New(opts)
	if errs := validator.ValidateStruct(); len(errs) > 0 {
		// 将验证错误转换为字符串
		return nil, fmt.Errorf("验证失败: %v", errs)
	}
	
	// 5. 额外的卡牌验证
	if len(req.Cards) == 0 {
		return nil, fmt.Errorf("至少需要选择一张卡牌")
	}
	
	// 验证卡牌号码是否有效
	for _, cardID := range req.Cards {
		if cardID < 1 || cardID > 78 {
			return nil, fmt.Errorf("无效的卡牌编号: %d", cardID)
		}
	}
	
	return &req, nil
}
