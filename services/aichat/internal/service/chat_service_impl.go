package service

import (
	"context"
	"fmt"
	"strings"

	"weave/pkg"
	"weave/services/aichat/internal/cache"
	"weave/services/aichat/internal/chat"
	"weave/services/aichat/internal/model"
	"weave/services/aichat/internal/template"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// chatServiceImpl 聊天服务实现
type chatServiceImpl struct {
	agent        *react.Agent
	chatCache    cache.Cache
	embedder     embedding.Embedder
	chatTemplate prompt.ChatTemplate
	logger       *pkg.Logger
	filter       *chat.SensitiveFilter
}

// NewChatService 创建聊天服务实例
func NewChatService() ChatService {
	return &chatServiceImpl{}
}

// Initialize 初始化服务
func (s *chatServiceImpl) Initialize(ctx context.Context) error {
	// 初始化日志
	s.logger = pkg.GetLogger()

	// 初始化配置
	viper.SetConfigFile("../.env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		s.logger.Warn("未找到 .env 文件或读取失败，将使用环境变量或默认值", zap.Error(err))
	} else {
		s.logger.Info("已加载 .env 配置文件")
	}

	// 创建agent
	s.logger.Info("开始创建agent")
	var err error
	s.agent, err = model.CreateAgent(ctx)
	if err != nil {
		s.logger.Error("创建agent失败", zap.Error(err))
		return err
	}
	s.logger.Info("创建agent成功")

	// 初始化缓存
	s.chatCache, err = cache.NewRedisClient(ctx)
	if err != nil {
		s.logger.Warn("Redis连接失败，将使用内存缓存", zap.Error(err))
		s.chatCache = cache.NewInMemoryCache()
	}

	// 初始化嵌入器
	s.embedder, err = model.NewOllamaEmbedder(ctx)
	if err != nil {
		s.logger.Warn("创建 Ollama 嵌入模型失败，将使用关键词匹配", zap.Error(err))
		s.embedder = nil // 触发 FilterRelevantHistory 回退机制
	}

	// 创建模板
	s.chatTemplate = template.CreateTemplate()

	// 初始化敏感内容过滤器
	s.filter = chat.NewSensitiveFilter()
	s.logger.Info("敏感内容过滤器初始化完成")

	return nil
}

// ProcessUserInput 处理用户输入并生成回复
func (s *chatServiceImpl) ProcessUserInput(ctx context.Context, userInput string, userID string) (string, error) {
	// 验证用户输入
	isValid, errMsg := s.filter.ValidateInput(userInput)
	if !isValid {
		s.logger.Warn("用户输入包含敏感或恶意内容", zap.String("user_id", userID), zap.String("input", userInput), zap.String("reason", errMsg))
		return "抱歉，您的输入包含不适当的内容，请重新输入。", nil
	}

	// 过滤用户输入中的敏感内容
	filteredInput := s.filter.FilterSensitiveContent(userInput)
	if filteredInput != userInput {
		s.logger.Info("已过滤用户输入中的敏感内容", zap.String("user_id", userID), zap.String("original_input", userInput), zap.String("filtered_input", filteredInput))
	}

	// 加载对话历史
	chatHistory, err := s.chatCache.LoadChatHistory(ctx, userID)
	if err != nil {
		s.logger.Warn("加载对话历史失败，将使用空历史", zap.Error(err), zap.String("user_id", userID))
		chatHistory = []*schema.Message{}
	}

	// 过滤与当前问题相关的对话历史
	filteredHistory := chat.FilterRelevantHistory(ctx, s.embedder, chatHistory, filteredInput, 50)

	// 将历史消息转换为字符串形式
	var chatHistoryStr string
	for _, msg := range filteredHistory {
		if msg.Content != "" {
			chatHistoryStr += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
	}

	// 使用模板生成消息
	messages, err := s.chatTemplate.Format(ctx, map[string]any{
		"role":         "PaiChat",
		"style":        "积极、温暖且专业",
		"question":     filteredInput,
		"chat_history": chatHistoryStr,
	})
	if err != nil {
		s.logger.Error("格式化模板失败", zap.Error(err), zap.String("user_id", userID))
		return "", err
	}

	// 生成回复
	streamReader, err := s.agent.Stream(ctx, messages)
	if err != nil {
		s.logger.Error("生成回复失败", zap.Error(err), zap.String("user_id", userID))
		return "", err
	}
	defer streamReader.Close()

	// 收集完整回复
	var fullContent strings.Builder
	for {
		message, err := streamReader.Recv()
		if err != nil {
			break
		}
		fullContent.WriteString(message.Content)
	}

	// 更新对话历史
	resultContent := fullContent.String()
	chatHistory = append(chatHistory,
		schema.UserMessage(filteredInput),
		schema.AssistantMessage(resultContent, nil),
	)

	// 保存对话历史到缓存
	err = s.chatCache.SaveChatHistory(ctx, userID, chatHistory)
	if err != nil {
		s.logger.Warn("保存对话历史失败", zap.Error(err), zap.String("user_id", userID))
		// 保存失败不影响返回结果
	}

	return resultContent, nil
}

// ProcessUserInputStream 流式处理用户输入并生成回复
func (s *chatServiceImpl) ProcessUserInputStream(ctx context.Context, userInput string, userID string,
	streamCallback func(content string, isToolCall bool) error,
	controlCallback func() (bool, bool)) (string, error) {

	// 验证用户输入
	isValid, errMsg := s.filter.ValidateInput(userInput)
	if !isValid {
		s.logger.Warn("用户输入包含敏感或恶意内容", zap.String("user_id", userID), zap.String("input", userInput), zap.String("reason", errMsg))
		streamCallback("抱歉，您的输入包含不适当的内容，请重新输入。", false)
		return "抱歉，您的输入包含不适当的内容，请重新输入。", nil
	}

	// 过滤用户输入中的敏感内容
	filteredInput := s.filter.FilterSensitiveContent(userInput)
	if filteredInput != userInput {
		s.logger.Info("已过滤用户输入中的敏感内容", zap.String("user_id", userID), zap.String("original_input", userInput), zap.String("filtered_input", filteredInput))
	}

	// 加载对话历史
	chatHistory, err := s.chatCache.LoadChatHistory(ctx, userID)
	if err != nil {
		s.logger.Warn("加载对话历史失败，将使用空历史", zap.Error(err), zap.String("user_id", userID))
		chatHistory = []*schema.Message{}
	}

	// 过滤与当前问题相关的对话历史
	filteredHistory := chat.FilterRelevantHistory(ctx, s.embedder, chatHistory, filteredInput, 50)

	// 将历史消息转换为字符串形式
	var chatHistoryStr string
	for _, msg := range filteredHistory {
		if msg.Content != "" {
			chatHistoryStr += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
	}

	// 使用模板生成消息
	messages, err := s.chatTemplate.Format(ctx, map[string]any{
		"role":         "PaiChat",
		"style":        "积极、温暖且专业",
		"question":     filteredInput,
		"chat_history": chatHistoryStr,
	})
	if err != nil {
		s.logger.Error("格式化模板失败", zap.Error(err), zap.String("user_id", userID))
		return "", err
	}

	// 生成回复（使用流式输出）
	s.logger.Info("开始生成流式回复", zap.String("user_id", userID))
	streamReader, err := s.agent.Stream(ctx, messages)
	if err != nil {
		s.logger.Error("生成回复失败", zap.Error(err), zap.String("user_id", userID))
		return "", err
	}
	defer streamReader.Close()

	// 实时处理流式输出
	var fullContent strings.Builder

	for {
		// 检查控制信号
		isPaused, isStopped := controlCallback()
		if isStopped {
			break
		}

		if !isPaused {
			message, err := streamReader.Recv()
			if err != nil {
				break
			}

			// 检查是否有工具调用
			isToolCall := len(message.ToolCalls) > 0
			if isToolCall {
				for _, toolCall := range message.ToolCalls {
					toolContent := "[调用工具: " + toolCall.Function.Name + "]"
					if err := streamCallback(toolContent, true); err != nil {
						return "", err
					}
					fullContent.WriteString(toolContent)
				}
			} else {
				// 输出当前片段
				if err := streamCallback(message.Content, false); err != nil {
					return "", err
				}
				fullContent.WriteString(message.Content)
			}
		}
	}

	// 更新对话历史
	resultContent := fullContent.String()
	chatHistory = append(chatHistory,
		schema.UserMessage(filteredInput),
		schema.AssistantMessage(resultContent, nil),
	)

	// 保存对话历史到缓存
	err = s.chatCache.SaveChatHistory(ctx, userID, chatHistory)
	if err != nil {
		s.logger.Warn("保存对话历史失败", zap.Error(err), zap.String("user_id", userID))
		// 保存失败不影响返回结果
	}

	return resultContent, nil
}

// GetChatHistory 获取用户对话历史
func (s *chatServiceImpl) GetChatHistory(ctx context.Context, userID string) ([]*schema.Message, error) {
	return s.chatCache.LoadChatHistory(ctx, userID)
}

// ClearChatHistory 清除用户对话历史
func (s *chatServiceImpl) ClearChatHistory(ctx context.Context, userID string) error {
	return s.chatCache.SaveChatHistory(ctx, userID, []*schema.Message{})
}

// Close 关闭服务资源
func (s *chatServiceImpl) Close(ctx context.Context) error {
	if s.chatCache != nil {
		s.chatCache.Close()
	}
	return nil
}
