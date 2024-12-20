package routes

import (
	"tarot/app/http/controllers/api/v1/tarot"
	"tarot/app/http/middlewares"

	"github.com/gin-gonic/gin"
)

// 路由限流配置
const (
	// 🌍 全局限流：每小时每IP 30000 请求
	GlobalRateLimit = "30000-H"
	// 🎴 创建塔罗牌解读限流：每小时每IP 100 请求
	CreateReadingLimit = "100-H"
	// 🔍 查询结果限流：每分钟每IP 300 请求
	QueryResultLimit = "300-M"
)

// RegisterAPIRoutes 注册所有 API 路由
func RegisterAPIRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")

	v1.Use(
		middlewares.Recovery(),
		middlewares.SecurityHeaders(),
		middlewares.LimitIP(GlobalRateLimit),
		middlewares.Cors(),
	)

	// 🎴 塔罗牌相关路由
	tarotRoutes := v1.Group("/tarot")
	{
		rc := tarot.NewReadingController()

		// 📝 创建塔罗牌解读任务
		// POST /v1/tarot/readings
		// 请求频率：每小时每IP最多100次
		tarotRoutes.POST("/readings",
			middlewares.LimitIP(CreateReadingLimit),
			rc.Store,
		)

		// 📊 获取解读结果
		// GET /v1/tarot/readings/:id
		// 请求频率：每分钟每IP最多300次
		tarotRoutes.GET("/readings/:id",
			middlewares.LimitIP(QueryResultLimit),
			rc.GetResult,
		)

		// 📡 获取任务状态
		// GET /v1/tarot/readings/:id/status
		// 请求频率：每分钟每IP最多300次
		tarotRoutes.GET("/readings/:id/status",
			middlewares.LimitIP(QueryResultLimit),
			rc.GetStatus,
		)

		// 添加新的路由
		v1.GET("/users/:user_id/readings", rc.GetHistory)       // 获取历史记录
		v1.GET("/users/:user_id/readings/:task_id", rc.GetReadingDetail) // 获取单次结果

	}
}
