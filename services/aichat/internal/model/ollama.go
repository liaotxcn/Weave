/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/spf13/viper"
)

// CreateOllamaChatModel 创建并返回一个Ollama聊天模型实例
func CreateOllamaChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	baseURL := viper.GetString("AICHAT_OLLAMA_BASE_URL")
	modelName := viper.GetString("AICHAT_OLLAMA_MODEL_NAME")

	if baseURL == "" || modelName == "" {
		return nil, fmt.Errorf("AICHAT_OLLAMA_BASE_URL 或 AICHAT_OLLAMA_MODEL_NAME 未在 .env 文件中配置")
	}

	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,   // Ollama 服务地址
		Model:   modelName, // 模型名称
	})
	if err != nil {
		return nil, fmt.Errorf("create ollama chat model failed: %w", err)
	}
	return chatModel, nil
}
