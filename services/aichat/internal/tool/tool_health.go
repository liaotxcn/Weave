package tool

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// 全局工具健康监控器实例
var GlobalToolHealthMonitor *ToolHealthMonitor

// SetGlobalToolHealthMonitor 设置全局工具健康监控器
func SetGlobalToolHealthMonitor(monitor *ToolHealthMonitor) {
	GlobalToolHealthMonitor = monitor
}

// RecordToolCall 全局函数：记录工具调用结果
func RecordToolCall(toolName string, success bool, responseTime time.Duration) {
	if GlobalToolHealthMonitor != nil {
		GlobalToolHealthMonitor.RecordToolCall(toolName, success, responseTime)
	}
}

// ToolHealthStatus 工具健康状态
type ToolHealthStatus struct {
	ToolName     string        `json:"tool_name"`
	IsHealthy    bool          `json:"is_healthy"`
	LastCheck    time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorCount   int           `json:"error_count"`
	TotalCalls   int           `json:"total_calls"`
	SuccessRate  float64       `json:"success_rate"`
	mutex        sync.RWMutex
}

// ToolHealthMonitor 工具健康监控器
type ToolHealthMonitor struct {
	toolStatus    map[string]*ToolHealthStatus
	checkInterval time.Duration
	logger        *zap.Logger
	mutex         sync.RWMutex
}

// NewToolHealthMonitor 创建工具健康监控器
func NewToolHealthMonitor(checkInterval time.Duration, logger *zap.Logger) *ToolHealthMonitor {
	monitor := &ToolHealthMonitor{
		toolStatus:    make(map[string]*ToolHealthStatus),
		checkInterval: checkInterval,
		logger:        logger,
	}

	// 启动健康检查协程
	go monitor.startHealthCheckLoop()

	// 设置为全局监控器
	SetGlobalToolHealthMonitor(monitor)

	return monitor
}

// RegisterTool 注册工具到健康监控
func (m *ToolHealthMonitor) RegisterTool(toolName string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.toolStatus[toolName]; !exists {
		m.toolStatus[toolName] = &ToolHealthStatus{
			ToolName:  toolName,
			IsHealthy: true, // 初始状态为健康
			LastCheck: time.Now(),
		}
		m.logger.Info("工具已注册到健康监控", zap.String("tool_name", toolName))
	}
}

// RecordToolCall 记录工具调用结果
func (m *ToolHealthMonitor) RecordToolCall(toolName string, success bool, responseTime time.Duration) {
	m.mutex.RLock()
	status, exists := m.toolStatus[toolName]
	m.mutex.RUnlock()

	if !exists {
		// 如果工具未注册，自动注册
		m.RegisterTool(toolName)
		m.mutex.RLock()
		status = m.toolStatus[toolName]
		m.mutex.RUnlock()
	}

	status.mutex.Lock()
	defer status.mutex.Unlock()

	status.TotalCalls++
	status.ResponseTime = (status.ResponseTime*time.Duration(status.TotalCalls-1) + responseTime) / time.Duration(status.TotalCalls)

	if success {
		status.ErrorCount = 0 // 成功调用重置错误计数
		status.SuccessRate = float64(status.TotalCalls-status.ErrorCount) / float64(status.TotalCalls)
	} else {
		status.ErrorCount++
		status.SuccessRate = float64(status.TotalCalls-status.ErrorCount) / float64(status.TotalCalls)

		// 连续错误达到阈值，标记为不健康
		if status.ErrorCount >= 3 {
			status.IsHealthy = false
			m.logger.Warn("工具标记为不健康",
				zap.String("tool_name", toolName),
				zap.Int("error_count", status.ErrorCount))
		}
	}

	status.LastCheck = time.Now()
}

// IsToolHealthy 检查工具是否健康
func (m *ToolHealthMonitor) IsToolHealthy(toolName string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	status, exists := m.toolStatus[toolName]
	if !exists {
		return true // 未注册的工具默认视为健康
	}

	status.mutex.RLock()
	defer status.mutex.RUnlock()

	return status.IsHealthy
}

// GetToolStatus 获取工具健康状态
func (m *ToolHealthMonitor) GetToolStatus(toolName string) *ToolHealthStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	status, exists := m.toolStatus[toolName]
	if !exists {
		return &ToolHealthStatus{
			ToolName:  toolName,
			IsHealthy: true,
			LastCheck: time.Now(),
		}
	}

	// 返回副本避免并发问题
	status.mutex.RLock()
	defer status.mutex.RUnlock()

	return &ToolHealthStatus{
		ToolName:     status.ToolName,
		IsHealthy:    status.IsHealthy,
		LastCheck:    status.LastCheck,
		ResponseTime: status.ResponseTime,
		ErrorCount:   status.ErrorCount,
		TotalCalls:   status.TotalCalls,
		SuccessRate:  status.SuccessRate,
	}
}

// GetAllToolStatus 获取所有工具的健康状态
func (m *ToolHealthMonitor) GetAllToolStatus() map[string]*ToolHealthStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*ToolHealthStatus)
	for toolName, status := range m.toolStatus {
		status.mutex.RLock()
		result[toolName] = &ToolHealthStatus{
			ToolName:     status.ToolName,
			IsHealthy:    status.IsHealthy,
			LastCheck:    status.LastCheck,
			ResponseTime: status.ResponseTime,
			ErrorCount:   status.ErrorCount,
			TotalCalls:   status.TotalCalls,
			SuccessRate:  status.SuccessRate,
		}
		status.mutex.RUnlock()
	}

	return result
}

// startHealthCheckLoop 启动健康检查循环
func (m *ToolHealthMonitor) startHealthCheckLoop() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.performHealthCheck()
	}
}

// performHealthCheck 执行健康检查
func (m *ToolHealthMonitor) performHealthCheck() {
	m.mutex.RLock()
	tools := make([]string, 0, len(m.toolStatus))
	for toolName := range m.toolStatus {
		tools = append(tools, toolName)
	}
	m.mutex.RUnlock()

	m.logger.Debug("执行工具健康检查", zap.Int("tool_count", len(tools)))

	// 这里可以添加具体的健康检查逻辑
	// 例如：发送测试请求验证工具可用性
	// 目前主要依赖调用记录自动判断健康状态
}

// GetHealthStats 获取健康监控统计信息
func (m *ToolHealthMonitor) GetHealthStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_tools"] = len(m.toolStatus)

	healthyCount := 0
	totalCalls := 0
	for _, status := range m.toolStatus {
		status.mutex.RLock()
		if status.IsHealthy {
			healthyCount++
		}
		totalCalls += status.TotalCalls
		status.mutex.RUnlock()
	}

	stats["healthy_tools"] = healthyCount
	stats["unhealthy_tools"] = len(m.toolStatus) - healthyCount
	stats["total_calls"] = totalCalls

	return stats
}
