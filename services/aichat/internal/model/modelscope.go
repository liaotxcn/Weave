package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/spf13/viper"
)

// createModelScopeChatModel 内部函数：创建并返回一个ModelScope聊天模型实例
// useVisionModel: 是否使用视觉模型
func createModelScopeChatModel(ctx context.Context, useVisionModel bool) (einomodel.ToolCallingChatModel, error) {
	apiKey := viper.GetString("AICHAT_MODELSCOPE_API_KEY")
	var modelName string

	// 根据是否需要视觉模型选择配置
	if useVisionModel {
		modelName = viper.GetString("AICHAT_MODELSCOPE_VISUAL_MODEL_NAME")
		// 如果视觉模型未配置，回退到默认模型
		if modelName == "" {
			modelName = viper.GetString("AICHAT_MODELSCOPE_MODEL_NAME")
		}
	} else {
		modelName = viper.GetString("AICHAT_MODELSCOPE_MODEL_NAME")
	}

	baseURL := viper.GetString("AICHAT_MODELSCOPE_BASE_URL")

	if apiKey == "" || modelName == "" || baseURL == "" {
		return nil, fmt.Errorf("AICHAT_MODELSCOPE_API_KEY/MODEL_NAME/AICHAT_MODELSCOPE_BASE_URL 未在 .env 文件中配置")
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  apiKey,
	})
	if err != nil {
		return nil, err
	}
	return chatModel, nil
}

// CreateModelScopeChatModel 创建并返回一个ModelScope聊天模型实例
func CreateModelScopeChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return createModelScopeChatModel(ctx, false)
}

// CreateModelScopeVisionChatModel 创建并返回一个ModelScope视觉聊天模型实例
func CreateModelScopeVisionChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return createModelScopeChatModel(ctx, true)
}
