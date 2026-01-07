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

package model

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"weave/pkg"
	"weave/services/aichat/internal/tool"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// 模型支持工具调用的配置映射
var (
	supportedToolCallModels   map[string]bool
	unsupportedToolCallModels map[string]bool
)

// 初始化模型支持配置
func initModelSupportConfig() {
	// 加载工具调用模型列表
	supportedModelList := viper.GetString("SUPPORTED_TOOL_CALL_MODELS")
	unsupportedModelList := viper.GetString("UNSUPPORTED_TOOL_CALL_MODELS")

	// 初始化映射
	supportedToolCallModels = make(map[string]bool)
	unsupportedToolCallModels = make(map[string]bool)

	// 解析支持的模型列表
	if supportedModelList != "" {
		for _, model := range SplitString(supportedModelList, ",") {
			if model = TrimSpace(model); model != "" {
				supportedToolCallModels[model] = true
			}
		}
	}

	// 解析不支持的模型列表
	if unsupportedModelList != "" {
		for _, model := range SplitString(unsupportedModelList, ",") {
			if model = TrimSpace(model); model != "" {
				unsupportedToolCallModels[model] = true
			}
		}
	}
}

// SplitString 分割字符串
func SplitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, sep)
}

// TrimSpace 去除字符串两端空白
func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

// CreateAgent 创建并初始化一个React Agent
func CreateAgent(ctx context.Context) (*react.Agent, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// 获取日志实例
	logger := pkg.GetLogger()
	if err := viper.ReadInConfig(); err != nil {
		logger.Warn("未找到 .env 文件或读取失败，将使用环境变量或默认值", zap.Error(err))
	}

	// 初始化模型支持配置
	initModelSupportConfig()

	// 初始化模型
	var llm einomodel.ToolCallingChatModel
	var err error
	var modelName string

	// 根据配置类型选择模型
	modelType := viper.GetString("AI_MODEL_TYPE")
	if modelType == "openai" {
		llm, err = CreateOpenAIChatModel(ctx)
		modelName = viper.GetString("OPENAI_MODEL_NAME")
	} else if modelType == "modelscope" {
		llm, err = CreateModelScopeChatModel(ctx)
		modelName = viper.GetString("MODELSCOPE_MODEL_NAME")
	} else {
		llm, err = CreateOllamaChatModel(ctx)
		modelName = viper.GetString("OLLAMA_MODEL_NAME")
	}

	if err != nil {
		return nil, err
	}

	// 检查模型是否支持工具调用
	var tools []einotool.BaseTool
	if isModelSupportToolCall(modelName) {
		tools = loadTools(ctx)
		logger.Info("当前模型支持工具调用", zap.String("model_name", modelName), zap.Int("tool_count", len(tools)))
	} else {
		tools = []einotool.BaseTool{}
		logger.Info("当前模型不支持工具调用，将以普通对话模式运行", zap.String("model_name", modelName))
	}

	// 创建React Agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: llm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	})

	if err != nil {
		return nil, err
	}

	return agent, nil
}

// loadTools 加载所有可用的工具
func loadTools(ctx context.Context) []einotool.BaseTool {
	var tools []einotool.BaseTool

	// 添加自定义工具
	tools = append(tools, tool.NewCustomTool())

	// 加载MCP工具
	mcpTools := loadMCPTools(ctx)
	tools = append(tools, mcpTools...)

	return tools
}

// loadMCPTools 加载MCP工具
func loadMCPTools(ctx context.Context) []einotool.BaseTool {
	var allMcpTools []einotool.BaseTool
	logger := pkg.GetLogger()

	// 初始化MCP请求
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "aichat-service",
		Version: "1.0.0",
	}

	// 工具名称去重
	toolNameSet := make(map[string]struct{})

	// 合并工具函数
	mergeToolsFn := func(source string, id string, tools []einotool.BaseTool) {
		logEntry := logger.With(zap.String("source", source), zap.String("id", id))
		count := 0
		for _, t := range tools {
			info, err := t.Info(ctx)
			if err != nil {
				logEntry.Error("获取工具信息失败，跳过工具", zap.Error(err))
				continue
			}
			name := info.Name
			if _, exists := toolNameSet[name]; exists {
				logEntry.Warn("工具名称重复，跳过工具", zap.String("name", name))
				continue
			}
			toolNameSet[name] = struct{}{}
			allMcpTools = append(allMcpTools, t)
			count++
		}
		if count > 0 {
			logEntry.Info("成功加载MCP工具", zap.Int("count", count))
		}
	}

	// 加载本地MCP工具
	mcpRootPath := viper.GetString("MCP_PATH")
	if mcpRootPath != "" {
		// 获取当前工作目录
		cwd, _ := os.Getwd()
		// 标准化路径
		mcpRootPath = filepath.Clean(mcpRootPath)

		// 检查路径是否存在
		if _, err := os.Stat(mcpRootPath); os.IsNotExist(err) {
			logger.Error("MCP_PATH路径不存在", zap.String("path", mcpRootPath))
		} else {
			stdioPaths, err := os.ReadDir(mcpRootPath)
			if err != nil {
				logger.Error("读取 MCP_PATH 失败", zap.String("path", mcpRootPath), zap.Error(err))
			} else {
				for _, path := range stdioPaths {
					serviceName := path.Name()
					// 跳过非可执行文件和隐藏文件
					if path.IsDir() || strings.HasPrefix(serviceName, ".") || strings.HasSuffix(serviceName, ".go") {
						continue
					}

					mcpPath := filepath.Join(mcpRootPath, serviceName)
					// 确保路径是绝对路径
					mcpPathAbs := mcpPath
					if !filepath.IsAbs(mcpPath) {
						mcpPathAbs = filepath.Join(cwd, mcpPath)
					}
					// 标准化路径
					mcpPathAbs = filepath.Clean(mcpPathAbs)

					// MCP客户端初始化添加超时控制
					timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
					defer cancel()

					// 初始化MCP客户端
					cli, err := client.NewStdioMCPClient(mcpPathAbs, nil, "")
					if err != nil {
						logger.Error("初始化MCP客户端失败", zap.String("service", serviceName), zap.Error(err))
						continue
					}

					// 获取MCP工具
					tools, err := loadMCPToolsFromClient(timeoutCtx, cli, &initRequest)
					if err != nil {
						logger.Error("获取MCP工具失败", zap.String("service", serviceName), zap.Error(err))
						continue
					}
					// 跳过重名的tool
					mergeToolsFn("local", serviceName, tools)
				}
			}
		}
	} else {
		logger.Info("MCP_PATH 未设置，跳过本地MCP工具加载")
	}

	// 加载 streamable_http tools
	httpUrlsStr := viper.GetString("MCP_HTTP_URLS")
	httpUrls := pkg.ParseKeyValuePairs(httpUrlsStr)
	if len(httpUrls) > 0 {
		logger.Info("开始加载HTTP MCP工具", zap.Int("count", len(httpUrls)))
		for name, url := range httpUrls {
			if url == "" {
				continue
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()

			tools, err := loadHttpMcpClient(timeoutCtx, url, &initRequest)
			if err != nil {
				logger.Error("加载HTTP MCP失败", zap.String("name", name), zap.String("url", url), zap.Error(err))
				continue
			}
			mergeToolsFn("http", name, tools)
		}
	}

	// 加载 sse tools
	sseUrlsStr := viper.GetString("MCP_SSE_URLS")
	sseUrls := pkg.ParseKeyValuePairs(sseUrlsStr)
	if len(sseUrls) > 0 {
		logger.Info("开始加载SSE MCP工具", zap.Int("count", len(sseUrls)))
		for name, url := range sseUrls {
			if url == "" {
				continue
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()

			tools, err := loadSSEMcpClient(timeoutCtx, url, &initRequest)
			if err != nil {
				logger.Error("加载SSE MCP失败", zap.String("name", name), zap.String("url", url), zap.Error(err))
				continue
			}
			mergeToolsFn("sse", name, tools)
		}
	}

	if len(allMcpTools) == 0 {
		logger.Info("无加载MCP工具")
	} else {
		logger.Info("MCP工具加载完成", zap.Int("total_count", len(allMcpTools)))
	}

	return allMcpTools
}

// loadMCPToolsFromClient 从MCP客户端加载工具
func loadMCPToolsFromClient(ctx context.Context, cli *client.Client, initRequest *mcp.InitializeRequest) ([]einotool.BaseTool, error) {
	if err := cli.Start(ctx); err != nil {
		return nil, fmt.Errorf("启动MCP客户端失败: %w", err)
	}

	if _, err := cli.Initialize(ctx, *initRequest); err != nil {
		return nil, fmt.Errorf("initialize MCP请求失败: %w", err)
	}

	tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
	if err != nil {
		return nil, fmt.Errorf("获取MCP工具失败: %w", err)
	}

	return tools, nil
}

// loadHttpMcpClient 加载并初始化 HTTP MCP 客户端，返回工具列表
func loadHttpMcpClient(ctx context.Context, url string, initRequest *mcp.InitializeRequest) ([]einotool.BaseTool, error) {
	cli, err := client.NewStreamableHttpClient(url)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP MCP客户端失败: %w", err)
	}
	return loadMCPToolsFromClient(ctx, cli, initRequest)
}

// loadSSEMcpClient 加载并初始化 SSE MCP 客户端，返回工具列表
func loadSSEMcpClient(ctx context.Context, url string, initRequest *mcp.InitializeRequest) ([]einotool.BaseTool, error) {
	cli, err := client.NewSSEMCPClient(url)
	if err != nil {
		return nil, fmt.Errorf("创建SSE MCP客户端失败: %w", err)
	}
	return loadMCPToolsFromClient(ctx, cli, initRequest)
}

// isModelSupportToolCall 检查模型是否支持工具调用
func isModelSupportToolCall(modelName string) bool {
	// 检索列表
	if unsupportedToolCallModels != nil && unsupportedToolCallModels[modelName] {
		return false
	}
	// 再检查支持的列表
	if supportedToolCallModels != nil && supportedToolCallModels[modelName] {
		return true
	}
	// 默认不支持
	return false
}
