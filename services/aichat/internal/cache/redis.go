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
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"weave/middleware"

	"github.com/redis/go-redis/v9"
)

// RedisClient Redis客户端结构体
type RedisClient struct {
	client        *redis.Client
	maxMessages   int
	maxMemoryMB   int                 // 最大内存使用限制（MB）
	cleanupFreq   time.Duration       // 定期清理频率
	cleanupTicker *time.Ticker        // 定期清理定时器
	stopChan      chan struct{}       // 停止定期清理的通道
	retryer       *middleware.Retryer // 重试器
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

	// 重试器配置
	retryConfig := middleware.DefaultRetryConfig()
	retryConfig.MaxRetries = 3
	retryConfig.InitialDelay = 50 * time.Millisecond
	retryConfig.MaxDelay = 2 * time.Second

	log.Printf("redis connection established at %s", redisAddr)
	cache := &RedisClient{
		client:      client,
		maxMessages: maxMessages,
		maxMemoryMB: maxMemoryMB,
		cleanupFreq: cleanupFreq,
		stopChan:    make(chan struct{}),
		retryer:     middleware.NewRetryer(retryConfig),
	}

	// 启动定期内存检查
	cache.startMemoryMonitor()
	return cache, nil
}

// GetConversationKey 生成结构化对话的Redis键名
func GetConversationKey(conversationID string) string {
	return fmt.Sprintf("chat:conversation:%s", conversationID)
}

// GetUserConversationsKey 生成用户对话关联的Redis键名
func GetUserConversationsKey(userID string) string {
	return fmt.Sprintf("chat:user_conversations:%s", userID)
}

// SaveConversation 保存结构化对话到Redis
func (rc *RedisClient) SaveConversation(ctx context.Context, conversation interface{}) error {
	// 将对话序列化为JSON
	data, err := json.Marshal(conversation)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	// 使用反射获取对话ID和用户ID
	convValue := reflect.ValueOf(conversation)
	if convValue.Kind() == reflect.Ptr {
		convValue = convValue.Elem()
	}

	idField := convValue.FieldByName("ID")
	userIDField := convValue.FieldByName("UserID")

	if !idField.IsValid() || !userIDField.IsValid() {
		return fmt.Errorf("conversation must have ID and UserID fields")
	}

	convID := idField.String()
	userID := userIDField.String()

	// 保存对话
	err = rc.retryer.Do(ctx, func() error {
		return rc.client.Set(ctx, GetConversationKey(convID), data, 7*24*time.Hour).Err()
	})
	if err != nil {
		return fmt.Errorf("failed to save conversation to redis: %w", err)
	}

	// 关联用户和对话
	err = rc.retryer.Do(ctx, func() error {
		// 检查对话是否已存在
		exists, retryErr := rc.client.SIsMember(ctx, GetUserConversationsKey(userID), convID).Result()
		if retryErr != nil {
			return retryErr
		}
		// 如果不存在，添加到集合
		if !exists {
			return rc.client.SAdd(ctx, GetUserConversationsKey(userID), convID).Err()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to associate conversation with user: %w", err)
	}

	// 设置用户对话关联的过期时间
	err = rc.retryer.Do(ctx, func() error {
		return rc.client.Expire(ctx, GetUserConversationsKey(userID), 7*24*time.Hour).Err()
	})
	if err != nil {
		return fmt.Errorf("failed to set expiry for user conversations: %w", err)
	}

	return nil
}

// LoadConversation 从Redis加载结构化对话
func (rc *RedisClient) LoadConversation(ctx context.Context, conversationID string) (interface{}, error) {
	var data []byte
	err := rc.retryer.Do(ctx, func() error {
		var err error
		data, err = rc.client.Get(ctx, GetConversationKey(conversationID)).Bytes()
		return err
	})
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, fmt.Errorf("failed to load conversation from redis: %w", err)
	}

	// 反序列化对话
	var conversation map[string]interface{}
	err = json.Unmarshal(data, &conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}

	return conversation, nil
}

// LoadUserConversations 从Redis加载用户的所有结构化对话
func (rc *RedisClient) LoadUserConversations(ctx context.Context, userID string) ([]interface{}, error) {
	// 获取用户的所有对话ID
	var convIDs []string
	err := rc.retryer.Do(ctx, func() error {
		var err error
		convIDs, err = rc.client.SMembers(ctx, GetUserConversationsKey(userID)).Result()
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user conversations: %w", err)
	}

	// 加载每个对话
	conversations := make([]interface{}, 0, len(convIDs))
	for _, convID := range convIDs {
		conversation, err := rc.LoadConversation(ctx, convID)
		if err == nil {
			conversations = append(conversations, conversation)
		}
	}

	return conversations, nil
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
	var info string
	err := rc.retryer.Do(ctx, func() error {
		var err error
		info, err = rc.client.Info(ctx, "memory").Result()
		return err
	})
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
	// 获取所有结构化对话键
	var keys []string
	err := rc.retryer.Do(ctx, func() error {
		var err error
		keys, err = rc.client.Keys(ctx, "chat:conversation:*").Result()
		return err
	})
	if err != nil {
		log.Printf("failed to get conversation keys: %v", err)
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
		var ttl time.Duration
		if err := rc.retryer.Do(ctx, func() error {
			var err error
			ttl, err = rc.client.TTL(ctx, key).Result()
			return err
		}); err == nil {
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
		if err := rc.retryer.Do(ctx, func() error {
			return rc.client.Del(ctx, keyTimes[i].key).Err()
		}); err != nil {
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
