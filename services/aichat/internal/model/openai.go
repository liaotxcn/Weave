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

	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/spf13/viper"
)

// CreateOpenAIChatModel 创建并返回一个OpenAI聊天模型实例
func CreateOpenAIChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	key := viper.GetString("AICHAT_OPENAI_API_KEY")
	modelName := viper.GetString("AICHAT_OPENAI_MODEL_NAME")
	baseURL := viper.GetString("AICHAT_OPENAI_BASE_URL")

	if key == "" || modelName == "" || baseURL == "" {
		return nil, fmt.Errorf("AICHAT_OPENAI_API_KEY、AICHAT_OPENAI_MODEL_NAME 或 AICHAT_OPENAI_BASE_URL 未在 .env 文件中配置")
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  key,
	})
	if err != nil {
		return nil, err
	}
	return chatModel, nil
}
