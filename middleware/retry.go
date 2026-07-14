package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"weave/pkg"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RetryConfig 重试配置
type RetryConfig struct {
	// MaxRetries 最大重试次数
	MaxRetries int
	// InitialDelay 初始延迟时间
	InitialDelay time.Duration
	// MaxDelay 最大延迟时间
	MaxDelay time.Duration
	// Multiplier 延迟倍数（指数退避）
	Multiplier float64
	// RandomizationFactor 随机因子（避免雷群效应）
	RandomizationFactor float64
	// RetryableFunc 判断是否可重试的函数
	RetryableFunc func(error) bool
	// OnRetry 重试回调函数
	OnRetry func(attempt int, err error)
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:          3,
		InitialDelay:        100 * time.Millisecond,
		MaxDelay:            5 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
		RetryableFunc:       DefaultRetryableFunc,
		OnRetry:             DefaultOnRetry,
	}
}

// DefaultRetryableFunc 默认的可重试判断函数
func DefaultRetryableFunc(err error) bool {
	if err == nil {
		return false
	}

	// 网络相关错误
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			return true
		}
		if netErr.Temporary() {
			return true
		}
	}

	// 系统调用错误
	if syscallErr, ok := err.(*net.OpError); ok {
		if syscallErr.Op == "dial" || syscallErr.Op == "read" || syscallErr.Op == "write" {
			return true
		}
	}

	// HTTP状态码错误
	if httpErr, ok := err.(*HTTPError); ok {
		// 5xx 服务器错误和 429 请求过多可重试
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
	}

	// 连接拒绝错误
	if opErr, ok := err.(*net.OpError); ok {
		if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
			if syscallErr.Err == syscall.ECONNREFUSED {
				return true
			}
		}
	}

	return false
}

// DefaultOnRetry 默认的重试回调函数
func DefaultOnRetry(attempt int, err error) {
	pkg.Warn("Request retry attempt",
		zap.Int("attempt", attempt),
		zap.Error(err),
		zap.Duration("next_retry_delay", calculateDelay(attempt, DefaultRetryConfig())))
}

// HTTPError HTTP错误结构体
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// Retryer 重试器
type Retryer struct {
	config RetryConfig
}

// NewRetryer 创建新的重试器
func NewRetryer(config RetryConfig) *Retryer {
	return &Retryer{
		config: config,
	}
}

// Do 执行带重试的操作
func (r *Retryer) Do(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 计算延迟时间
			delay := r.calculateDelay(attempt)

			// 执行重试回调
			if r.config.OnRetry != nil {
				r.config.OnRetry(attempt, lastErr)
			}

			// 等待延迟时间或上下文取消
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// 继续重试
			}
		}

		// 执行操作
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否可重试
		if r.config.RetryableFunc != nil && !r.config.RetryableFunc(err) {
			break
		}

		// 检查上下文是否取消
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return lastErr
}

// DoWithResult 执行带重试的操作并返回结果
func DoWithResult[T any](retryer *Retryer, ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= retryer.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := retryer.calculateDelay(attempt)

			if retryer.config.OnRetry != nil {
				retryer.config.OnRetry(attempt, lastErr)
			}

			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}

		res, err := fn()
		if err == nil {
			return res, nil
		}

		result = res
		lastErr = err

		if retryer.config.RetryableFunc != nil && !retryer.config.RetryableFunc(err) {
			break
		}

		if ctx.Err() != nil {
			return result, ctx.Err()
		}
	}

	return result, lastErr
}

// calculateDelay 计算延迟时间（指数退避 + 随机抖动）
func (r *Retryer) calculateDelay(attempt int) time.Duration {
	// 指数退避
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt-1))

	// 添加随机抖动（避免雷群效应）
	if r.config.RandomizationFactor > 0 {
		// 生成 [-randomizationFactor, +randomizationFactor] 范围的随机数
		randomFactor := r.config.RandomizationFactor * (2*rand.Float64() - 1)
		delay = delay * (1 + randomFactor)
	}

	// 限制最大延迟时间
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	return time.Duration(delay)
}

// calculateDelay 计算延迟时间的辅助函数
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt-1))

	if config.RandomizationFactor > 0 {
		randomFactor := config.RandomizationFactor * (2*rand.Float64() - 1)
		delay = delay * (1 + randomFactor)
	}

	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}

// HTTPRetryer HTTP重试器
type HTTPRetryer struct {
	retryer *Retryer
	client  *http.Client
}

// NewHTTPRetryer 创建HTTP重试器
func NewHTTPRetryer(config RetryConfig, client *http.Client) *HTTPRetryer {
	return &HTTPRetryer{
		retryer: NewRetryer(config),
		client:  client,
	}
}

// Do 执行HTTP请求并重试
func (h *HTTPRetryer) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return DoWithResult(h.retryer, ctx, func() (*http.Response, error) {
		// 为每次重试创建新的请求体
		var bodyCopy []byte
		if req.Body != nil {
			var err error
			bodyCopy, err = io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			req.Body.Close()
		}

		// 执行请求
		resp, err := h.client.Do(req.WithContext(ctx))

		// 恢复请求体以供下次重试使用
		if len(bodyCopy) > 0 {
			req.Body = io.NopCloser(bytes.NewReader(bodyCopy))
		}

		// 检查HTTP状态码
		if resp != nil && (resp.StatusCode >= 500 || resp.StatusCode == 429) {
			return resp, &HTTPError{
				StatusCode: resp.StatusCode,
				Message:    http.StatusText(resp.StatusCode),
			}
		}

		return resp, err
	})
}

// RetryMiddleware 重试中间件
func RetryMiddleware(config RetryConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// API 路径记录重试信息
		if !shouldRetry(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 注入重试配置到上下文，供服务层使用
		c.Set("retry_config", config)
		c.Next()
	}
}

// shouldRetry 判断是否应该重试该请求
func shouldRetry(path string) bool {
	// 排除不需要重试的路径
	noRetryPaths := []string{
		"/health",
		"/metrics",
		"/auth/login",
		"/auth/register",
	}

	for _, noRetryPath := range noRetryPaths {
		if path == noRetryPath {
			return false
		}
	}

	// 对API路径进行重试
	return strings.HasPrefix(path, "/api/v1/")
}
