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

	"weave/services/aichat/internal/model/models"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/spf13/viper"
)

func CreateChatModel(ctx context.Context, modelType string) (einomodel.ToolCallingChatModel, error) {
	switch modelType {
	case "openai":
		return models.CreateOpenAIChatModel(ctx)
	case "modelscope":
		return models.CreateModelScopeChatModel(ctx)
	case "ollama":
		return models.CreateOllamaChatModel(ctx)
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", modelType)
	}
}

func CreateVisionChatModel(ctx context.Context, modelType string) (einomodel.ToolCallingChatModel, error) {
	switch modelType {
	case "modelscope":
		return models.CreateModelScopeVisionChatModel(ctx)
	case "openai":
		return models.CreateOpenAIChatModel(ctx)
	case "ollama":
		return models.CreateOllamaChatModel(ctx)
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", modelType)
	}
}

func GetModelNameByType(modelType string) string {
	switch modelType {
	case "openai":
		return viper.GetString("AICHAT_OPENAI_MODEL_NAME")
	case "modelscope":
		return viper.GetString("AICHAT_MODELSCOPE_MODEL_NAME")
	case "ollama":
		return viper.GetString("AICHAT_OLLAMA_MODEL_NAME")
	default:
		return ""
	}
}
