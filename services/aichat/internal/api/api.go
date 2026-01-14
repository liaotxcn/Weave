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

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"weave/middleware"
	"weave/pkg"
	"weave/services/aichat/internal/service"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// API Server 结构体
type APIServer struct {
	chatService service.ChatService
	router      *gin.Engine
	addr        string
	logger      *pkg.Logger
}

// Request/Response 结构体定义

// ChatRequest 聊天请求结构
type ChatRequest struct {
	UserInput    string   `json:"user_input" binding:"required"`
	UserID       string   `json:"user_id" binding:"required"`
	ImageURLs    []string `json:"image_urls"`    // 图片 URL 列表
	Base64Images []string `json:"base64_images"` // Base64 编码的图片列表
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

// ChatHistoryResponse 聊天历史响应结构
type ChatHistoryResponse struct {
	Messages []*schema.Message `json:"messages"`
	Count    int               `json:"count"`
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Error  string `json:"error"`
	Status string `json:"status"`
}

// NewAPIServer 创建新的API服务器
func NewAPIServer(chatService service.ChatService, addr string) *APIServer {
	// Gin 发布模式
	gin.SetMode(gin.ReleaseMode)

	server := &APIServer{
		chatService: chatService,
		router:      gin.Default(),
		addr:        addr,
		logger:      pkg.GetLogger(),
	}

	// 注册路由
	server.registerRoutes()

	return server
}

// registerRoutes 注册API路由
func (s *APIServer) registerRoutes() {
	// API 分组
	api := s.router.Group("/api")

	// 聊天相关路由
	chat := api.Group("/chat").Use(middleware.RateLimiter(20, 30))
	{
		// 非流式聊天接口
		chat.POST("", s.handleChat)

		// 流式聊天接口
		chat.POST("/stream", s.handleChatStream)

		// 聊天历史相关接口
		chat.GET("/history", s.handleGetChatHistory)
		chat.DELETE("/history", s.handleClearChatHistory)
	}

	// 健康检查
	s.router.GET("/health", s.handleHealthCheck)
}

// handleChat 处理非流式聊天请求
func (s *APIServer) handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:  "请求参数无效: " + err.Error(),
			Status: "error",
		})
		return
	}

	// 调用服务层处理
	var content string
	var err error
	
	// 检查是否包含图片
	if len(req.ImageURLs) > 0 || len(req.Base64Images) > 0 {
		// 处理包含图片的请求
		content, err = s.chatService.ProcessUserInputWithImages(c.Request.Context(), req.UserInput, req.UserID, req.ImageURLs, req.Base64Images)
	} else {
		// 处理纯文本请求
		content, err = s.chatService.ProcessUserInput(c.Request.Context(), req.UserInput, req.UserID)
	}
	
	if err != nil {
		s.logger.Error("处理聊天请求失败", zap.Error(err), zap.String("user_id", req.UserID))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:  "处理请求失败: " + err.Error(),
			Status: "error",
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, ChatResponse{
		Content: content,
		Status:  "success",
	})
}

// handleChatStream 处理流式聊天请求
func (s *APIServer) handleChatStream(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:  "请求参数无效: " + err.Error(),
			Status: "error",
		})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建上下文
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 控制变量
	var isStopped bool
	var mu sync.Mutex

	// 控制回调函数
	controlCallback := func() (bool, bool) {
		mu.Lock()
		defer mu.Unlock()
		return false, isStopped // 流式API默认不支持暂停，只支持停止
	}

	// 流式回调函数
	streamCallback := func(content string, isToolCall bool) error {
		// 将内容包装为SSE格式
		response := ChatResponse{
			Content: content,
			Status:  "streaming",
		}

		data, err := json.Marshal(response)
		if err != nil {
			return err
		}

		// 发送SSE消息
		if _, err := c.Writer.WriteString("data: " + string(data) + "\n\n"); err != nil {
			mu.Lock()
			isStopped = true
			mu.Unlock()
			return err
		}

		// 刷新响应
		c.Writer.Flush()

		return nil
	}

	// 使用服务层处理用户输入
	var fullContent string
	var err error
	
	// 检查是否包含图片
	if len(req.ImageURLs) > 0 || len(req.Base64Images) > 0 {
		// 处理包含图片的请求
		fullContent, err = s.chatService.ProcessUserInputStreamWithImages(ctx, req.UserInput, req.UserID, req.ImageURLs, req.Base64Images, streamCallback, controlCallback)
	} else {
		// 处理纯文本请求
		fullContent, err = s.chatService.ProcessUserInputStream(ctx, req.UserInput, req.UserID, streamCallback, controlCallback)
	}
	
	if err != nil && !strings.Contains(err.Error(), "context canceled") {
		s.logger.Error("流式处理请求失败", zap.Error(err), zap.String("user_id", req.UserID))
		response := ErrorResponse{
			Error:  "流式处理失败: " + err.Error(),
			Status: "error",
		}

		data, _ := json.Marshal(response)
		c.Writer.WriteString("data: " + string(data) + "\n\n")
		c.Writer.Flush()
		return
	}

	// 发送结束消息
	finalResponse := ChatResponse{
		Content: fullContent,
		Status:  "completed",
	}

	data, _ := json.Marshal(finalResponse)
	c.Writer.WriteString("data: " + string(data) + "\n\n")
	c.Writer.Flush()
}

// handleGetChatHistory 处理获取聊天历史请求
func (s *APIServer) handleGetChatHistory(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:  "缺少 user_id 参数",
			Status: "error",
		})
		return
	}

	// 获取聊天历史
	messages, err := s.chatService.GetChatHistory(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("获取聊天历史失败", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:  "获取聊天历史失败: " + err.Error(),
			Status: "error",
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, ChatHistoryResponse{
		Messages: messages,
		Count:    len(messages),
	})
}

// handleClearChatHistory 处理清除聊天历史请求
func (s *APIServer) handleClearChatHistory(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:  "缺少 user_id 参数",
			Status: "error",
		})
		return
	}

	// 清除聊天历史
	err := s.chatService.ClearChatHistory(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("清除聊天历史失败", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:  "清除聊天历史失败: " + err.Error(),
			Status: "error",
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "聊天历史已清除",
	})
}

// handleHealthCheck 处理健康检查请求
func (s *APIServer) handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "aichat",
	})
}

// Start 启动API服务器
func (s *APIServer) Start() error {
	s.logger.Info("aichat Server 启动", zap.String("listen_addr", s.addr))
	return s.router.Run(s.addr)
}
