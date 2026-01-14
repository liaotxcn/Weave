package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"weave/services/aichat/internal/cache"

	"github.com/redis/go-redis/v9"
)

// UserUploadStats 用户上传统计信息
type UserUploadStats struct {
	Count      int       // 上传图片数量
	LastUpdate time.Time // 最后更新时间
}

// ImageRateLimiter 图片上传速率限制器
type ImageRateLimiter struct {
	cache          cache.Cache   // 缓存实例
	requestLimit   int           // 单个请求最大图片数
	perMinuteLimit int           // 每分钟最大图片数
	timeWindow     time.Duration // 时间窗口
	// 内存存储（当使用内存缓存时）
	memStats        sync.Map      // 存储用户上传统计信息
	cleanupInterval time.Duration // 清理间隔
	cleanupTicker   *time.Ticker  // 清理定时器
	stopCleanup     chan struct{} // 停止清理的通道
}

// NewImageRateLimiter 创建图片上传速率限制器
func NewImageRateLimiter(cache cache.Cache) *ImageRateLimiter {
	limiter := &ImageRateLimiter{
		cache:           cache,
		requestLimit:    5,               // 单个请求最多 5 张图片
		perMinuteLimit:  10,              // 每分钟最多 10 张图片
		timeWindow:      time.Minute,     // 时间窗口为 1 分钟
		cleanupInterval: 5 * time.Minute, // 每 5 分钟清理一次过期数据
		stopCleanup:     make(chan struct{}),
	}

	// 启动定期清理
	limiter.startCleanup()

	return limiter
}

// startCleanup 启动定期清理过期数据的 goroutine
func (r *ImageRateLimiter) startCleanup() {
	r.cleanupTicker = time.NewTicker(r.cleanupInterval)
	go func() {
		for {
			select {
			case <-r.cleanupTicker.C:
				r.cleanupExpired()
			case <-r.stopCleanup:
				r.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanupExpired 清理过期的上传统计信息
func (r *ImageRateLimiter) cleanupExpired() {
	now := time.Now()
	r.memStats.Range(func(key, value interface{}) bool {
		stats, ok := value.(UserUploadStats)
		if !ok {
			return true
		}

		// 如果统计信息已过期，删除它
		if now.Sub(stats.LastUpdate) > r.timeWindow {
			r.memStats.Delete(key)
		}
		return true
	})
}

// Stop 停止速率限制器，清理资源
func (r *ImageRateLimiter) Stop() {
	if r.stopCleanup != nil {
		close(r.stopCleanup)
	}
}

// CheckRequestLimit 检查单个请求中的图片数量限制
func (r *ImageRateLimiter) CheckRequestLimit(imageCount int) error {
	if imageCount > r.requestLimit {
		return fmt.Errorf("单请求图片数量超过限制，最多允许 %d 张", r.requestLimit)
	}
	return nil
}

// CheckRateLimit 检查单位时间内的上传频率限制
func (r *ImageRateLimiter) CheckRateLimit(ctx context.Context, userID string, imageCount int) error {
	key := fmt.Sprintf("image_upload_limit:%s", userID)

	// 尝试使用 Redis 实现
	if redisClient, ok := r.cache.(interface{ GetRedisClient() *redis.Client }); ok {
		return r.checkRedisRateLimit(ctx, redisClient.GetRedisClient(), key, imageCount)
	}

	// 回退到内存存储实现
	return r.checkMemRateLimit(ctx, key, userID, imageCount)
}

// checkRedisRateLimit 使用 Redis 实现速率限制
func (r *ImageRateLimiter) checkRedisRateLimit(ctx context.Context, client *redis.Client, key string, imageCount int) error {
	// 获取当前计数
	currentCount, err := client.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		// Redis 错误，回退到内存存储
		return r.checkMemRateLimit(ctx, key, key, imageCount)
	}

	// 检查是否超过限制
	if currentCount+imageCount > r.perMinuteLimit {
		return fmt.Errorf("图片上传频率超过限制，每分钟最多允许 %d 张", r.perMinuteLimit)
	}

	// 更新计数
	newCount := currentCount + imageCount
	if err := client.Set(ctx, key, newCount, r.timeWindow).Err(); err != nil {
		// Redis 错误，回退到内存存储
		return r.checkMemRateLimit(ctx, key, key, imageCount)
	}

	return nil
}

// checkMemRateLimit 使用内存存储实现速率限制
func (r *ImageRateLimiter) checkMemRateLimit(ctx context.Context, key string, userID string, imageCount int) error {
	now := time.Now()

	// 获取当前统计信息
	value, exists := r.memStats.Load(key)
	var stats UserUploadStats

	if exists {
		if s, ok := value.(UserUploadStats); ok {
			// 检查是否在时间窗口内
			if now.Sub(s.LastUpdate) <= r.timeWindow {
				// 检查是否超过限制
				if s.Count+imageCount > r.perMinuteLimit {
					return fmt.Errorf("图片上传频率超过限制，每分钟最多允许 %d 张", r.perMinuteLimit)
				}
				// 更新计数
				s.Count += imageCount
				s.LastUpdate = now
				stats = s
			} else {
				// 时间窗口已过，重置计数
				stats = UserUploadStats{
					Count:      imageCount,
					LastUpdate: now,
				}
			}
		} else {
			// 类型错误，重置计数
			stats = UserUploadStats{
				Count:      imageCount,
				LastUpdate: now,
			}
		}
	} else {
		// 新用户，初始化计数
		stats = UserUploadStats{
			Count:      imageCount,
			LastUpdate: now,
		}
	}

	// 保存统计信息
	r.memStats.Store(key, stats)

	return nil
}

// ResetLimit 重置用户的上传限制计数
func (r *ImageRateLimiter) ResetLimit(ctx context.Context, userID string) error {
	key := fmt.Sprintf("image_upload_limit:%s", userID)

	// 尝试使用 Redis 实现
	if redisClient, ok := r.cache.(interface{ GetRedisClient() *redis.Client }); ok {
		return redisClient.GetRedisClient().Del(ctx, key).Err()
	}

	// 回退到内存存储实现
	r.memStats.Delete(key)
	return nil
}
