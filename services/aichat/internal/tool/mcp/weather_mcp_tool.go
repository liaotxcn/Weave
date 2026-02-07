package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// MCPRequest MCP请求结构
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse MCP响应结构
type MCPResponse struct {
	JSONRPC   string      `json:"jsonrpc"`
	ID        string      `json:"id"`
	Result    interface{} `json:"result,omitempty"`
	Error     *MCPError   `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// MCPError MCP错误结构
type MCPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Implementation 客户端信息结构
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams 初始化请求参数
type InitializeParams struct {
	ProtocolVersion string         `json:"protocol_version"`
	ClientInfo      Implementation `json:"client_info"`
}

// InvokeParams 工具调用参数
type InvokeParams struct {
	Tool   string          `json:"tool"`
	Params json.RawMessage `json:"params"`
}

// WeatherToolParams 天气查询工具参数
type WeatherToolParams struct {
	City string `json:"city"`
}

func main() {
	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	// 读取标准输入，使用Scanner处理不同系统的行结束符
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// 解析请求
		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Println(formatError("", "ParseError", "无效的JSON格式"))
			continue
		}

		// 处理请求
		var resp string
		switch req.Method {
		case "initialize":
			resp = handleInitialize(&req)
		case "list_tools":
			resp = handleListTools(&req)
		case "invoke":
			resp = handleInvoke(&req)
		default:
			resp = formatError(req.ID, "MethodNotFound", "不支持的方法")
		}

		// 发送响应
		fmt.Println(resp)
	}
}

// handleInitialize 处理初始化请求
func handleInitialize(req *MCPRequest) string {
	var params InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return formatError(req.ID, "InvalidParams", "无效的初始化参数")
	}

	if params.ProtocolVersion != "1.0.0" {
		return formatError(req.ID, "ProtocolVersionMismatch", "支持1.0.0版本")
	}

	result := map[string]interface{}{
		"protocol_version": "1.0.0",
		"service_info": map[string]string{
			"name":        "weather_mcp_tool",
			"version":     "1.0.0",
			"description": "天气查询工具",
		},
	}

	return formatSuccess(req.ID, result)
}

// handleListTools 处理列出工具请求
func handleListTools(req *MCPRequest) string {
	// 直接构造工具信息结构，避免JSON字符串解析
	toolsInfo := map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "weather_query",
				"description": "查询指定城市的天气信息",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"city": map[string]string{
							"type":        "string",
							"description": "要查询天气的城市名称",
						},
					},
					"required": []string{"city"},
				},
			},
		},
	}

	return formatSuccess(req.ID, toolsInfo)
}

// handleInvoke 处理工具调用请求
func handleInvoke(req *MCPRequest) string {
	var invokeParams InvokeParams
	if err := json.Unmarshal(req.Params, &invokeParams); err != nil {
		return formatError(req.ID, "InvalidParams", "无效的调用参数")
	}

	if invokeParams.Tool != "weather_query" {
		return formatError(req.ID, "ToolNotFound", "工具不存在")
	}

	var toolParams WeatherToolParams
	if err := json.Unmarshal(invokeParams.Params, &toolParams); err != nil {
		return formatError(req.ID, "InvalidParams", "无效的工具参数")
	}

	// 示例数据
	weatherTypes := []string{"晴天", "多云", "阴天", "小雨", "中雨"}
	temperatures := []int{15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}

	weatherIndex := rand.Intn(len(weatherTypes))
	tempIndex := rand.Intn(len(temperatures))

	weatherInfo := map[string]interface{}{
		"city":        toolParams.City,
		"weather":     weatherTypes[weatherIndex],
		"temperature": temperatures[tempIndex],
		"humidity":    rand.Intn(30) + 50,
		"wind":        fmt.Sprintf("%d级", rand.Intn(5)+1),
		"update_time": time.Now().Format("2006-01-02 15:04:05"),
	}

	return formatSuccess(req.ID, map[string]interface{}{
		"result": weatherInfo,
		"status": "success",
	})
}

// formatSuccess 格式化成功响应
func formatSuccess(id string, result interface{}) string {
	resp := MCPResponse{
		JSONRPC:   "2.0",
		ID:        id,
		Result:    result,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, _ := json.Marshal(resp)
	return string(data)
}

// formatError 格式化错误响应
func formatError(id, code, message string) string {
	resp := MCPResponse{
		JSONRPC:   "2.0",
		ID:        id,
		Error:     &MCPError{Code: code, Message: message},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, _ := json.Marshal(resp)
	return string(data)
}
