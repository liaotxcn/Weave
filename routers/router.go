package routers

import (
	"net/http"
	"time"
	"weave/controllers"
	"weave/middleware"
	"weave/pkg"
	"weave/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter 配置路由
func SetupRouter() *gin.Engine {
	router := gin.New()

	// 请求体大小限制
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
		c.Next()
	})

	// 初始化指标管理器
	mm := metrics.NewMetricsManager()

	// 添加基本中间件
	router.Use(gin.Recovery()) // 恢复中间件，处理panic
	router.Use(gin.Logger())   // 使用gin内置的日志中间件
	router.Use(middleware.CORSMiddleware())

	// 注册Prometheus指标导出路由
	mm.RegisterMetricsRouter(router)

	// 注册插件特定指标路由
	router.GET("/metrics/plugins/:name", func(c *gin.Context) {
		_ = c.Param("name")
		// 创建一个自定义的registry，只包含特定插件的指标
		registry := prometheus.NewRegistry()

		// 注册插件执行计数指标
		registry.MustRegister(metrics.PluginExecutionCount)
		// 注册插件执行时间指标
		registry.MustRegister(metrics.PluginExecutionDuration)
		// 注册插件方法调用指标
		registry.MustRegister(metrics.PluginMethodCalls)
		// 注册插件错误指标
		registry.MustRegister(metrics.PluginErrors)
		// 注册插件内存使用指标
		registry.MustRegister(metrics.PluginMemoryUsage)
		// 注册插件重载指标
		registry.MustRegister(metrics.PluginReloads)

		// 使用自定义registry创建handler
		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		})

		// 输出指标
		handler.ServeHTTP(c.Writer, c.Request)
	})

	// 启动指标更新器，每30秒更新一次系统指标
	mm.StartMetricsUpdater(30 * time.Second)

	// 创建一个应用组，为所有其他路由应用完整的中间件链
	appGroup := router.Group("")
	{
		// 添加其他必要的中间件，但仅应用于appGroup而不是全局
		appGroup.Use(middleware.RequestBufferMiddleware())
		appGroup.Use(middleware.CSRFMiddleware())
		appGroup.Use(mm.HTTPMonitoringMiddleware()) // 添加HTTP请求监控中间件
		appGroup.Use(pkg.AuditLogMiddleware())      // 添加安全审计日志中间件

		// 认证相关路由
		auth := appGroup.Group("/auth")
		{
			// 为认证服务添加重试和超时保护
			auth.Use(middleware.RetryMiddleware(middleware.DefaultRetryConfig()))
			auth.Use(middleware.TimeoutMiddleware(middleware.DefaultTimeoutConfig()))

			// 限流保护，为认证接口添加限流：每秒允许10个请求，突发容量20
			auth.Use(middleware.RateLimiter(10, 20))
			userCtrl := controllers.NewUserController()
			auth.POST("/register", userCtrl.Register)
			auth.POST("/login", userCtrl.Login)
			auth.POST("/refresh-token", userCtrl.RefreshToken)
			// 添加验证码相关接口
			auth.POST("/send-verification-code", userCtrl.SendVerificationCode)
			auth.POST("/login-with-code", userCtrl.LoginWithVerificationCode)
		}

		// API分组
		api := appGroup.Group("/api/v1")
		{
			// 使用认证中间件
			api.Use(middleware.AuthMiddleware())
			// 为API接口添加限流：每秒允许20个请求，突发容量50
			api.Use(middleware.RateLimiter(20, 50))

			// 用户相关路由
			users := api.Group("/users")
			{
				// 为用户服务添加重试和超时保护
				users.Use(middleware.RetryMiddleware(middleware.DefaultRetryConfig()))
				users.Use(middleware.TimeoutMiddleware(middleware.DefaultTimeoutConfig()))

				userCtrl := controllers.NewUserController()
				users.GET("/", userCtrl.GetUsers)
				users.GET("/:id", userCtrl.GetUser)
				users.POST("/", userCtrl.CreateUser)
				users.PUT("/:id", userCtrl.UpdateUser)
				users.DELETE("/:id", userCtrl.DeleteUser)
				// 更新密码接口，不需要用户ID参数，当前登录用户修改个人密码
				users.POST("/change-password", userCtrl.ChangePassword)
			}

			// 团队相关路由
			teams := api.Group("/teams")
			{
				// 为团队服务添加重试和超时保护
				teams.Use(middleware.RetryMiddleware(middleware.DefaultRetryConfig()))
				teams.Use(middleware.TimeoutMiddleware(middleware.DefaultTimeoutConfig()))

				teamCtrl := &controllers.TeamController{}
				teams.GET("/", teamCtrl.GetTeams) // 获取用户所属的团队列表
				teams.POST("/", teamCtrl.CreateTeam)
				teams.PUT("/:id", teamCtrl.UpdateTeam)                        // 更新团队信息
				teams.POST("/:id/transfer-owner", teamCtrl.TransferTeamOwner) // 转让团队所有权

				// 团队成员管理路由
				teams.GET("/:id/members", teamCtrl.GetTeamMembers)                  // 获取团队成员列表
				teams.GET("/:id/members/search", teamCtrl.SearchTeamMembers)        // 搜索团队成员
				teams.POST("/:id/members", teamCtrl.AddTeamMember)                  // 添加团队成员
				teams.DELETE("/:id/members/:memberId", teamCtrl.RemoveTeamMember)   // 移除团队成员
				teams.PUT("/:id/members/:memberId/role", teamCtrl.UpdateMemberRole) // 更新成员角色
			}

			// 审计日志相关路由
			audit := api.Group("/audit")
			{
				// 为审计服务添加重试和超时保护
				audit.Use(middleware.RetryMiddleware(middleware.DefaultRetryConfig()))
				audit.Use(middleware.TimeoutMiddleware(middleware.DefaultTimeoutConfig()))

				auditCtrl := &controllers.AuditController{}
				audit.GET("/logs", auditCtrl.GetAuditLogs)    // 获取审计日志列表
				audit.GET("/logs/:id", auditCtrl.GetAuditLog) // 获取单个审计日志详情
				audit.GET("/stats", auditCtrl.GetAuditStats)  // 获取审计日志统计信息
			}

			// 工具相关路由
			tools := api.Group("/tools")
			{
				// 为工具服务添加重试和超时保护
				tools.Use(middleware.RetryMiddleware(middleware.DefaultRetryConfig()))
				tools.Use(middleware.TimeoutMiddleware(middleware.DefaultTimeoutConfig()))

				toolCtrl := &controllers.ToolController{}
				tools.GET("/", toolCtrl.GetTools)
				tools.GET("/:id", toolCtrl.GetTool)
				tools.POST("/", toolCtrl.CreateTool)
				tools.PUT("/:id", toolCtrl.UpdateTool)
				tools.DELETE("/:id", toolCtrl.DeleteTool)
				// 工具执行接口使用更严格的超时配置
				tools.POST("/:id/execute",
					middleware.TimeoutMiddleware(middleware.TimeoutConfig{
						DefaultTimeout: 60 * time.Second, // 工具执行使用60秒超时
					}),
					toolCtrl.ExecuteTool)
			}

			// 插件相关路由
			plugins := api.Group("/plugins")
			{
				// 为插件服务添加重试和超时保护
				plugins.Use(middleware.RetryMiddleware(middleware.DefaultRetryConfig()))
				plugins.Use(middleware.TimeoutMiddleware(middleware.DefaultTimeoutConfig()))

				pluginCtrl := &controllers.PluginController{}
				// 获取所有插件信息
				plugins.GET("/", pluginCtrl.GetAllPlugins)
				// 获取插件状态
				plugins.GET("/:name/status", pluginCtrl.GetPluginStatus)
				// 启用插件
				plugins.POST("/:name/enable", pluginCtrl.EnablePlugin)
				// 禁用插件
				plugins.POST("/:name/disable", pluginCtrl.DisablePlugin)
				// 重载插件
				plugins.POST("/:name/reload", pluginCtrl.ReloadPlugin)
				// 获取插件依赖图
				plugins.GET("/dependency-graph", pluginCtrl.GetDependencyGraph)
			}

			// 负载均衡管理路由
			loadbalancer := api.Group("/loadbalancer")
			{
				// 为负载均衡服务添加重试和超时保护
				loadbalancer.Use(middleware.RetryMiddleware(middleware.DefaultRetryConfig()))
				loadbalancer.Use(middleware.TimeoutMiddleware(middleware.DefaultTimeoutConfig()))

				lbCtrl := &controllers.LoadBalancerController{}
				// 获取负载均衡状态
				loadbalancer.GET("/status", lbCtrl.GetLoadBalancerStatus)
				// 获取特定实例健康状态
				loadbalancer.GET("/instance/:instanceId/health", lbCtrl.GetInstanceHealth)
				// 更新实例权重
				loadbalancer.PUT("/instance/:instanceId/weight", lbCtrl.UpdateInstanceWeight)
				// 排干实例（停止接收新请求）
				loadbalancer.POST("/instance/:instanceId/drain", lbCtrl.DrainInstance)
				// 启用实例
				loadbalancer.POST("/instance/:instanceId/enable", lbCtrl.EnableInstance)
			}
		}
	}

	// 根路径和健康检查路由放在appGroup内，确保一致的中间件处理
	healthCtrl := &controllers.HealthController{}
	appGroup.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":             "欢迎使用 Weave 服务！",
			"version":             "1.0.0",
			"api_base":            "/api/v1",
			"health_check":        "/health",
			"available_endpoints": []string{"/api/v1/users", "/api/v1/tools", "/api/v1/plugins", "/health"},
		})
	})

	// 健康检查 - 健康检查控制器提供更全面的健康状态信息
	appGroup.GET("/health", healthCtrl.GetHealth)
	// 插件健康检查API
	appGroup.GET("/health/plugins/:name", healthCtrl.PluginHealthCheck)

	return router
}
