package service

import (
	"context"
	"log"

	"weave/services/aichat/internal/model"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/spf13/viper"
)

// agentServiceImpl agent服务实现
type agentServiceImpl struct {
	agent *react.Agent
}

// NewAgentService 创建agent服务实例
func NewAgentService() AgentService {
	return &agentServiceImpl{}
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

	return agent, nil
}

// GetCurrentAgent 获取当前AI代理
func (s *agentServiceImpl) GetCurrentAgent(ctx context.Context) (*react.Agent, error) {
	// 如果agent不存在，创建新的
	if s.agent == nil {
		return s.CreateAgent(ctx)
	}

	return s.agent, nil
}
