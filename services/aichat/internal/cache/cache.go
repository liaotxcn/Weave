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

package cache

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// Cache 接口定义了对话历史缓存的基本操作
type Cache interface {
	// SaveChatHistory 保存对话历史
	SaveChatHistory(ctx context.Context, userID string, history []*schema.Message) error

	// LoadChatHistory 加载对话历史
	LoadChatHistory(ctx context.Context, userID string) ([]*schema.Message, error)

	// AddMessageToHistory 添加消息到对话历史
	AddMessageToHistory(ctx context.Context, userID string, messages ...*schema.Message) error

	// Close 关闭缓存连接
	Close() error
}
