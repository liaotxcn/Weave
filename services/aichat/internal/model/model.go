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

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/spf13/viper"
)

// CreateChatModel 根据模型类型创建聊天模型实例
func CreateChatModel(ctx context.Context, modelType string) (einomodel.ToolCallingChatModel, error) {
	switch modelType {
	case "openai":
		return CreateOpenAIChatModel(ctx)
	case "modelscope":
		return CreateModelScopeChatModel(ctx)
	case "ollama":
		return CreateOllamaChatModel(ctx)
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", modelType)
	}
}

// CreateVisionChatModel 根据模型类型创建支持视觉的聊天模型实例
func CreateVisionChatModel(ctx context.Context, modelType string) (einomodel.ToolCallingChatModel, error) {
	switch modelType {
	case "modelscope":
		return CreateModelScopeVisionChatModel(ctx)
	case "openai":
		return CreateOpenAIChatModel(ctx) 
	case "ollama":
		return CreateOllamaChatModel(ctx) 
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", modelType)
	}
}

// GetModelNameByType 根据模型类型获取模型名称
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