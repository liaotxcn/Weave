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

package models

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/spf13/viper"
)

func createModelScopeChatModel(ctx context.Context, useVisionModel bool) (einomodel.ToolCallingChatModel, error) {
	apiKey := viper.GetString("AICHAT_MODELSCOPE_API_KEY")
	var modelName string

	rerankModelName := viper.GetString("AICHAT_RERANK_MODELSCOPE_MODEL_NAME")
	if rerankModelName != "" {
		modelName = rerankModelName
	} else {
		if useVisionModel {
			modelName = viper.GetString("AICHAT_MODELSCOPE_VISUAL_MODEL_NAME")
			if modelName == "" {
				modelName = viper.GetString("AICHAT_MODELSCOPE_MODEL_NAME")
			}
		} else {
			modelName = viper.GetString("AICHAT_MODELSCOPE_MODEL_NAME")
		}
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

func CreateModelScopeChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return createModelScopeChatModel(ctx, false)
}

func CreateModelScopeVisionChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return createModelScopeChatModel(ctx, true)
}
