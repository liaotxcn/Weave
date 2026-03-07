package service

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

// agentServiceImpl agent服务实现
type agentServiceImpl struct {
	agent         *react.Agent
	healthChecker *HealthChecker
	status        *AgentHealthStatus
	mutex         sync.RWMutex
	logger        *zap.Logger
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

// CreateAgent 创建AI代理
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
	})
}

// GetCurrentAgent 获取当前AI代理
func (s *agentServiceImpl) GetCurrentAgent(ctx context.Context) (*react.Agent, error) {
	// 如果agent不存在，创建新的
	if s.agent == nil {
		return s.CreateAgent(ctx)
	}

	return s.agent, nil
}

// GetHealthStatus 获取Agent健康状态
func (s *agentServiceImpl) GetHealthStatus() *AgentHealthStatus {
	return s.healthChecker.GetHealthStatus(s.agent)
}

// RecordError 记录错误
func (s *agentServiceImpl) RecordError() {
	s.healthChecker.RecordError()
}
