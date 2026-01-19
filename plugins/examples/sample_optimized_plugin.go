package examples

import (
	"fmt"
	"log"

	"weave/plugins/core"

	"github.com/gin-gonic/gin"
)

// SampleOptimizedPlugin 展示如何使用优化后的插件路由注册机制的示例插件
// 这个插件演示了新的GetRoutes方法的使用，替代了原来的RegisterRoutes方法

type SampleOptimizedPlugin struct {
	pluginManager *core.PluginManager
}

// NewSampleOptimizedPlugin 创建新的SampleOptimizedPlugin实例
func NewSampleOptimizedPlugin() *SampleOptimizedPlugin {
	return &SampleOptimizedPlugin{}
}

// Name 返回插件名称
func (p *SampleOptimizedPlugin) Name() string {
	return "sample_optimized"
}

// Description 返回插件描述
func (p *SampleOptimizedPlugin) Description() string {
	return "一个演示优化后插件路由注册机制的示例插件"
}

// Version 返回插件版本
func (p *SampleOptimizedPlugin) Version() string {
	return "1.0.0"
}

// GetDependencies 返回依赖的插件
func (p *SampleOptimizedPlugin) GetDependencies() []string {
	return []string{} // 不依赖其他插件
}

// GetConflicts 返回冲突的插件
func (p *SampleOptimizedPlugin) GetConflicts() []string {
	return []string{} // 与其他插件无冲突
}

// SetPluginManager 设置插件管理器
func (p *SampleOptimizedPlugin) SetPluginManager(manager *core.PluginManager) {
	p.pluginManager = manager
}

// Init 初始化插件
func (p *SampleOptimizedPlugin) Init() error {
	return nil
}

// Shutdown 关闭插件
func (p *SampleOptimizedPlugin) Shutdown() error {
	log.Printf("%s: 插件已关闭", p.Name())
	return nil
}

// OnEnable 插件启用时调用
func (p *SampleOptimizedPlugin) OnEnable() error {
	log.Printf("%s: 插件已启用", p.Name())
	return nil
}

// OnDisable 插件禁用时调用
func (p *SampleOptimizedPlugin) OnDisable() error {
	log.Printf("%s: 插件已禁用", p.Name())
	return nil
}

// GetRoutes 使用新的方式提供路由定义
// 替代原来的RegisterRoutes方法
func (p *SampleOptimizedPlugin) GetRoutes() []core.Route {
	return []core.Route{
		{
			Path:         "/",
			Method:       "GET",
			Handler:      p.handlePluginInfo,
			Description:  "获取插件信息",
			AuthRequired: false,
			Tags:         []string{"info", "metadata"},
		},
		{
			Path:         "/greet",
			Method:       "GET",
			Handler:      p.handleGreet,
			Middlewares:  []gin.HandlerFunc{p.logMiddleware},
			Description:  "问候API",
			AuthRequired: false,
			Tags:         []string{"demo", "greeting"},
			Params: map[string]string{
				"name": "可选，问候的对象名称",
			},
		},
		{
			Path:         "/echo",
			Method:       "POST",
			Handler:      p.handleEcho,
			Middlewares:  []gin.HandlerFunc{p.logMiddleware, p.validateEchoRequest},
			Description:  "回显API",
			AuthRequired: false,
			Tags:         []string{"demo", "echo"},
		},
	}
}

// GetDefaultMiddlewares 返回插件的默认中间件
// 这些中间件会应用到插件的所有路由上
func (p *SampleOptimizedPlugin) GetDefaultMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		p.corsMiddleware,
		// 可以在这里添加认证中间件等其他全局中间件
	}
}

// RegisterRoutes 保留旧的方法以确保兼容性
// 在使用新的GetRoutes方法后，这个方法实际上不会被调用
func (p *SampleOptimizedPlugin) RegisterRoutes(router *gin.Engine) {
	// 这个方法在使用新的GetRoutes时不会被调用
	// 保留只是为了兼容性
	log.Printf("%s: 注意：使用了旧的RegisterRoutes方法，建议使用新的GetRoutes方法", p.Name())
}

// Execute 执行插件功能
func (p *SampleOptimizedPlugin) Execute(params map[string]interface{}) (interface{}, error) {
	action, ok := params["action"].(string)
	if !ok {
		action = "default"
	}

	switch action {
	case "greet":
		name, _ := params["name"].(string)
		return map[string]interface{}{
				"message": fmt.Sprintf("Hello, %s!", name),
			},
			nil
	case "echo":
		message, _ := params["message"].(string)
		return map[string]interface{}{
				"echo": message,
			},
			nil
	default:
		return map[string]interface{}{
				"plugin":  p.Name(),
				"version": p.Version(),
				"message": "使用新的路由注册机制的示例插件",
			},
			nil
	}
}

// 路由处理函数

// 获取插件信息
func (p *SampleOptimizedPlugin) handlePluginInfo(c *gin.Context) {
	c.JSON(200, gin.H{
		"plugin":      p.Name(),
		"description": p.Description(),
		"version":     p.Version(),
		"message":     "这个插件使用了优化后的路由注册机制",
		"features": []string{
			"使用GetRoutes方法定义路由",
			"支持路由中间件",
			"提供路由元数据",
			"统一的路由管理",
		},
	})
}

// 问候处理函数
func (p *SampleOptimizedPlugin) handleGreet(c *gin.Context) {
	name := c.DefaultQuery("name", "World")
	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Hello, %s!", name),
		"plugin":  p.Name(),
	})
}

// 回显处理函数
func (p *SampleOptimizedPlugin) handleEcho(c *gin.Context) {
	var request struct {
		Message string `json:"message"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"echo":   request.Message,
		"plugin": p.Name(),
	})
}

// 中间件

// 日志中间件示例
func (p *SampleOptimizedPlugin) logMiddleware(c *gin.Context) {
	log.Printf("%s: 收到请求: %s %s", p.Name(), c.Request.Method, c.Request.URL.Path)
	c.Next()
}

// CORS中间件示例
func (p *SampleOptimizedPlugin) corsMiddleware(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// 处理预检请求
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}

	c.Next()
}

// 请求验证中间件示例
func (p *SampleOptimizedPlugin) validateEchoRequest(c *gin.Context) {
	// 仅在POST请求中验证请求体
	if c.Request.Method == "POST" {
		var request struct {
			Message string `json:"message" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{"error": "消息不能为空"})
			c.Abort()
			return
		}

		// 如果验证通过，可以将解析后的数据存储在上下文中供后续处理函数使用
		c.Set("echo_message", request.Message)
	}

	c.Next()
}

// 在main.go中注册这个插件：
// plugins.PluginManager.Register(&plugins.SampleOptimizedPlugin{})

// 访问示例：
// GET  /plugins/sample_optimized/ - 获取插件信息
// GET  /plugins/sample_optimized/greet?name=John - 问候API
// POST /plugins/sample_optimized/echo - 回显API，请求体: {"message": "Hello World"}
