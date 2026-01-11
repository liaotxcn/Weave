package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
)

// RedisCache Redis缓存客户端
type RedisCache struct {
	client            *redis.Client
	logger            *slog.Logger
	maxQueryCacheSize int // 查询结果缓存的最大数量
}

// NewRedisCache 创建Redis缓存客户端
func NewRedisCache(ctx context.Context, logger *slog.Logger) (*RedisCache, error) {
	redisAddr := os.Getenv("AICHAT_REDIS_ADDR")
	redisPassword := os.Getenv("AICHAT_REDIS_PASSWORD")
	redisDB := 0
	if dbStr := os.Getenv("AICHAT_REDIS_DB"); dbStr != "" {
		fmt.Sscanf(dbStr, "%d", &redisDB)
	}

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// 测试连接
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)
	}

	// 设置默认的最大查询缓存大小
	maxQueryCacheSize := 1000
	if val := os.Getenv("RAG_QUERY_CACHE_SIZE"); val != "" {
		if size, err := fmt.Sscanf(val, "%d", &maxQueryCacheSize); err == nil && size > 0 {
			maxQueryCacheSize = size
		}
	}

	logger.Info("redis缓存初始化成功", slog.String("addr", redisAddr), slog.Int("max_query_cache_size", maxQueryCacheSize))
	return &RedisCache{
		client:            client,
		logger:            logger.With("component", "redis_cache"),
		maxQueryCacheSize: maxQueryCacheSize,
	}, nil
}

// GetDocumentChunks 从缓存获取文档块
func (rc *RedisCache) GetDocumentChunks(ctx context.Context, documentDir string) ([]*schema.Document, error) {
	key := fmt.Sprintf("rag:document_chunks:%s", documentDir)
	data, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("缓存不存在")
		}
		return nil, fmt.Errorf("获取文档块缓存失败: %w", err)
	}

	var chunks []*schema.Document
	if err := json.Unmarshal(data, &chunks); err != nil {
		return nil, fmt.Errorf("解析文档块缓存失败: %w", err)
	}

	return chunks, nil
}

// SetDocumentChunks 将文档块存入缓存
func (rc *RedisCache) SetDocumentChunks(ctx context.Context, documentDir string, chunks []*schema.Document) error {
	key := fmt.Sprintf("rag:document_chunks:%s", documentDir)
	data, err := json.Marshal(chunks)
	if err != nil {
		return fmt.Errorf("序列化文档块失败: %w", err)
	}

	// 设置缓存过期时间为1小时
	return rc.client.Set(ctx, key, data, time.Hour).Err()
}

// hashQuery 对查询字符串进行哈希处理
func hashQuery(query string) string {
	hash := sha256.Sum256([]byte(query))
	return hex.EncodeToString(hash[:])
}

// GetQueryResult 从缓存获取查询结果
func (rc *RedisCache) GetQueryResult(ctx context.Context, query string) (string, error) {
	// 使用查询的哈希值作为键的一部分
	queryHash := hashQuery(query)
	key := fmt.Sprintf("rag:query_result:%s", queryHash)
	return rc.client.Get(ctx, key).Result()
}

// SetQueryResult 将查询结果存入缓存
func (rc *RedisCache) SetQueryResult(ctx context.Context, query string, result string) error {
	// 使用查询的哈希值作为键的一部分
	queryHash := hashQuery(query)
	key := fmt.Sprintf("rag:query_result:%s", queryHash)

	// 执行事务：设置缓存并管理LRU
	pipe := rc.client.TxPipeline()

	// 设置查询结果缓存，过期时间30分钟
	pipe.Set(ctx, key, result, 30*time.Minute)

	// 添加到LRU列表
	lruKey := "rag:query_lru"
	pipe.ZAdd(ctx, lruKey, redis.Z{Score: float64(time.Now().UnixNano()), Member: key})

	// 限制LRU列表大小
	pipe.ZRemRangeByRank(ctx, lruKey, 0, -int64(rc.maxQueryCacheSize+1))

	// 执行事务
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("设置查询结果缓存失败: %w", err)
	}

	return nil
}

// Close 关闭Redis连接
func (rc *RedisCache) Close(ctx context.Context) error {
	return rc.client.Close()
}
