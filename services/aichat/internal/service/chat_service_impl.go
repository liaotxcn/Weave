package service

import (
	"context"
	"strings"

	"weave/pkg"
	"weave/services/aichat/internal/cache"
	"weave/services/aichat/internal/chat"
	"weave/services/aichat/internal/model"
	"weave/services/aichat/internal/security"
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
	visionAgent  *react.Agent
	chatCache    cache.Cache
	embedder     embedding.Embedder
	chatTemplate prompt.ChatTemplate
	logger       *pkg.Logger
	filter       *chat.SensitiveFilter
	modelType    string
	rateLimiter  *security.ImageRateLimiter
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

	// 保存模型类型
	s.modelType = viper.GetString("AICHAT_MODEL_TYPE")

	// 创建对话agent
	var err error
	s.agent, err = model.CreateAgent(ctx)
	if err != nil {
		s.logger.Error("创建普通agent失败", zap.Error(err))
		return err
	}
	s.logger.Info("创建对话agent成功")

	// 创建视觉agent（仅当使用modelscope时）
	if s.modelType == "modelscope" {
		s.visionAgent, err = model.CreateModelScopeVisionAgent(ctx)
		if err != nil {
			s.logger.Warn("创建视觉agent失败，将使用普通agent处理图像", zap.Error(err))
			s.visionAgent = s.agent // 失败时回退到普通agent
		} else {
			s.logger.Info("创建视觉agent成功")
		}
	} else {
		// 非modelscope时，使用普通agent处理所有请求
		s.visionAgent = s.agent
	}

	// 初始化缓存
	s.chatCache, err = cache.NewRedisClient(ctx)
	if err != nil {
		s.logger.Warn("Redis连接失败，将使用内存缓存", zap.Error(err))
		s.chatCache = cache.NewInMemoryCache()
	}

	// 初始化嵌入器
	s.embedder, err = model.NewOllamaEmbedder(ctx)
	if err != nil {

		s.embedder = nil // 触发 FilterRelevantHistory 回退机制
	}

	// 创建模板
	s.chatTemplate = template.CreateTemplate()

	// 初始化敏感内容过滤器
	s.filter = chat.NewSensitiveFilter()
	s.logger.Info("敏感内容过滤器初始化完成")

	// 初始化图片上传速率限制器
	s.rateLimiter = security.NewImageRateLimiter(s.chatCache)
	s.logger.Info("图像上传限流器初始化完成")

	return nil
}

// ProcessUserInput 处理用户输入并生成回复
func (s *chatServiceImpl) ProcessUserInput(ctx context.Context, userInput string, userID string) (string, error) {
	return s.processUserInputWithImages(ctx, userInput, userID, nil, nil)
}

// ProcessUserInputWithImages 处理用户输入（包含图片）并生成回复
func (s *chatServiceImpl) ProcessUserInputWithImages(ctx context.Context, userInput string, userID string, imageURLs []string, base64Images []string) (string, error) {
	return s.processUserInputWithImages(ctx, userInput, userID, imageURLs, base64Images)
}

// processUserInputWithImages 内部方法：处理用户输入（包含图片）并生成回复
func (s *chatServiceImpl) processUserInputWithImages(ctx context.Context, userInput string, userID string, imageURLs []string, base64Images []string) (string, error) {
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

	// 构造消息
	messages := []*schema.Message{}

	// 添加历史消息
	for _, msg := range filteredHistory {
		messages = append(messages, msg)
	}

	// 构造当前用户消息
	userMessage := &schema.Message{
		Role: schema.User,
	}

	// 如果有图片，构造多模态消息
	if len(imageURLs) > 0 || len(base64Images) > 0 {
		// 检查单个请求的图片数量限制
		totalImages := len(imageURLs) + len(base64Images)
		if err := s.rateLimiter.CheckRequestLimit(totalImages); err != nil {
			s.logger.Warn("单请求图片数量超过限制", zap.Error(err), zap.String("user_id", userID))
			return "", err
		}

		// 检查单位时间内的上传频率限制
		if err := s.rateLimiter.CheckRateLimit(ctx, userID, totalImages); err != nil {
			s.logger.Warn("图片上传频率超过限制", zap.Error(err), zap.String("user_id", userID))
			return "", err
		}

		parts := []schema.MessageInputPart{}

		// 处理图片 URL
		for _, imgURL := range imageURLs {
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeImageURL,
				Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{
						URL: &imgURL,
					},
				},
			})
		}

		// 处理 base64 图片
		for _, imgBase64 := range base64Images {
			// 检查 Base64 图片是否合法
			if err := security.IsValidBase64Image(imgBase64); err != nil {
				s.logger.Warn("无效的 Base64 图片", zap.Error(err), zap.String("user_id", userID))
				return "", err
			}
			dataURL := "data:image/jpeg;base64," + imgBase64
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeImageURL,
				Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{
						URL: &dataURL,
					},
				},
			})
		}

		// 添加文本部分
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeText,
			Text: filteredInput,
		})

		userMessage.UserInputMultiContent = parts
	} else {
		// 纯文本消息
		userMessage.Content = filteredInput
	}

	messages = append(messages, userMessage)

	// 选择合适的agent
	targetAgent := s.agent
	if (len(imageURLs) > 0 || len(base64Images) > 0) && s.visionAgent != nil {
		targetAgent = s.visionAgent
		s.logger.Info("使用视觉agent处理包含图片的请求", zap.String("user_id", userID))
	}

	// 生成回复
	streamReader, err := targetAgent.Stream(ctx, messages)
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
		userMessage,
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
	return s.processUserInputStreamWithImages(ctx, userInput, userID, nil, nil, streamCallback, controlCallback)
}

// ProcessUserInputStreamWithImages 流式处理用户输入（包含图片）并生成回复
func (s *chatServiceImpl) ProcessUserInputStreamWithImages(ctx context.Context, userInput string, userID string, imageURLs []string, base64Images []string,
	streamCallback func(content string, isToolCall bool) error,
	controlCallback func() (bool, bool)) (string, error) {
	return s.processUserInputStreamWithImages(ctx, userInput, userID, imageURLs, base64Images, streamCallback, controlCallback)
}

// processUserInputStreamWithImages 内部方法：流式处理用户输入（包含图片）并生成回复
func (s *chatServiceImpl) processUserInputStreamWithImages(ctx context.Context, userInput string, userID string, imageURLs []string, base64Images []string,
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

	// 构造消息
	messages := []*schema.Message{}

	// 添加历史消息
	for _, msg := range filteredHistory {
		messages = append(messages, msg)
	}

	// 构造当前用户消息
	userMessage := &schema.Message{
		Role: schema.User,
	}

	// 如果有图片，构造多模态消息
	if len(imageURLs) > 0 || len(base64Images) > 0 {
		// 检查单个请求的图片数量限制
		totalImages := len(imageURLs) + len(base64Images)
		if err := s.rateLimiter.CheckRequestLimit(totalImages); err != nil {
			s.logger.Warn("单请求图片数量超过限制", zap.Error(err), zap.String("user_id", userID))
			return "", err
		}

		// 检查单位时间内的上传频率限制
		if err := s.rateLimiter.CheckRateLimit(ctx, userID, totalImages); err != nil {
			s.logger.Warn("图片上传频率超过限制", zap.Error(err), zap.String("user_id", userID))
			return "", err
		}

		parts := []schema.MessageInputPart{}

		// 处理图片 URL
		for _, imgURL := range imageURLs {
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeImageURL,
				Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{
						URL: &imgURL,
					},
				},
			})
		}

		// 处理 base64 图片
		for _, imgBase64 := range base64Images {
			// 检查 Base64 图片是否合法
			if err := security.IsValidBase64Image(imgBase64); err != nil {
				s.logger.Warn("无效的 Base64 图片", zap.Error(err), zap.String("user_id", userID))
				return "", err
			}
			dataURL := "data:image/jpeg;base64," + imgBase64
			parts = append(parts, schema.MessageInputPart{
				Type: schema.ChatMessagePartTypeImageURL,
				Image: &schema.MessageInputImage{
					MessagePartCommon: schema.MessagePartCommon{
						URL: &dataURL,
					},
				},
			})
		}

		// 添加文本部分
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeText,
			Text: filteredInput,
		})

		userMessage.UserInputMultiContent = parts
	} else {
		// 纯文本消息
		userMessage.Content = filteredInput
	}

	messages = append(messages, userMessage)

	// 选择合适的agent
	targetAgent := s.agent
	if (len(imageURLs) > 0 || len(base64Images) > 0) && s.visionAgent != nil {
		targetAgent = s.visionAgent
		s.logger.Info("使用视觉agent处理包含图片的流式请求", zap.String("user_id", userID))
	}

	// 生成回复（使用流式输出）
	s.logger.Info("开始生成流式回复", zap.String("user_id", userID))
	streamReader, err := targetAgent.Stream(ctx, messages)
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
		userMessage,
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
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
	if s.chatCache != nil {
		s.chatCache.Close()
	}
	return nil
}
