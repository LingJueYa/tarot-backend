// Package requests 处理请求数据和表单验证
package requests

import (
	"fmt"
	"net/url"
	
	"github.com/gin-gonic/gin"
	"github.com/thedevsaddam/govalidator"
)

// ValidationError 自定义验证错误
type ValidationError struct {
	Errors url.Values
}

// Error 实现 error 接口
func (v ValidationError) Error() string {
	return fmt.Sprintf("验证错误: %v", v.Errors)
}

// ValidatorFunc 验证函数类型
type ValidatorFunc func(interface{}, *gin.Context) map[string][]string

// ValidateStruct 通用的结构体验证函数
func ValidateStruct(data interface{}, rules govalidator.MapData, messages govalidator.MapData) error {
	opts := govalidator.Options{
		Data:          data,
		Rules:         rules,
		TagIdentifier: "valid", // 模型中的 Struct 标签标识符
		Messages:      messages,
	}
	
	if errs := govalidator.New(opts).ValidateStruct(); len(errs) > 0 {
		return ValidationError{Errors: errs}
	}
	
	return nil
}

// ValidateRequest 通用的请求验证函数
func ValidateRequest[T any](c *gin.Context, rules govalidator.MapData, messages govalidator.MapData) (T, error) {
	var req T
	
	// 1. 解析请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		var zero T
		return zero, fmt.Errorf("解析请求失败: %w", err)
	}
	
	// 2. 验证结构体
	if err := ValidateStruct(req, rules, messages); err != nil {
		var zero T
		return zero, err
	}
	
	return req, nil
}
