package pool

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// MCPConnection MCP连接
type MCPConnection struct {
	Client    *client.Client
	ServerURL string
	LastUsed  time.Time
	IsHealthy bool
	mutex     sync.RWMutex
}

// MCPConnectionPool MCP连接池
type MCPConnectionPool struct {
	connections map[string]*MCPConnection
	maxSize     int
	idleTimeout time.Duration
	mutex       sync.RWMutex
	logger      *zap.Logger
}

// NewMCPConnectionPool 创建新的MCP连接池
func NewMCPConnectionPool(maxSize int, idleTimeout time.Duration, logger *zap.Logger) *MCPConnectionPool {
	return &MCPConnectionPool{
		connections: make(map[string]*MCPConnection),
		maxSize:     maxSize,
		idleTimeout: idleTimeout,
		logger:      logger,
	}
}

// GetConnection 获取MCP连接
func (p *MCPConnectionPool) GetConnection(ctx context.Context, serverURL string) (*MCPConnection, error) {
	p.mutex.RLock()
	conn, exists := p.connections[serverURL]
	p.mutex.RUnlock()

	if exists && conn.IsHealthy {
		conn.mutex.Lock()
		conn.LastUsed = time.Now()
		conn.mutex.Unlock()
		p.logger.Debug("复用现有MCP连接", zap.String("server_url", serverURL))
		return conn, nil
	}

	// 创建新连接
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 双重检查，防止并发创建
	if conn, exists := p.connections[serverURL]; exists && conn.IsHealthy {
		conn.mutex.Lock()
		conn.LastUsed = time.Now()
		conn.mutex.Unlock()
		return conn, nil
	}

	// 清理过期连接
	p.cleanupExpiredConnections()

	// 创建新连接
	var cli *client.Client
	var err error

	// 根据URL类型创建不同类型的客户端
	if isHTTPURL(serverURL) {
		cli, err = client.NewStreamableHttpClient(serverURL)
	} else if isSSEURL(serverURL) {
		cli, err = client.NewSSEMCPClient(serverURL)
	} else {
		// 默认为Stdio客户端
		cli, err = client.NewStdioMCPClient(serverURL, nil, "")
	}

	if err != nil {
		p.logger.Error("创建MCP客户端失败", zap.String("server_url", serverURL), zap.Error(err))
		return nil, err
	}

	// 初始化连接
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "aichat-service",
		Version: "1.0.0",
	}

	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		p.logger.Error("初始化MCP连接失败", zap.String("server_url", serverURL), zap.Error(err))
		return nil, err
	}

	// 创建连接包装器
	newConn := &MCPConnection{
		Client:    cli,
		ServerURL: serverURL,
		LastUsed:  time.Now(),
		IsHealthy: true,
	}

	p.connections[serverURL] = newConn
	p.logger.Info("创建新的MCP连接", zap.String("server_url", serverURL))

	return newConn, nil
}

// ReleaseConnection 释放连接（标记为可用）
func (p *MCPConnectionPool) ReleaseConnection(conn *MCPConnection) {
	conn.mutex.Lock()
	conn.LastUsed = time.Now()
	conn.mutex.Unlock()
}

// MarkConnectionUnhealthy 标记连接为不健康
func (p *MCPConnectionPool) MarkConnectionUnhealthy(serverURL string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if conn, exists := p.connections[serverURL]; exists {
		conn.mutex.Lock()
		conn.IsHealthy = false
		conn.mutex.Unlock()
		p.logger.Warn("标记MCP连接为不健康", zap.String("server_url", serverURL))
	}
}

// CloseConnection 关闭连接
func (p *MCPConnectionPool) CloseConnection(serverURL string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if conn, exists := p.connections[serverURL]; exists {
		if conn.Client != nil {
			conn.Client.Close()
		}
		delete(p.connections, serverURL)
		p.logger.Info("关闭MCP连接", zap.String("server_url", serverURL))
	}
}

// cleanupExpiredConnections 清理过期连接
func (p *MCPConnectionPool) cleanupExpiredConnections() {
	now := time.Now()
	for url, conn := range p.connections {
		conn.mutex.RLock()
		idleTime := now.Sub(conn.LastUsed)
		isHealthy := conn.IsHealthy
		conn.mutex.RUnlock()

		if idleTime > p.idleTimeout || !isHealthy {
			if conn.Client != nil {
				conn.Client.Close()
			}
			delete(p.connections, url)
			p.logger.Info("清理过期MCP连接", zap.String("server_url", url), zap.Duration("idle_time", idleTime))
		}
	}
}

// isHTTPURL 判断是否为HTTP URL
func isHTTPURL(url string) bool {
	return len(url) > 7 && (url[:7] == "http://" || url[:8] == "https://")
}

// isSSEURL 判断是否为SSE URL
func isSSEURL(url string) bool {
	// SSE URL通常包含sse标识
	return len(url) > 3 && (url[len(url)-3:] == "sse" || strings.Contains(url, "/sse/"))
}

// GetStats 获取连接池统计信息
func (p *MCPConnectionPool) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_connections"] = len(p.connections)

	healthyCount := 0
	for _, conn := range p.connections {
		if conn.IsHealthy {
			healthyCount++
		}
	}
	stats["healthy_connections"] = healthyCount
	stats["max_size"] = p.maxSize

	return stats
}
