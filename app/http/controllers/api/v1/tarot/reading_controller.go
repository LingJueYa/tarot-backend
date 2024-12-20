package tarot

import (
	"log"
	"strconv"
	"time"
	"math/rand"
	"fmt"
	"strings"
	
	"github.com/gin-gonic/gin"
	
	"tarot/app/requests"
	"tarot/pkg/dify"
	"tarot/pkg/queue"
	"tarot/pkg/response"
	"tarot/app/repositories"
	"tarot/app/models/reading"
	"tarot/pkg/redis"
	"tarot/pkg/logger"
	"tarot/pkg/config"
)

type ReadingController struct {
	queueService *queue.QueueService
	difyService  *dify.DifyService
}

func NewReadingController() *ReadingController {
	// 创建 Dify 配置
	difyConfig := &dify.Config{
		URLs:       strings.Split(config.GetString("dify.urls"), ","),
		APIKeys:    strings.Split(config.GetString("dify.api_keys"), ","),
		Timeout:    time.Duration(config.GetInt("dify.timeout")) * time.Second,
		MaxRetries: config.GetInt("dify.max_retries"),
	}

	return &ReadingController{
		queueService: queue.NewQueueService(),
		difyService:  dify.NewDifyService(difyConfig),
	}
}

// Store 处理塔罗牌解读请求
func (rc *ReadingController) Store(c *gin.Context) {
	// 1. 验证请求
	request, err := requests.ValidateTarotReading(c)
	if err != nil {
		response.BadRequest(c, err, "请求验证失败")
		return
	}
	
	// 2. 生成唯一的 task_id
	taskID := generateTaskID()
	
	// 3. 创建塔罗牌阅读记录
	readingRecord := &reading.Reading{
		TaskID:   taskID,
		UserID:   request.UserID,
		Question: request.Question,
		Cards:    reading.Cards(request.Cards),
		Type:     request.Type,
		Status:   string(reading.StatusPending),
	}
	
	// 4. 保存到数据库
	if err := readingRecord.Create(); err != nil {
		log.Printf("创建塔罗牌阅读失败: %v", err)
		response.Abort500(c, "创建塔罗牌阅读失败")
		return
	}
	
	// 5. 创建队列任务
	task := &queue.TarotTask{
		ID:        taskID,
		UserID:    request.UserID,
		Question:  request.Question,
		Cards:     request.Cards,
		Status:    queue.TaskPending,
		CreatedAt: time.Now(),
	}
	
	// 6. 推送到队列
	if err := rc.queueService.PushTask(c.Request.Context(), task); err != nil {
		logger.ErrorString("Reading", "Queue", fmt.Sprintf("推送任务失败: %v", err))
		// 更新记录状态为错误
		readingRecord.Status = string(reading.StatusFailed)
		if updateErr := readingRecord.Save(); updateErr != nil {
			log.Printf("更新状态失败: %v", updateErr)
		}
		response.Abort500(c, "推送任务失败")
		return
	}
	
	response.Created(c, readingRecord, "塔罗牌阅读创建成功")
}

// generateTaskID 生成唯一的任务ID
func generateTaskID() string {
	// 格式: task_时间戳_随机数
	timestamp := time.Now().UnixNano() / 1e6 // 毫秒时间戳
	random := rand.Intn(10000)               // 随机数
	return fmt.Sprintf("task_%d_%04d", timestamp, random)
}

// GetResult 获取解读结果
func (rc *ReadingController) GetResult(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		response.Abort400(c, "缺少任务 ID")
		return
	}

	// 获取任务进度
	progress, err := rc.queueService.GetTaskProgress(c.Request.Context(), taskID)
	if err != nil {
		response.Abort500(c, "获取任务进度失败")
		return
	}

	if progress == nil {
		response.Abort404(c, "任务不存在")
		return
	}

	// 如果任务未完成，返回进度信息
	if progress.Status != queue.TaskCompleted {
		response.Data(c, gin.H{
			"task_id": taskID,
			"status":  progress.Status,
			"message": "任务处理中",
		})
		return
	}

	response.Data(c, gin.H{
		"task_id": taskID,
		"status":  progress.Status,
		"result":  progress.Result,
	})
}

// GetStatus 获取任务状态
func (rc *ReadingController) GetStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		response.Abort400(c, "缺少任务 ID")
		return
	}

	status, err := rc.queueService.GetTaskStatus(c.Request.Context(), taskID)
	if err != nil {
		response.Abort500(c, "获取任务状态失败")
		return
	}

	if status == "" {
		response.Abort404(c, "任务不存在")
		return
	}

	response.Data(c, gin.H{
		"task_id": taskID,
		"status":  status,
	})
}

// HealthCheck 健康检查端点
func (rc *ReadingController) HealthCheck(c *gin.Context) {
	// 检查 Redis 连接
	if err := rc.queueService.Ping(c.Request.Context()); err != nil {
		response.Abort500(c, "Queue service unavailable")
		return
	}

	// 检查 Dify 服务
	if err := rc.difyService.HealthCheck(c.Request.Context()); err != nil {
		response.Abort500(c, "Dify service unavailable")
		return
	}

	response.Data(c, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

// GetHistory 获取用户历史记录
func (rc *ReadingController) GetHistory(c *gin.Context) {
	// 获取分页参数
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "10")
	
	pageNum, _ := strconv.Atoi(page)
	size, _ := strconv.Atoi(pageSize)
	
	// 参数验证
	if pageNum < 1 {
		pageNum = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	
	userID := c.Param("user_id")
	if userID == "" {
		response.Abort400(c, "用户ID不能为空")
		return
	}
	
	// 获取历史记录
	repo := repositories.NewReadingRepository()
	readings, total, err := repo.GetByUserID(c.Request.Context(), userID, pageNum, size)
	if err != nil {
		response.Abort500(c, "获取历史记录失败")
		return
	}
	
	response.Data(c, gin.H{
		"data": readings,
		"meta": gin.H{
			"total":     total,
			"page":      pageNum,
			"page_size": size,
		},
	})
}

// GetReadingDetail 获取单次测算结果
func (rc *ReadingController) GetReadingDetail(c *gin.Context) {
	userID := c.Param("user_id")
	taskID := c.Param("task_id")
	
	if userID == "" || taskID == "" {
		response.Abort400(c, "参数不完整")
		return
	}
	
	// 获取测算结果
	repo := repositories.NewReadingRepository()
	reading, err := repo.GetByTaskID(c.Request.Context(), userID, taskID)
	if err != nil {
		response.Abort404(c, "记录不存在")
		return
	}
	
	response.Data(c, reading)
}

// CheckRedisHealth Redis 健康检查
func (rc *ReadingController) CheckRedisHealth(c *gin.Context) {
	// 检查主 Redis 实例
	mainRedis := redis.GetRedis(redis.MainDB)
	if err := mainRedis.Ping(); err != nil {
		response.JSON(c, gin.H{
			"status": "error",
			"main_db": "unavailable",
			"error": err.Error(),
		})
		return
	}
	
	// 检查队列 Redis 实例
	queueRedis := redis.GetRedis(redis.QueueDB)
	if err := queueRedis.Ping(); err != nil {
		response.JSON(c, gin.H{
			"status": "error",
			"queue_db": "unavailable",
			"error": err.Error(),
		})
		return
	}
	
	response.Data(c, gin.H{
		"status": "ok",
		"main_db": "available",
		"queue_db": "available",
		"time": time.Now().Unix(),
	})
} 