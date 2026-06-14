package templates

import (
	"fmt"
	"time"

	"weave/pkg"
	"weave/plugins/core"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 插件版本号，用于测试热重载功能
var pluginVersion = "1.0.0"

// ComprehensivePlugin 综合示例插件，整合了热重载、依赖管理和优化路由功能
type ComprehensivePlugin struct {
	pluginManager *core.PluginManager
	startTime     time.Time
}

// NewComprehensivePlugin 创建新的综合示例插件实例
func NewComprehensivePlugin() *ComprehensivePlugin {
	return &ComprehensivePlugin{}
}

// Name 返回插件名称
func (p *ComprehensivePlugin) Name() string {
	return "comprehensive_plugin"
}

// Description 返回插件描述
func (p *ComprehensivePlugin) Description() string {
	return "整合了热重载、依赖管理和优化路由功能的综合示例插件"
}

// Version 返回插件版本
func (p *ComprehensivePlugin) Version() string {
	return pluginVersion
}

// GetDependencies 返回依赖的插件
func (p *ComprehensivePlugin) GetDependencies() []string {
	return []string{"hello"} // 依赖 hello 插件（如果需要更多依赖可添加）
}

// GetConflicts 返回冲突的插件
func (p *ComprehensivePlugin) GetConflicts() []string {
	return []string{} // 与其他插件无冲突
}

// SetPluginManager 设置插件管理器
func (p *ComprehensivePlugin) SetPluginManager(manager *core.PluginManager) {
	p.pluginManager = manager
}

// Init 初始化插件
func (p *ComprehensivePlugin) Init() error {
	p.startTime = time.Now()
	pkg.Info("Plugin initialized", zap.String("plugin", p.Name()), zap.String("version", pluginVersion))

	// 检查并访问依赖的插件
	if helloPlugin, exists := p.pluginManager.GetPlugin("hello"); exists {
		pkg.Info("Dependency plugin available", zap.String("plugin", p.Name()), zap.String("dependency", helloPlugin.Name()))
	} else {
		pkg.Warn("Dependency plugin unavailable, continuing anyway", zap.String("plugin", p.Name()), zap.String("dependency", "hello"))
	}

	return nil
}

// Shutdown 关闭插件
func (p *ComprehensivePlugin) Shutdown() error {
	pkg.Info("Plugin shutdown", zap.String("plugin", p.Name()))
	return nil
}

// OnEnable 插件启用时调用（热重载相关）
func (p *ComprehensivePlugin) OnEnable() error {
	pkg.Info("Plugin enabled, checking dependencies", zap.String("plugin", p.Name()))
	// 在启用时检查依赖是否可用
	for _, depName := range p.GetDependencies() {
		if _, exists := p.pluginManager.GetPlugin(depName); exists {
			pkg.Info("Dependency available", zap.String("plugin", p.Name()), zap.String("dependency", depName))
		} else {
			pkg.Warn("Dependency unavailable", zap.String("plugin", p.Name()), zap.String("dependency", depName))
		}
	}
	return nil
}

// OnDisable 插件禁用时调用（热重载相关）
func (p *ComprehensivePlugin) OnDisable() error {
	pkg.Info("Plugin disabled", zap.String("plugin", p.Name()))
	return nil
}

// GetRoutes 使用优化方式提供路由定义
func (p *ComprehensivePlugin) GetRoutes() []core.Route {
	return []core.Route{
		// 基础信息路由
		{
			Path:         "/",
			Method:       "GET",
			Handler:      p.handlePluginInfo,
			Description:  "获取插件基本信息",
			AuthRequired: false,
			Tags:         []string{"info", "metadata"},
		},
		// 热重载测试路由
		{
			Path:         "/hotreload",
			Method:       "GET",
			Handler:      p.handleHotReloadTest,
			Description:  "测试热重载功能",
			AuthRequired: false,
			Tags:         []string{"hotreload", "test"},
		},
		// 依赖管理路由
		{
			Path:         "/dependencies",
			Method:       "GET",
			Handler:      p.handleDependencies,
			Description:  "查看插件依赖信息",
			AuthRequired: false,
			Tags:         []string{"dependencies", "management"},
		},
		// 示例功能路由 - 问候
		{
			Path:         "/greet",
			Method:       "GET",
			Handler:      p.handleGreet,
			Middlewares:  []gin.HandlerFunc{p.logMiddleware},
			Description:  "问候API示例",
			AuthRequired: false,
			Tags:         []string{"demo", "greeting"},
			Params: map[string]string{
				"name": "可选，问候的对象名称",
			},
		},
		// 示例功能路由 - 回显
		{
			Path:         "/echo",
			Method:       "POST",
			Handler:      p.handleEcho,
			Middlewares:  []gin.HandlerFunc{p.logMiddleware, p.validateEchoRequest},
			Description:  "回显API示例",
			AuthRequired: false,
			Tags:         []string{"demo", "echo"},
		},
	}
}

// GetDefaultMiddlewares 返回插件的默认中间件
func (p *ComprehensivePlugin) GetDefaultMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		p.corsMiddleware,
		// 可以在这里添加认证中间件等其他全局中间件
	}
}

// RegisterRoutes 保留旧的方法以确保兼容性
// 在使用新的GetRoutes方法后，这个方法实际上不会被调用
func (p *ComprehensivePlugin) RegisterRoutes(router *gin.Engine) {
	// 这个方法在使用新的GetRoutes时不会被调用
	// 保留只是为了兼容性
	pkg.Warn("Using legacy RegisterRoutes method, recommend using GetRoutes", zap.String("plugin", p.Name()))
}

// Execute 执行插件功能
func (p *ComprehensivePlugin) Execute(params map[string]interface{}) (interface{}, error) {
	action, ok := params["action"].(string)
	if !ok {
		action = "default"
	}

	switch action {
	case "greet":
		name, _ := params["name"].(string)
		return map[string]interface{}{
				"message": fmt.Sprintf("Hello, %s!", name),
				"version": pluginVersion,
			},
			nil
	case "echo":
		message, _ := params["message"].(string)
		return map[string]interface{}{
				"echo":    message,
				"version": pluginVersion,
			},
			nil
	case "hotreload_test":
		return map[string]interface{}{
				"message":   "热重载测试成功",
				"version":   pluginVersion,
				"timestamp": time.Now().Format("2006-01-02 15:04:05"),
				"uptime":    time.Since(p.startTime).String(),
			},
			nil
	case "get_dependencies":
		var dependenciesStatus []map[string]interface{}
		for _, depName := range p.GetDependencies() {
			if depPlugin, exists := p.pluginManager.GetPlugin(depName); exists {
				dependenciesStatus = append(dependenciesStatus, map[string]interface{}{
					"name":        depName,
					"version":     depPlugin.Version(),
					"description": depPlugin.Description(),
					"status":      "available",
				})
			} else {
				dependenciesStatus = append(dependenciesStatus, map[string]interface{}{
					"name":   depName,
					"status": "missing",
				})
			}
		}
		return map[string]interface{}{
				"plugin":       p.Name(),
				"dependencies": dependenciesStatus,
			},
			nil
	default:
		return map[string]interface{}{
				"plugin":      p.Name(),
				"version":     pluginVersion,
				"description": p.Description(),
				"features": []string{
					"热重载功能支持",
					"插件依赖管理",
					"优化的路由注册机制",
					"示例API功能",
				},
			},
			nil
	}
}

// 路由处理函数

// 获取插件信息
func (p *ComprehensivePlugin) handlePluginInfo(c *gin.Context) {
	c.JSON(200, gin.H{
		"plugin":      p.Name(),
		"description": p.Description(),
		"version":     p.Version(),
		"features": []string{
			"热重载功能支持",
			"插件依赖管理",
			"优化的路由注册机制",
		},
		"available_endpoints": []string{
			"GET /plugins/comprehensive_plugin/ - 获取插件信息",
			"GET /plugins/comprehensive_plugin/hotreload - 测试热重载功能",
			"GET /plugins/comprehensive_plugin/dependencies - 查看依赖信息",
			"GET /plugins/comprehensive_plugin/greet - 问候API",
			"POST /plugins/comprehensive_plugin/echo - 回显API",
		},
	})
}

// 热重载测试处理函数
func (p *ComprehensivePlugin) handleHotReloadTest(c *gin.Context) {
	c.JSON(200, gin.H{
		"message":   "热重载功能测试成功",
		"version":   pluginVersion,
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"uptime":    time.Since(p.startTime).String(),
		"tip":       "修改pluginVersion并重新编译后，刷新此页面可看到版本变化",
	})
}

// 依赖信息处理函数
func (p *ComprehensivePlugin) handleDependencies(c *gin.Context) {
	// 获取所有已注册插件的依赖关系
	depGraph := p.pluginManager.GetDependencyGraph()

	// 获取当前插件的依赖状态
	var dependenciesStatus []map[string]interface{}
	for _, depName := range p.GetDependencies() {
		if depPlugin, exists := p.pluginManager.GetPlugin(depName); exists {
			dependenciesStatus = append(dependenciesStatus, map[string]interface{}{
				"name":        depName,
				"version":     depPlugin.Version(),
				"description": depPlugin.Description(),
				"status":      "available",
			})
		} else {
			dependenciesStatus = append(dependenciesStatus, map[string]interface{}{
				"name":   depName,
				"status": "missing",
			})
		}
	}

	c.JSON(200, gin.H{
		"plugin":           p.Name(),
		"dependencies":     dependenciesStatus,
		"dependency_graph": depGraph,
	})
}

// 问候处理函数
func (p *ComprehensivePlugin) handleGreet(c *gin.Context) {
	name := c.DefaultQuery("name", "World")
	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Hello, %s!", name),
		"plugin":  p.Name(),
		"version": pluginVersion,
	})
}

// 回显处理函数
func (p *ComprehensivePlugin) handleEcho(c *gin.Context) {
	var request struct {
		Message string `json:"message"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"echo":    request.Message,
		"plugin":  p.Name(),
		"version": pluginVersion,
	})
}

// 中间件

// 日志中间件示例
func (p *ComprehensivePlugin) logMiddleware(c *gin.Context) {
	pkg.Debug("Request received",
		zap.String("plugin", p.Name()),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path))
	c.Next()
}

// CORS中间件示例
func (p *ComprehensivePlugin) corsMiddleware(c *gin.Context) {
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
func (p *ComprehensivePlugin) validateEchoRequest(c *gin.Context) {
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
// plugins.PluginManager.Register(templates.NewComprehensivePlugin())

// 访问示例：
// GET  /plugins/comprehensive_plugin/ - 获取插件信息
// GET  /plugins/comprehensive_plugin/hotreload - 测试热重载功能
// GET  /plugins/comprehensive_plugin/dependencies - 查看依赖信息
// GET  /plugins/comprehensive_plugin/greet?name=John - 问候API
// POST /plugins/comprehensive_plugin/echo - 回显API，请求体: {"message": "Hello World"}
