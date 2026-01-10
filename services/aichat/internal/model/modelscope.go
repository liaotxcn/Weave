package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/spf13/viper"
)

// CreateModelScopeChatModel 创建并返回一个ModelScope聊天模型实例
func CreateModelScopeChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	apiKey := viper.GetString("AICHAT_MODELSCOPE_API_KEY")
	modelName := viper.GetString("AICHAT_MODELSCOPE_MODEL_NAME")
	baseURL := viper.GetString("AICHAT_MODELSCOPE_BASE_URL")

	if apiKey == "" || modelName == "" || baseURL == "" {
		return nil, fmt.Errorf("AICHAT_MODELSCOPE_API_KEY、AICHAT_MODELSCOPE_MODEL_NAME 或 AICHAT_MODELSCOPE_BASE_URL 未在 .env 文件中配置")
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
