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
	"os"
	"strconv"
	"time"
)

// Cache 接口定义了对话历史缓存的基本操作
type Cache interface {
	// SaveConversation 保存对话
	SaveConversation(ctx context.Context, conversation interface{}) error

	// LoadConversation 加载对话
	LoadConversation(ctx context.Context, conversationID string) (interface{}, error)

	// LoadUserConversations 加载用户的所有对话
	LoadUserConversations(ctx context.Context, userID string) ([]interface{}, error)

	// Close 关闭缓存连接
	Close() error
}

// CacheConfig 缓存配置结构体
type CacheConfig struct {
	// 缓存类型: "memory" 或 "redis"
	CacheType string
	// Redis配置
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	// 内存缓存配置
	MaxUsers int
	// 通用配置
	MaxMessages int
	MaxMemoryMB int
	TTL         time.Duration
	CleanupFreq time.Duration
}

// GetDefaultCacheConfig 获取默认缓存配置
func GetDefaultCacheConfig() CacheConfig {
	return CacheConfig{
		CacheType:     getEnv("CACHE_TYPE", "memory"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		MaxUsers:      getEnvAsInt("CACHE_MAX_USERS", 1000),
		MaxMessages:   getEnvAsInt("CACHE_MAX_MESSAGES", 100),
		MaxMemoryMB:   getEnvAsInt("CACHE_MAX_MEMORY_MB", 100),
		TTL:           time.Duration(getEnvAsInt("CACHE_TTL_HOURS", 3)) * time.Hour,
		CleanupFreq:   time.Duration(getEnvAsInt("CACHE_CLEANUP_FREQ_MINUTES", 5)) * time.Minute,
	}
}

// CreateCache 创建缓存实例
func CreateCache(ctx context.Context, config *CacheConfig) (Cache, error) {
	if config == nil {
		config = &CacheConfig{}
		*config = GetDefaultCacheConfig()
	}

	switch config.CacheType {
	case "redis":
		return NewRedisClientWithConfig(ctx, config)
	case "memory":
		return NewInMemoryCacheWithConfig(config), nil
	default:
		// 默认使用内存缓存
		return NewInMemoryCacheWithConfig(config), nil
	}
}

// 从环境变量获取字符串值，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 从环境变量获取整数值，如果不存在或解析失败则返回默认值
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
