package agent

import (
	"context"
	"time"

	"github.com/cloudwego/eino/flow/agent/react"
)

// AgentService 定义代理服务接口
type AgentService interface {
	// CreateAgent 创建Agent
	CreateAgent(ctx context.Context) (*react.Agent, error)

	// GetCurrentAgent 获取当前Agent
	GetCurrentAgent(ctx context.Context) (*react.Agent, error)

	// GetHealthStatus 获取Agent健康状态
	GetHealthStatus() *AgentHealthStatus

	// RecordError 记录错误
	RecordError()

	// GetRestartInfo 获取重启信息
	GetRestartInfo() (count int, lastTime time.Time)
}
