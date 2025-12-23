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
	"sync"

	"github.com/cloudwego/eino/schema"
)

// InMemoryCache 内存缓存实现结构体
type InMemoryCache struct {
	mutex   sync.RWMutex
	history map[string][]*schema.Message
}

// NewInMemoryCache 创建内存缓存实例
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		history: make(map[string][]*schema.Message),
	}
}

// SaveChatHistory 保存对话历史到内存
func (mc *InMemoryCache) SaveChatHistory(ctx context.Context, userID string, history []*schema.Message) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.history[userID] = history
	return nil
}

// LoadChatHistory 从内存加载对话历史
func (mc *InMemoryCache) LoadChatHistory(ctx context.Context, userID string) ([]*schema.Message, error) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	history, exists := mc.history[userID]
	if !exists {
		return []*schema.Message{}, nil
	}

	// 返回历史的副本，避免外部修改影响内部存储
	result := make([]*schema.Message, len(history))
	copy(result, history)
	return result, nil
}

// AddMessageToHistory 添加消息到对话历史
func (mc *InMemoryCache) AddMessageToHistory(ctx context.Context, userID string, messages ...*schema.Message) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.history[userID] = append(mc.history[userID], messages...)
	return nil
}

// Close 关闭内存缓存（空实现）
func (mc *InMemoryCache) Close() error {
	return nil
}
