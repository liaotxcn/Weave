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
	"time"

	"github.com/cloudwego/eino/schema"
)

// Conversation 对话结构体，用于结构化管理对话历史
type Conversation struct {
	ID        string            // 对话 ID
	UserID    string            // 用户 ID
	StartTime time.Time         // 开始时间
	EndTime   time.Time         // 结束时间
	Messages  []*schema.Message // 消息列表
	Metadata  map[string]string // 元数据（如意图、标签、核心实体）
	Summary   string            // 对话摘要
}

// NewConversation 创建新的对话实例
func NewConversation(userID string) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:        userID + "_" + now.Format("20060102150405"),
		UserID:    userID,
		StartTime: now,
		EndTime:   now,
		Messages:  []*schema.Message{},
		Metadata:  make(map[string]string),
		Summary:   "",
	}
}

// AddMessage 添加消息到对话
func (c *Conversation) AddMessage(message *schema.Message) {
	c.Messages = append(c.Messages, message)
	c.EndTime = time.Now()
}

// GenerateSummary 生成对话摘要
func (c *Conversation) GenerateSummary(summaryGenerator interface{}) (string, error) {
	if sg, ok := summaryGenerator.(interface {
		GenerateSummary(context.Context, []*schema.Message) (string, error)
	}); ok {
		ctx := context.Background()
		summary, err := sg.GenerateSummary(ctx, c.Messages)
		if err != nil {
			return "", err
		}
		c.Summary = summary
		return summary, nil
	}
	return "", nil
}

// UpdateSummary 更新对话摘要
func (c *Conversation) UpdateSummary(summaryGenerator interface{}, newMessages []*schema.Message) (string, error) {
	if sg, ok := summaryGenerator.(interface {
		UpdateSummary(context.Context, string, []*schema.Message) (string, error)
	}); ok {
		ctx := context.Background()
		summary, err := sg.UpdateSummary(ctx, c.Summary, newMessages)
		if err != nil {
			return "", err
		}
		c.Summary = summary
		return summary, nil
	}
	return "", nil
}

// UpdateMetadata 更新对话元数据
func (c *Conversation) UpdateMetadata(key, value string) {
	c.Metadata[key] = value
}

// SetSummary 设置对话摘要
func (c *Conversation) SetSummary(summary string) {
	c.Summary = summary
}

// ConversationManager 对话管理器接口
type ConversationManager interface {
	// GetConversations 获取用户的所有对话
	GetConversations(ctx interface{}, userID string) ([]*Conversation, error)
	// GetConversation 获取指定对话
	GetConversation(ctx interface{}, conversationID string) (*Conversation, error)
	// CreateConversation 创建新对话
	CreateConversation(ctx interface{}, userID string) (*Conversation, error)
	// SaveConversation 保存对话
	SaveConversation(ctx interface{}, conversation *Conversation) error
	// AddMessageToConversation 添加消息到对话
	AddMessageToConversation(ctx interface{}, conversationID string, message *schema.Message) error
	// Close 关闭管理器
	Close() error
}
