package service

import (
	"context"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// ChatService 定义聊天服务接口
type ChatService interface {
	// Initialize 初始化服务
	Initialize(ctx context.Context) error

	// ProcessUserInput 处理用户输入并生成回复
	ProcessUserInput(ctx context.Context, userInput string, userID string) (string, error)

	// ProcessUserInputStream 流式处理用户输入并生成回复
	ProcessUserInputStream(ctx context.Context, userInput string, userID string,
		streamCallback func(content string, isToolCall bool) error,
		controlCallback func() (bool, bool)) (string, error)

	// GetChatHistory 获取用户对话历史
	GetChatHistory(ctx context.Context, userID string) ([]*schema.Message, error)

	// ClearChatHistory 清除用户对话历史
	ClearChatHistory(ctx context.Context, userID string) error

	// Close 关闭服务资源
	Close(ctx context.Context) error
}

// AgentService 定义代理服务接口
type AgentService interface {
	// CreateAgent 创建AI代理
	CreateAgent(ctx context.Context) (*react.Agent, error)

	// GetCurrentAgent 获取当前AI代理
	GetCurrentAgent(ctx context.Context) (*react.Agent, error)
}
