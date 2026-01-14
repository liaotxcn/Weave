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

package stream

import (
	"context"
	"fmt"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Generate 使用模型生成响应（支持工具调用）
func Generate(ctx context.Context, llm einomodel.ToolCallingChatModel, in []*schema.Message) (*schema.Message, error) {
	result, err := llm.Generate(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %w", err)
	}
	return result, nil
}

// Stream 使用模型生成流式响应（支持工具调用）
func Stream(ctx context.Context, llm einomodel.ToolCallingChatModel, in []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	result, err := llm.Stream(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %w", err)
	}
	return result, nil
}

// StreamWithToolCall 处理包含工具调用的流式响应
func StreamWithToolCall(ctx context.Context, llm einomodel.ToolCallingChatModel, in []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	// 用于更复杂的工具调用处理
	// 目前与Stream函数功能相同，预留扩展空间
	return Stream(ctx, llm, in)
}
