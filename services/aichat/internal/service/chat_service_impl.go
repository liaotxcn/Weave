package service

import (
	"context"
	"strings"
	"time"

	"weave/pkg"
	"weave/services/aichat/internal/cache"
	"weave/services/aichat/internal/chat"
	"weave/services/aichat/internal/model"
	"weave/services/aichat/internal/security"
	"weave/services/aichat/internal/summary"
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
	agent               *react.Agent
	visionAgent         *react.Agent
	chatCache           cache.Cache
	embedder            embedding.Embedder
	chatTemplate        prompt.ChatTemplate
	logger              *pkg.Logger
	filter              *chat.SensitiveFilter
	modelType           string
	rateLimiter         *security.ImageRateLimiter
	activeConversations map[string]*model.Conversation  // 当前活跃对话
	summaryGenerator    *summary.SimpleSummaryGenerator // 摘要生成器
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

	// 创建普通agent
	var err error
	s.agent, err = model.CreateAgent(ctx)
	if err != nil {
		s.logger.Error("创建普通agent失败", zap.Error(err))
		return err
	}
	s.logger.Info("创建普通agent成功")

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

	// 创建模板（单例模式）
	s.chatTemplate = template.GetTemplate()

	// 初始化敏感内容过滤器
	s.filter = chat.NewSensitiveFilter()
	s.logger.Info("敏感内容过滤器初始化完成")

	// 初始化图片上传速率限制器
	s.rateLimiter = security.NewImageRateLimiter(s.chatCache)
	s.logger.Info("图片上传速率限制器初始化完成")

	// 初始化活跃对话映射
	s.activeConversations = make(map[string]*model.Conversation)
	s.logger.Info("活跃对话管理器初始化完成")

	// 初始化摘要生成器
	s.summaryGenerator = summary.NewBM25SummaryGenerator([]string{})
	s.logger.Info("摘要生成器初始化完成")

	return nil
}

// ProcessUserInput 处理用户输入并生成回复
func (s *chatServiceImpl) ProcessUserInput(ctx context.Context, userInput string, userID string) (string, error) {
	return s.processUserInputWithImages(ctx, userInput, userID, nil, nil)
}

// updateSummaryGenerator 更新BM25摘要生成器（增量学习）
func (s *chatServiceImpl) updateSummaryGenerator(conversation *model.Conversation) {
	if s.summaryGenerator != nil && len(conversation.Messages) > 0 {
		// 增量添加新对话内容到BM25词汇表
		for _, msg := range conversation.Messages {
			if msg.Content != "" {
				s.summaryGenerator.UpdateSummary(context.Background(), "", []*schema.Message{msg})
			}
		}
	}
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

	// 提取关键词
	var keywords []string
	if s.summaryGenerator != nil {
		keywords = s.summaryGenerator.ExtractKeywords(filteredInput, 5)
		if len(keywords) > 0 {
			s.logger.Info("提取到关键词", zap.String("user_id", userID), zap.Strings("keywords", keywords))
		}
	}

	// 获取或创建用户的活跃对话
	conversation := s.getOrCreateConversation(userID)

	// 从结构化对话中获取消息历史
	chatHistory := conversation.Messages

	// 过滤与当前问题相关的对话历史
	var filteredHistory []*schema.Message

	// 如果有摘要且历史消息较长，使用摘要替代部分历史消息
	if conversation.Summary != "" && len(chatHistory) > 30 {
		// 创建摘要消息
		summaryMsg := &schema.Message{
			Role:    schema.System,
			Content: "对话摘要：" + conversation.Summary,
		}
		filteredHistory = append(filteredHistory, summaryMsg)

		// 只添加最近的10条消息
		startIdx := len(chatHistory) - 10
		if startIdx < 0 {
			startIdx = 0
		}
		recentMessages := chatHistory[startIdx:]
		filteredHistory = append(filteredHistory, recentMessages...)
		s.logger.Info("使用摘要作为上下文", zap.String("user_id", userID), zap.Int("message_count", len(chatHistory)))
	} else {
		// 过滤相关历史消息
		filteredHistory = chat.FilterRelevantHistory(ctx, s.embedder, chatHistory, filteredInput, 50)
	}

	// 添加关键词上下文
	if len(keywords) > 0 {
		keywordContext := "关键词: " + strings.Join(keywords, ", ")
		keywordMsg := &schema.Message{
			Role:    schema.System,
			Content: keywordContext,
		}
		filteredHistory = append(filteredHistory, keywordMsg)
		s.logger.Info("添加关键词上下文", zap.String("user_id", userID))
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

	// 将历史消息格式化为字符串
	var chatHistoryStr strings.Builder
	for _, msg := range filteredHistory {
		if msg.Role == schema.User {
			chatHistoryStr.WriteString("user: " + msg.Content + "\n")
		} else {
			chatHistoryStr.WriteString("assistant: " + msg.Content + "\n")
		}
	}

	// 构造消息（包含图片数据）
	var messages []*schema.Message
	if len(imageURLs) > 0 || len(base64Images) > 0 {
		messages = []*schema.Message{}
		for _, msg := range filteredHistory {
			messages = append(messages, msg)
		}
		messages = append(messages, userMessage)
		s.logger.Info("使用原始消息格式处理包含图片的请求", zap.String("user_id", userID))
	} else {
		// 纯文本消息，使用模板格式化
		var err error
		messages, err = template.FormatMessage(ctx, "PaiChat", "积极、温暖且专业", chatHistoryStr.String(), filteredInput)
		if err != nil {
			s.logger.Error("模板格式化失败", zap.Error(err), zap.String("user_id", userID))
			// 如果模板格式化失败，回退到直接使用消息
			messages = []*schema.Message{}
			for _, msg := range filteredHistory {
				messages = append(messages, msg)
			}
			messages = append(messages, userMessage)
		}
	}

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
		message, recvErr := streamReader.Recv()
		if recvErr != nil {
			break
		}
		fullContent.WriteString(message.Content)
	}

	// 更新结构化对话
	resultContent := fullContent.String()
	assistantMessage := schema.AssistantMessage(resultContent, nil)
	conversation.AddMessage(userMessage)
	conversation.AddMessage(assistantMessage)

	// 生成或更新摘要
	if s.summaryGenerator != nil {
		// 每5条消息更新一次摘要
		if len(conversation.Messages)%5 == 0 {
			summary, summaryErr := conversation.GenerateSummary(s.summaryGenerator)
			if summaryErr != nil {
				s.logger.Warn("生成摘要失败", zap.Error(summaryErr), zap.String("user_id", userID))
			} else if summary != "" {
				s.logger.Info("生成对话摘要成功", zap.String("user_id", userID), zap.Int("message_count", len(conversation.Messages)))
			}
		}

		// 增量更新TF-IDF词汇表
		s.updateSummaryGenerator(conversation)
	}

	// 保存结构化对话到缓存
	err = s.chatCache.SaveConversation(ctx, conversation)
	if err != nil {
		s.logger.Warn("保存结构化对话失败", zap.Error(err), zap.String("user_id", userID))
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

	// 提取关键词
	var keywords []string
	if s.summaryGenerator != nil {
		keywords = s.summaryGenerator.ExtractKeywords(filteredInput, 5)
		if len(keywords) > 0 {
			s.logger.Info("提取到关键词", zap.String("user_id", userID), zap.Strings("keywords", keywords))
		}
	}

	// 获取或创建用户的活跃对话
	conversation := s.getOrCreateConversation(userID)

	// 从结构化对话中获取消息历史
	chatHistory := conversation.Messages

	// 过滤与当前问题相关的对话历史
	filteredHistory := chat.FilterRelevantHistory(ctx, s.embedder, chatHistory, filteredInput, 50)

	// 添加关键词上下文
	if len(keywords) > 0 {
		keywordContext := "关键词: " + strings.Join(keywords, ", ")
		keywordMsg := &schema.Message{
			Role:    schema.System,
			Content: keywordContext,
		}
		filteredHistory = append(filteredHistory, keywordMsg)
		s.logger.Info("添加关键词上下文", zap.String("user_id", userID))
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

	// 将历史消息格式化为字符串
	var chatHistoryStr strings.Builder
	for _, msg := range filteredHistory {
		if msg.Role == schema.User {
			chatHistoryStr.WriteString("user: " + msg.Content + "\n")
		} else {
			chatHistoryStr.WriteString("assistant: " + msg.Content + "\n")
		}
	}

	// 构造消息（包含图片数据）
	var messages []*schema.Message
	if len(imageURLs) > 0 || len(base64Images) > 0 {
		messages = []*schema.Message{}
		for _, msg := range filteredHistory {
			messages = append(messages, msg)
		}
		messages = append(messages, userMessage)
		s.logger.Info("使用原始消息格式处理包含图片的流式请求", zap.String("user_id", userID))
	} else {
		// 纯文本消息，使用模板格式化
		var err error
		messages, err = template.FormatMessage(ctx, "PaiChat", "积极、温暖且专业", chatHistoryStr.String(), filteredInput)
		if err != nil {
			s.logger.Error("模板格式化失败", zap.Error(err), zap.String("user_id", userID))
			// 如果模板格式化失败，回退到直接使用消息
			messages = []*schema.Message{}
			for _, msg := range filteredHistory {
				messages = append(messages, msg)
			}
			messages = append(messages, userMessage)
		}
	}

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
			message, recvErr := streamReader.Recv()
			if recvErr != nil {
				break
			}

			// 检查是否有工具调用
			isToolCall := len(message.ToolCalls) > 0
			if isToolCall {
				for _, toolCall := range message.ToolCalls {
					toolContent := "[调用工具: " + toolCall.Function.Name + "]"
					if callbackErr := streamCallback(toolContent, true); callbackErr != nil {
						return "", callbackErr
					}
					fullContent.WriteString(toolContent)
				}
			} else {
				// 输出当前片段
				if callbackErr := streamCallback(message.Content, false); callbackErr != nil {
					return "", callbackErr
				}
				fullContent.WriteString(message.Content)
			}
		}
	}

	// 更新结构化对话
	resultContent := fullContent.String()
	assistantMessage := schema.AssistantMessage(resultContent, nil)
	conversation.AddMessage(userMessage)
	conversation.AddMessage(assistantMessage)

	// 生成或更新摘要
	if s.summaryGenerator != nil {
		// 每5条消息更新一次摘要
		if len(conversation.Messages)%5 == 0 {
			summary, summaryErr := conversation.GenerateSummary(s.summaryGenerator)
			if summaryErr != nil {
				s.logger.Warn("生成摘要失败", zap.Error(summaryErr), zap.String("user_id", userID))
			} else if summary != "" {
				s.logger.Info("生成对话摘要成功", zap.String("user_id", userID), zap.Int("message_count", len(conversation.Messages)))
			}
		}
	}

	// 保存结构化对话到缓存
	err = s.chatCache.SaveConversation(ctx, conversation)
	if err != nil {
		s.logger.Warn("保存结构化对话失败", zap.Error(err), zap.String("user_id", userID))
		// 保存失败不影响返回结果
	}

	return resultContent, nil
}

// GetChatHistory 获取用户对话历史
func (s *chatServiceImpl) GetChatHistory(ctx context.Context, userID string) ([]*schema.Message, error) {
	// 从活跃对话中获取消息历史
	if conv, exists := s.activeConversations[userID]; exists {
		return conv.Messages, nil
	}

	// 如果没有活跃对话，尝试从缓存加载
	conversations, err := s.chatCache.LoadUserConversations(ctx, userID)
	if err != nil {
		s.logger.Warn("加载用户对话失败", zap.Error(err), zap.String("user_id", userID))
		return []*schema.Message{}, nil
	}

	// 如果有对话，返回最近的对话消息
	if len(conversations) > 0 {
		if conv, ok := conversations[0].(*model.Conversation); ok {
			return conv.Messages, nil
		}
	}

	return []*schema.Message{}, nil
}

// getOrCreateConversation 获取或创建用户的结构化对话
func (s *chatServiceImpl) getOrCreateConversation(userID string) *model.Conversation {
	// 检查是否已有活跃对话
	if conv, exists := s.activeConversations[userID]; exists {
		// 检查最后活动时间，超过6小时无活动则创建新对话
		if time.Since(conv.EndTime) > 6*time.Hour {
			// 创建新对话
			newConv := model.NewConversation(userID)
			s.activeConversations[userID] = newConv
			return newConv
		}
		return conv
	}

	// 创建新对话
	conv := model.NewConversation(userID)
	s.activeConversations[userID] = conv

	return conv
}

// ClearChatHistory 清除用户对话历史
func (s *chatServiceImpl) ClearChatHistory(ctx context.Context, userID string) error {
	// 清除用户的活跃对话
	delete(s.activeConversations, userID)

	// 清除缓存中的结构化对话
	conversations, err := s.chatCache.LoadUserConversations(ctx, userID)
	if err != nil {
		s.logger.Warn("加载用户对话失败", zap.Error(err), zap.String("user_id", userID))
		return err
	}

	// 删除每个对话
	for _, conv := range conversations {
		if _, ok := conv.(*model.Conversation); ok {
			// 可以添加删除单个对话的方法到缓存接口
			// 目前只清除活跃对话，保留缓存中的对话
		}
	}

	s.logger.Info("已清除用户对话历史", zap.String("user_id", userID))
	return nil
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
