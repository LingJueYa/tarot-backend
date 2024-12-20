package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"tarot/bootstrap"
	btsConfig "tarot/config"
	"tarot/pkg/config"
	"time"

	"github.com/gin-gonic/gin"
)

// 加载应用程序的基础配置
func init() {
	// 加载 config 目录下的配置信息
	btsConfig.Initialize()
}

// 应用程序上下文，用于优雅关闭
type App struct {
	server *http.Server
}

func main() {
	// 解析命令行参数
	env := parseFlags()

	// 初始化应用配置
	if err := setupApplication(env); err != nil {
		log.Fatalf("初始化应用程序失败: %v", err)
	}

	// 创建并配置 Gin 服务器
	router := setupServer()

	// 创建应用实例
	app := &App{
		server: &http.Server{
			Addr:    ":" + config.Get("app.port"),
			Handler: router,
		},
	}

	// 启动服务器（包含优雅关闭）
	app.start()
}

// parseFlags 解析命令行参数
// 返回环境配置参数
func parseFlags() string {
	var env string
	flag.StringVar(&env, "env", "", "加载 .env 文件，例如 --env=testing 将加载 .env.testing 文件")
	flag.Parse()
	return env
}

// setupApplication 初始化应用程序所需的各种组件
func setupApplication(env string) error {
	// 先初始化配置
	config.InitConfig(env)

	// 然后初始化日志
	bootstrap.SetupLogger()

	// 初始化数据库
	bootstrap.SetupDB()

	// 初始化 Redis
	bootstrap.SetupRedis()

	// 初始化队列服务
	bootstrap.SetupQueue()

	// 初始化 Dify 服务
	difyService := bootstrap.SetupDify()
	if difyService == nil {
		log.Println("Dify 服务初始化失败，请检查配置")
		return nil
	}

	return nil
}

// setupServer 配置并返回 Gin 服务器实例
func setupServer() *gin.Engine {
	// 设置 gin 为生产模式
	// 这样可以减少不必要的日志输出，提高性能
	gin.SetMode(gin.ReleaseMode)

	// 创建一个新的 Gin 引擎实例
	router := gin.New()

	// 设置路由
	bootstrap.SetupRoute(router)

	return router
}

// start 启动服务器并处理优雅关闭
func (a *App) start() {
	// 创建系统信号监听器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("服务器正在启动，监听端口 %s\n", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	<-quit
	log.Println("正在关闭服务器...")

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := a.server.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭异常: %v", err)
	}

	log.Println("服务器已成功关闭")
}
