package agent

import (
	"runtime"
	"sync"
	"time"

	"github.com/cloudwego/eino/flow/agent/react"
	"go.uber.org/zap"
)

// AgentHealthStatus Agent健康状态
type AgentHealthStatus struct {
	Status         string    `json:"status"` // HEALTHY, UNHEALTHY, DEGRADED
	LastCheckTime  time.Time `json:"last_check_time"`
	ResponseTime   int64     `json:"response_time"`
	MemoryUsageMB  int64     `json:"memory_usage_mb"`
	GoroutineCount int       `json:"goroutine_count"`
	ErrorCount     int       `json:"error_count"` // 最近错误次数
	Message        string    `json:"message"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	checkInterval      time.Duration
	unhealthyThreshold int
	logger             *zap.Logger
	metrics            *HealthMetrics
	mutex              sync.RWMutex
}

// HealthMetrics 健康指标收集器
type HealthMetrics struct {
	recentErrors []time.Time
	mutex        sync.RWMutex
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(checkInterval time.Duration, unhealthyThreshold int, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		checkInterval:      checkInterval,
		unhealthyThreshold: unhealthyThreshold,
		logger:             logger,
		metrics: &HealthMetrics{
			recentErrors: make([]time.Time, 0),
		},
	}
}

// RecordError 记录错误
func (hc *HealthChecker) RecordError() {
	hc.metrics.mutex.Lock()
	defer hc.metrics.mutex.Unlock()

	now := time.Now()
	hc.metrics.recentErrors = append(hc.metrics.recentErrors, now)

	// 清理超过5分钟的错误记录
	cutoff := now.Add(-5 * time.Minute)
	validErrors := make([]time.Time, 0)
	for _, errTime := range hc.metrics.recentErrors {
		if errTime.After(cutoff) {
			validErrors = append(validErrors, errTime)
		}
	}
	hc.metrics.recentErrors = validErrors
}

// GetRecentErrorCount 获取最近错误次数
func (hc *HealthChecker) GetRecentErrorCount() int {
	hc.metrics.mutex.RLock()
	defer hc.metrics.mutex.RUnlock()
	return len(hc.metrics.recentErrors)
}

// CheckAgentHealth 检查Agent健康状态
func (hc *HealthChecker) CheckAgentHealth(agent *react.Agent) *AgentHealthStatus {
	status := &AgentHealthStatus{
		LastCheckTime: time.Now(),
	}

	// 检查内存使用
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	status.MemoryUsageMB = int64(m.Alloc / 1024 / 1024)

	// 检查Goroutine数量
	status.GoroutineCount = runtime.NumGoroutine()

	// 检查响应时间
	status.ResponseTime = hc.checkResponseTime(agent)

	// 检查错误计数
	status.ErrorCount = hc.GetRecentErrorCount()

	// 评估健康状态
	status.Status = hc.evaluateHealthStatus(status)

	return status
}

// checkResponseTime 检查响应时间
func (hc *HealthChecker) checkResponseTime(agent *react.Agent) int64 {
	if agent == nil {
		return -1
	}

	start := time.Now()

	elapsed := time.Since(start)
	return elapsed.Milliseconds()
}

// evaluateHealthStatus 评估健康状态
func (hc *HealthChecker) evaluateHealthStatus(status *AgentHealthStatus) string {
	// 内存使用超过1GB视为不健康
	if status.MemoryUsageMB > 1024 {
		status.Message = "内存使用过高"
		return "UNHEALTHY"
	}

	// Goroutine数量超过1000视为不健康
	if status.GoroutineCount > 1000 {
		status.Message = "Goroutine数量过多"
		return "UNHEALTHY"
	}

	// 响应时间超过10秒视为不健康
	if status.ResponseTime > 10000 && status.ResponseTime != -1 {
		status.Message = "响应时间过长"
		return "UNHEALTHY"
	}

	// 最近5分钟内错误超过阈值视为不健康
	if status.ErrorCount > hc.unhealthyThreshold {
		status.Message = "错误次数过多"
		return "UNHEALTHY"
	}

	// 内存使用超过500MB或响应时间超过3秒视为降级
	if status.MemoryUsageMB > 500 || (status.ResponseTime > 3000 && status.ResponseTime != -1) {
		status.Message = "性能降级"
		return "DEGRADED"
	}

	status.Message = "运行正常"
	return "HEALTHY"
}

// StartBackgroundMonitoring 启动后台监控
func (hc *HealthChecker) StartBackgroundMonitoring(agent *react.Agent, callback func(*AgentHealthStatus)) {
	hc.logger.Info("启动Agent健康监控", zap.Duration("interval", hc.checkInterval))

	go func() {
		ticker := time.NewTicker(hc.checkInterval)
		defer ticker.Stop()

		for range ticker.C {
			status := hc.CheckAgentHealth(agent)

			// 记录健康状态日志
			switch status.Status {
			case "UNHEALTHY":
				hc.logger.Warn("Agent健康状态异常",
					zap.String("status", status.Status),
					zap.String("message", status.Message),
					zap.Int64("memory_mb", status.MemoryUsageMB),
					zap.Int("goroutines", status.GoroutineCount),
					zap.Int("errors", status.ErrorCount))
			case "DEGRADED":
				hc.logger.Info("Agent性能降级",
					zap.String("message", status.Message),
					zap.Int64("response_time", status.ResponseTime))
			default:
				hc.logger.Debug("Agent健康状态正常",
					zap.Int64("memory_mb", status.MemoryUsageMB),
					zap.Int("goroutines", status.GoroutineCount))
			}

			// 调用回调函数
			if callback != nil {
				callback(status)
			}
		}
	}()
}

// GetHealthStatus 获取当前健康状态
func (hc *HealthChecker) GetHealthStatus(agent *react.Agent) *AgentHealthStatus {
	return hc.CheckAgentHealth(agent)
}
