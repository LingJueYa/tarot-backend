// Package response 提供统一的 HTTP 响应处理

package response

import (
	"net/http"
	"tarot/pkg/logger"

	"github.com/gin-gonic/gin"
)

// 预定义响应状态
const (
	Success = "success" // 成功状态
	Error   = "error"   // 错误状态
)

/* 标准响应结构
{
    "status": "success",
    "data": {},     // 成功时返回的数据
    "error": "",    // 错误时返回的信息
    "message": "",  // 提示信息
}
*/

// Response 统一响应结构体
type Response struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ------------------ 🎯 成功响应系列 ------------------

// Data 响应 200 和数据
func Data(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Status: Success,
		Data:   data,
	})
}

// JSON 直接返回 JSON 数据
func JSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// Created 成功创建的响应
func Created(c *gin.Context, data interface{}, msg ...string) {
	message := "创建成功"
	if len(msg) > 0 {
		message = msg[0]
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

//  ------------------ 错误响应系列 ------------------

// Abort400 响应 400 错误
func Abort400(c *gin.Context, msg ...string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, Response{
		Status:  Error,
		Message: getMsg("请求参数错误", msg...),
	})
}

// Abort404 响应 404 错误
func Abort404(c *gin.Context, msg ...string) {
	c.AbortWithStatusJSON(http.StatusNotFound, Response{
		Status:  Error,
		Message: getMsg("资源不存在", msg...),
	})
}

// Abort500 响应 500 错误
func Abort500(c *gin.Context, msg ...string) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
		Status:  Error,
		Message: getMsg("服务器内部错误", msg...),
	})
}

// BadRequest 响应 400 错误（带错误信息）
func BadRequest(c *gin.Context, err error, msg ...string) {
	logger.LogIf(err)
	c.AbortWithStatusJSON(http.StatusBadRequest, Response{
		Status:  Error,
		Message: getMsg("请求格式错误", msg...),
		Error:   err.Error(),
	})
}

// ServerError 响应 500 错误（带错误信息）
func ServerError(c *gin.Context, err error, msg ...string) {
	logger.LogIf(err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
		Status:  Error,
		Message: getMsg("服务器内部错误", msg...),
		Error:   err.Error(),
	})
}

// ValidationError 响应 422 表单验证错误
func ValidationError(c *gin.Context, errors map[string][]string) {
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, Response{
		Status:  Error,
		Message: "表单验证失败",
		Data:    errors,
	})
}

// getMsg 获取消息内容
func getMsg(defaultMsg string, msg ...string) string {
	if len(msg) > 0 {
		return msg[0]
	}
	return defaultMsg
}
