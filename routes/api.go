package routes

import (
	"tarot/app/http/controllers/api/v1/tarot"
	"tarot/app/http/middlewares"

	"github.com/gin-gonic/gin"
)

// 路由限流配置
const (
	// 🌍 全局限流：每小时每IP 30000 请求
	GlobalLimit = "30000-h"
	// 🎴 创建塔罗牌解读限流：每小时每IP 100 请求
	ReadingLimit = "100-h"
	// 🔍 查询结果限流：每分钟每IP 300 请求
	QueryLimit = "300-m"
)

// RegisterAPIRoutes 注册所有 API 路由
func RegisterAPIRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")

	v1.Use(
		middlewares.Recovery(),
		middlewares.SecurityHeaders(),
		// TODO: 限流功能后续实现
		// middlewares.LimitIP(GlobalLimit),
		middlewares.Cors(),
	)

	// 🎴 塔罗牌相关路由
	tarotRoutes := v1.Group("/tarot")
	{
		rc := tarot.NewReadingController()

		// 📝 创建塔罗牌解读任务
		// POST /v1/tarot/readings
		// 请求频率：每小时每IP最多100次
		tarotRoutes.POST("/readings", rc.Store)

		// 📊 获取解读结果
		// GET /v1/tarot/readings/:id
		// 请求频率：每分钟每IP最多300次
		tarotRoutes.GET("/readings/:id", rc.GetResult)

		// 📡 获取任务状态
		// GET /v1/tarot/readings/:id/status
		// 请求频率：每分钟每IP最多300次
		tarotRoutes.GET("/readings/:id/status", rc.GetStatus)

		// 添加新的路由
		v1.GET("/users/:user_id/readings", rc.GetHistory)                // 获取历史记录
		v1.GET("/users/:user_id/readings/:task_id", rc.GetReadingDetail) // 获取单结果

		// 添加健康检查路由
		tarotRoutes.GET("/health/redis", rc.CheckRedisHealth)
	}
}
