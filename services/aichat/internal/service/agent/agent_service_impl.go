package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"weave/services/aichat/internal/model"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// 重启策略配置
const (
	maxRestartsPerHour = 5                // 每小时最大重启次数
	cooldownPeriod     = 5 * time.Minute  // 重启冷却期
	gracefulTimeout    = 30 * time.Second // 优雅关闭超时
)

// agentServiceImpl agent服务实现
type agentServiceImpl struct {
	agent         *react.Agent
	healthChecker *HealthChecker
	status        *AgentHealthStatus
	mutex         sync.RWMutex
	logger        *zap.Logger

	// 重启控制
	restartCount    int
	lastRestartTime time.Time
	shuttingDown    bool
}

// NewAgentService 创建agent服务实例
func NewAgentService(logger *zap.Logger) AgentService {
	service := &agentServiceImpl{
		logger: logger,
	}

	// 初始化健康检查器（每30秒检查一次，错误阈值3次）
	service.healthChecker = NewHealthChecker(30*time.Second, 3, logger)

	return service
}

// CreateAgent 创建Agent
func (s *agentServiceImpl) CreateAgent(ctx context.Context) (*react.Agent, error) {
	// 如果agent已存在，直接返回
	if s.agent != nil {
		return s.agent, nil
	}

	// 初始化配置
	viper.SetConfigFile("../.env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("未找到 .env 文件或读取失败: %v，将使用环境变量或默认值", err)
	}

	// 创建agent
	agent, err := model.CreateAgent(ctx)
	if err != nil {
		return nil, err
	}

	// 保存agent实例
	s.agent = agent

	// 启动健康监控
	s.startHealthMonitoring()

	return agent, nil
}

// startHealthMonitoring 启动健康监控
func (s *agentServiceImpl) startHealthMonitoring() {
	s.healthChecker.StartBackgroundMonitoring(s.agent, func(status *AgentHealthStatus) {
		s.mutex.Lock()
		s.status = status
		s.mutex.Unlock()

		// 不健康时触发自动重启
		if status.Status == "UNHEALTHY" {
			go s.autoRestart()
		}
	})
}

// autoRestart 自动重启Agent
func (s *agentServiceImpl) autoRestart() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 检查是否正在关闭
	if s.shuttingDown {
		s.logger.Warn("服务正在关闭，跳过重启")
		return
	}

	// 检查冷却期
	if time.Since(s.lastRestartTime) < cooldownPeriod {
		s.logger.Warn("重启冷却期中，跳过重启",
			zap.Time("last_restart", s.lastRestartTime),
			zap.Duration("cooldown", cooldownPeriod))
		return
	}

	// 检查重启次数限制
	if s.restartCount >= maxRestartsPerHour {
		s.logger.Error("超过每小时最大重启次数，停止自动重启",
			zap.Int("restart_count", s.restartCount))
		return
	}

	s.logger.Info("开始自动重启Agent",
		zap.Int("restart_count", s.restartCount+1),
		zap.String("status", s.status.Status),
		zap.String("message", s.status.Message))

	// 执行重启
	if err := s.restartUnsafe(); err != nil {
		s.logger.Error("Agent自动重启失败", zap.Error(err))
	}
}

// restartUnsafe 执行重启（必须在持有锁的情况下调用）
func (s *agentServiceImpl) restartUnsafe() error {
	ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
	defer cancel()

	// 1. 优雅关闭旧Agent
	if s.agent != nil {
		s.gracefulShutdownUnsafe()
	}

	// 2. 创建新Agent
	agent, err := model.CreateAgent(ctx)
	if err != nil {
		return err
	}

	// 3. 替换Agent
	s.agent = agent

	// 4. 更新重启记录
	s.restartCount++
	s.lastRestartTime = time.Now()

	s.logger.Info("Agent自动重启成功",
		zap.Int("restart_count", s.restartCount))

	return nil
}

// gracefulShutdownUnsafe 优雅关闭（必须在持有锁的情况下调用）
func (s *agentServiceImpl) gracefulShutdownUnsafe() {
	s.shuttingDown = true
	defer func() { s.shuttingDown = false }()

	// 释放旧Agent资源
	if s.agent != nil {
		s.agent = nil
	}

	s.logger.Debug("旧Agent资源已释放")
}

// GetCurrentAgent 获取当前Agent
func (s *agentServiceImpl) GetCurrentAgent(ctx context.Context) (*react.Agent, error) {
	s.mutex.RLock()
	agent := s.agent
	s.mutex.RUnlock()

	// 如果agent不存在，创建新的
	if agent == nil {
		return s.CreateAgent(ctx)
	}

	return agent, nil
}

// GetHealthStatus 获取Agent健康状态
func (s *agentServiceImpl) GetHealthStatus() *AgentHealthStatus {
	return s.healthChecker.GetHealthStatus(s.agent)
}

// RecordError 记录错误
func (s *agentServiceImpl) RecordError() {
	s.healthChecker.RecordError()
}

// GetRestartInfo 获取重启信息
func (s *agentServiceImpl) GetRestartInfo() (count int, lastTime time.Time) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.restartCount, s.lastRestartTime
}
