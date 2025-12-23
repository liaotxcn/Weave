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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
)

// RedisClient Redis客户端结构体
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient 创建Redis客户端实例
func NewRedisClient(ctx context.Context) (*RedisClient, error) {
	// 从环境变量获取Redis配置
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // 默认地址
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// 测试连接
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Printf("redis connection established at %s", redisAddr)
	return &RedisClient{client: client}, nil
}

// GetChatHistoryKey 生成对话历史的Redis键名
func GetChatHistoryKey(userID string) string {
	return fmt.Sprintf("chat:history:%s", userID)
}

// SaveChatHistory 保存对话历史到Redis
func (rc *RedisClient) SaveChatHistory(ctx context.Context, userID string, history []*schema.Message) error {
	// 将对话历史序列化为JSON
	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal chat history: %w", err)
	}

	// 保存到Redis，设置过期时间（例如7天）
	err = rc.client.Set(ctx, GetChatHistoryKey(userID), data, 7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save chat history to redis: %w", err)
	}

	return nil
}

// LoadChatHistory 从Redis加载对话历史
func (rc *RedisClient) LoadChatHistory(ctx context.Context, userID string) ([]*schema.Message, error) {
	// 从Redis获取对话历史
	data, err := rc.client.Get(ctx, GetChatHistoryKey(userID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			// 键不存在，返回空切片
			return []*schema.Message{}, nil
		}
		return nil, fmt.Errorf("failed to load chat history from redis: %w", err)
	}

	// 反序列化为对话历史
	var history []*schema.Message
	err = json.Unmarshal(data, &history)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat history: %w", err)
	}

	return history, nil
}

// AddMessageToHistory 添加消息到对话历史并保存到Redis
func (rc *RedisClient) AddMessageToHistory(ctx context.Context, userID string, messages ...*schema.Message) error {
	// 先加载现有历史
	history, err := rc.LoadChatHistory(ctx, userID)
	if err != nil {
		return err
	}

	// 添加新消息
	history = append(history, messages...)

	// 保存更新后的历史
	return rc.SaveChatHistory(ctx, userID, history)
}

// Close 关闭Redis连接
func (rc *RedisClient) Close() error {
	return rc.client.Close()
}
