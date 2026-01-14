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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
)

// RedisClient Redis客户端结构体
type RedisClient struct {
	client        *redis.Client
	maxMessages   int
	maxMemoryMB   int           // 最大内存使用限制（MB）
	cleanupFreq   time.Duration // 定期清理频率
	cleanupTicker *time.Ticker  // 定期清理定时器
	stopChan      chan struct{} // 停止定期清理的通道
}

// NewRedisClient 创建Redis客户端实例
func NewRedisClient(ctx context.Context) (*RedisClient, error) {
	config := GetDefaultCacheConfig()
	return NewRedisClientWithConfig(ctx, &config)
}

// NewRedisClientWithConfig 基于配置创建Redis客户端实例
func NewRedisClientWithConfig(ctx context.Context, config *CacheConfig) (*RedisClient, error) {
	// 使用配置值或默认值
	redisAddr := config.RedisAddr
	if redisAddr == "" {
		redisAddr = "localhost:6379" // 默认地址
	}

	redisPassword := config.RedisPassword
	redisDB := config.RedisDB

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

	// 默认配置
	maxMessages := 200
	maxMemoryMB := 100             // 默认最大内存使用限制为100MB
	cleanupFreq := 5 * time.Minute // 每5分钟检查一次内存使用

	log.Printf("redis connection established at %s", redisAddr)
	cache := &RedisClient{
		client:      client,
		maxMessages: maxMessages,
		maxMemoryMB: maxMemoryMB,
		cleanupFreq: cleanupFreq,
		stopChan:    make(chan struct{}),
	}

	// 启动定期内存检查
	cache.startMemoryMonitor()
	return cache, nil
}

// GetChatHistoryKey 生成对话历史的Redis键名
func GetChatHistoryKey(userID string) string {
	return fmt.Sprintf("chat:history:%s", userID)
}

// limitHistoryLength 限制对话历史长度
func (rc *RedisClient) limitHistoryLength(history []*schema.Message) []*schema.Message {
	if len(history) > rc.maxMessages {
		// 只保留最近的消息
		return history[len(history)-rc.maxMessages:]
	}
	return history
}

// SaveChatHistory 保存对话历史到Redis
func (rc *RedisClient) SaveChatHistory(ctx context.Context, userID string, history []*schema.Message) error {
	// 限制对话历史长度
	history = rc.limitHistoryLength(history)

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

// startMemoryMonitor 启动Redis内存使用监控
func (rc *RedisClient) startMemoryMonitor() {
	rc.cleanupTicker = time.NewTicker(rc.cleanupFreq)
	go func() {
		for {
			select {
			case <-rc.cleanupTicker.C:
				rc.checkMemoryUsage()
			case <-rc.stopChan:
				return
			}
		}
	}()
}

// checkMemoryUsage 检查Redis内存使用情况并在必要时清理
func (rc *RedisClient) checkMemoryUsage() {
	ctx := context.Background()

	// 获取Redis内存使用情况
	info, err := rc.client.Info(ctx, "memory").Result()
	if err != nil {
		log.Printf("failed to get redis memory info: %v", err)
		return
	}

	// 解析内存使用情况
	var usedMemoryMB float64
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory_rss_human:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				memoryStr := strings.TrimSpace(parts[1])
				if strings.HasSuffix(memoryStr, "MB") {
					memoryVal := strings.TrimSuffix(memoryStr, "MB")
					if val, err := strconv.ParseFloat(memoryVal, 64); err == nil {
						usedMemoryMB = val
						break
					}
				}
			}
		}
	}

	// 如果内存使用超过限制，清理最旧对话
	if usedMemoryMB > float64(rc.maxMemoryMB) {
		log.Printf("redis memory usage %.2fMB exceeds limit %dMB, starting cleanup", usedMemoryMB, rc.maxMemoryMB)
		rc.cleanupOldConversations(ctx)
	}
}

// cleanupOldConversations 清理最旧对话
func (rc *RedisClient) cleanupOldConversations(ctx context.Context) {
	// 获取所有对话键
	keys, err := rc.client.Keys(ctx, "chat:history:*").Result()
	if err != nil {
		log.Printf("failed to get chat history keys: %v", err)
		return
	}

	// 如果对话数量较少，不需要清理
	if len(keys) < 10 {
		return
	}

	// 获取每个键的最后访问时间（使用键的过期时间作为参考）
	type keyWithTime struct {
		key  string
		time time.Time
	}

	keyTimes := make([]keyWithTime, 0, len(keys))

	for _, key := range keys {
		ttl, err := rc.client.TTL(ctx, key).Result()
		if err == nil {
			// 计算过期时间
			expiry := time.Now().Add(ttl)
			keyTimes = append(keyTimes, keyWithTime{key: key, time: expiry})
		}
	}

	// 按过期时间排序（最早过期的在前）
	sort.Slice(keyTimes, func(i, j int) bool {
		return keyTimes[i].time.Before(keyTimes[j].time)
	})

	// 清理最旧的20%对话
	cleanupCount := len(keyTimes) / 5
	if cleanupCount < 1 {
		cleanupCount = 1
	}

	for i := 0; i < cleanupCount && i < len(keyTimes); i++ {
		if err := rc.client.Del(ctx, keyTimes[i].key).Err(); err != nil {
			log.Printf("failed to delete old conversation %s: %v", keyTimes[i].key, err)
		} else {
			log.Printf("deleted old conversation %s to free up memory", keyTimes[i].key)
		}
	}
}

// GetRedisClient 获取底层的 Redis 客户端实例
func (rc *RedisClient) GetRedisClient() *redis.Client {
	return rc.client
}

// Close 关闭Redis连接
func (rc *RedisClient) Close() error {
	// 停止内存监控
	if rc.cleanupTicker != nil {
		close(rc.stopChan)
		rc.cleanupTicker.Stop()
	}
	return rc.client.Close()
}
