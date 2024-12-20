package requests

import (
	"github.com/gin-gonic/gin"
	"github.com/thedevsaddam/govalidator"
)

type TarotReadingRequest struct {
	UserID   string `json:"user_id" valid:"required"`
	Question string `json:"question" valid:"required,min:10,max:500"`
	Cards    []int  `json:"cards" valid:"required"`
}

func ValidateTarotReading(c *gin.Context) (TarotReadingRequest, error) {
	rules := govalidator.MapData{
		"user_id":  []string{"required", "uuid"},
		"question": []string{"required", "min:10", "max:500"},
		"cards":    []string{"required", "min:1", "max:3"},
	}

	messages := govalidator.MapData{
		"user_id": []string{
			"required:用户 ID 不能为空",
			"uuid:用户 ID 格式错误",
		},
		"question": []string{
			"required:问题不能为空",
			"min:问题最少 10 个字符",
			"max:问题最多 500 个字符",
		},
		"cards": []string{
			"required:卡牌不能为空",
			"min:至少选择 1 张卡牌",
			"max:最多选择 3 张卡牌",
		},
	}

	return ValidateRequest[TarotReadingRequest](c, rules, messages)
}
