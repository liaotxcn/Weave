package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"weave/middleware"

	"github.com/redis/go-redis/v9"
)

// EmbeddingCache 向量嵌入缓存接口
type EmbeddingCache interface {
	// Get 获取嵌入向量
	Get(ctx context.Context, text string) ([][]float64, error)
	// Set 设置嵌入向量
	Set(ctx context.Context, text string, embedding [][]float64) error
	// Close 关闭缓存
	Close() error
}

// RedisEmbeddingCache Redis实现的嵌入缓存
type RedisEmbeddingCache struct {
	client     *redis.Client
	expiration time.Duration
	retryer    *middleware.Retryer
}

// NewRedisEmbeddingCache 创建Redis嵌入缓存
func NewRedisEmbeddingCache(ctx context.Context, client *redis.Client) *RedisEmbeddingCache {
	// 重试器配置
	retryConfig := middleware.DefaultRetryConfig()
	retryConfig.MaxRetries = 3
	retryConfig.InitialDelay = 50 * time.Millisecond
	retryConfig.MaxDelay = 2 * time.Second

	return &RedisEmbeddingCache{
		client:     client,
		expiration: 24 * time.Hour, // 嵌入缓存24小时
		retryer:    middleware.NewRetryer(retryConfig),
	}
}

// getEmbeddingKey 生成嵌入向量的Redis键名
func getEmbeddingKey(text string) string {
	// 使用MD5哈希作为键，避免键过长
	hash := md5.Sum([]byte(text))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("embedding:%s", hashStr)
}

// Get 获取嵌入向量
func (c *RedisEmbeddingCache) Get(ctx context.Context, text string) ([][]float64, error) {
	key := getEmbeddingKey(text)

	var data []byte
	err := c.retryer.Do(ctx, func() error {
		var err error
		data, err = c.client.Get(ctx, key).Bytes()
		return err
	})

	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("failed to get embedding from redis: %w", err)
	}

	var embedding [][]float64
	err = json.Unmarshal(data, &embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
	}

	return embedding, nil
}

// Set 设置嵌入向量
func (c *RedisEmbeddingCache) Set(ctx context.Context, text string, embedding [][]float64) error {
	key := getEmbeddingKey(text)

	data, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	err = c.retryer.Do(ctx, func() error {
		return c.client.Set(ctx, key, data, c.expiration).Err()
	})

	if err != nil {
		return fmt.Errorf("failed to set embedding in redis: %w", err)
	}

	return nil
}

// Close 关闭缓存
func (c *RedisEmbeddingCache) Close() error {
	// Redis客户端由外部管理，这里不需要关闭
	return nil
}