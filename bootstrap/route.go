package bootstrap

import (
	"tarot/app/http/middlewares"
	"tarot/routes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SetupRoute 路由初始化
// 该方法用于设置 Web 应用的路由配置，包括：
// 1. 注册全局中间件
// 2. 注册 API 路由
// 3. 配置 404 处理器
func SetupRoute(router *gin.Engine) {
	// 注册全局中间件
	registerGlobalMiddleWare(router)

	// 注册 API 路由
	// 具体路由定义在 routes 包中
	routes.RegisterAPIRoutes(router)

	// 配置 404 路由处理器
	setup404Handler(router)
}

// registerGlobalMiddleWare 注册全局中间件
// 设置应用级别的中间件，作用于所有请求
// - Logger 中间件：记录请求日志
// - Recovery 中间件：从 panic 中恢复
func registerGlobalMiddleWare(router *gin.Engine) {
	router.Use(
		middlewares.Logger(),    // 记录请求日志
		middlewares.Recovery(),  // 在发生 panic 时恢复
	)
}

// setup404Handler 配置 404 请求处理器
// 根据请求的 Accept 头来返回不同格式的 404 响应：
// - 当请求接受 HTML 时返回 HTML 格式的 404 页面
// - 其他情况返回 JSON 格式的错误信息
func setup404Handler(router *gin.Engine) {
	router.NoRoute(func(c *gin.Context) {
		// 获取请求头中的 Accept 信息
		acceptString := c.Request.Header.Get("Accept")
		
		// 根据 Accept 返回相应格式的响应
		if strings.Contains(acceptString, "text/html") {
			// 对于 HTML 请求返回简单的文本信息
			c.String(http.StatusNotFound, "页面返回 404")
		} else {
			// 默认返回 JSON 格式的错误信息
			c.JSON(http.StatusNotFound, gin.H{
				"error_code":    404,
				"error_message": "路由未定义，请确认 url 和请求方法是否正确。",
			})
		}
	})
}
