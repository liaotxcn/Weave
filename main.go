package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"weave/config"
	"weave/controllers"
	"weave/middleware"
	"weave/models"
	"weave/pkg"
	"weave/pkg/migrate/migration"
	"weave/plugins"
	"weave/plugins/examples"
	fc "weave/plugins/features/FormatConverter"
	note "weave/plugins/features/Note"
	"weave/routers"
	"weave/services/health"
	"weave/services/tool"
	"weave/services/user"
	"weave/services/audit"
	"weave/services/team"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func loadEnvFile(filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			pkg.Warn(".env file not found", zap.String("path", filePath))
		} else {
			pkg.Warn("Failed to read .env file", zap.Error(err))
		}
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	successCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}

		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[0] == value[len(value)-1] {
				value = value[1 : len(value)-1]
			}

			if err := os.Setenv(key, value); err != nil {
				pkg.Warn("Failed to set environment variable", zap.String("key", key), zap.Error(err))
			} else {
				successCount++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		pkg.Warn("Error reading .env file", zap.Error(err))
	}
	pkg.Info(".env file loaded", zap.Int("variables", successCount))
}

func main() {
	// 加载.env 配置文件
	loadEnvFile(".env")

	// 初始化日志系统
	if err := pkg.InitLogger(pkg.Options{
		Level:       config.Config.Logger.Level,
		OutputPath:  config.Config.Logger.OutputPath,
		ErrorPath:   config.Config.Logger.ErrorPath,
		Development: config.Config.Logger.Development,
	}); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer pkg.Sync()

	// 设置PluginManager的日志记录器
	plugins.PluginManager.SetLogger(pkg.GetLogger())

	// 加载配置
	if err := config.LoadConfig(); err != nil {
		pkg.Fatal("Failed to load configuration", zap.Error(err))
	}

	// 输出清理后的配置信息（隐藏敏感数据）
	pkg.Info("Configuration loaded successfully", zap.Any("config", config.SanitizeConfig()))

	// 验证配置完整性（确保所有配置项都经过验证）
	if err := config.ValidateConfig(); err != nil {
		pkg.Fatal("Configuration validation failed", zap.Error(err))
	}
	pkg.Info("Configuration validation passed successfully")

	// 初始化数据库（优化连接参数）
	if err := pkg.InitDatabase(); err != nil {
		pkg.Fatal("Failed to initialize database", zap.Error(err))
	}
	pkg.Info("Database initialized successfully")

	// 数据库迁移（异步）
	go func() {
		if !config.Config.AutoMigrate {
			pkg.Info("Starting SQL migrations...")
			mm := migration.NewMigrationManager()
			if err := mm.Init(); err != nil {
				pkg.Warn("Failed to initialize migration manager", zap.Error(err))
			} else {
				if err := mm.Up(); err != nil {
					pkg.Warn("Migration errors", zap.Error(err))
				} else {
					pkg.Info("SQL migrations completed successfully")
				}
			}
		} else {
			// 仅当启用自动迁移时才使用GORM自动迁移
			pkg.Info("Starting GORM auto-migration...")
			if err := models.MigrateTables(pkg.DB); err != nil {
				pkg.Warn("Failed to migrate database tables", zap.Error(err))
			} else {
				pkg.Info("GORM auto-migration completed successfully")
			}
		}
	}()

	// 创建Service实例
	userSvc := user.NewUserService(pkg.DB, user.EmailConfig{
		SMTPServer: config.Config.Email.SMTPServer,
		SMTPPort:   config.Config.Email.SMTPPort,
		Username:   config.Config.Email.Username,
		Password:   config.Config.Email.Password,
		From:       config.Config.Email.From,
	})
	teamSvc := team.NewTeamService(pkg.DB)
	auditSvc := audit.NewAuditService(pkg.DB)
	toolSvc := tool.NewToolService(pkg.DB)
	healthSvc := health.NewHealthService(pkg.DB)

	// 创建Controller实例
	userCtrl := controllers.NewUserController(userSvc)
	teamCtrl := controllers.NewTeamController(teamSvc)
	auditCtrl := controllers.NewAuditController(auditSvc)
	toolCtrl := controllers.NewToolController(toolSvc)
	healthCtrl := controllers.NewHealthController(healthSvc)
	pluginCtrl := controllers.NewPluginController()
	// 初始化路由
	router := routers.SetupRouter(userCtrl, teamCtrl, auditCtrl, toolCtrl, healthCtrl, pluginCtrl)

	// 添加错误处理中间件
	errHandler := middleware.NewErrorHandler()
	router.Use(errHandler.HandlerFunc())

	// 注册插件
	registerPlugins(router)

	// 初始化插件系统
	if err := plugins.InitPluginSystem(); err != nil {
		pkg.Error("Failed to initialize plugin system", zap.Error(err))
	}

	// 启动服务器
	port := config.Config.Server.Port
	instanceID := config.Config.Server.InstanceID

	// 创建HTTP服务器并配置连接复用参数
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        router,
		ReadTimeout:    15 * time.Second, // 请求读取超时时间
		WriteTimeout:   15 * time.Second, // 响应写入超时时间
		IdleTimeout:    60 * time.Second, // 空闲连接超时时间（影响Keep-Alive）
		MaxHeaderBytes: 1 << 20,          // 最大请求头大小（1MB）
	}

	go func() {
		pkg.Info("Weave 服务启动成功",
			zap.String("instance_id", instanceID),
			zap.String("address", fmt.Sprintf("http://localhost:%d", port)))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			pkg.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待中断信号优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	pkg.Info("Shutting down server...")

	// 停止插件监控器
	plugins.PluginManager.StopPluginWatcher()

	// 创建超时上下文，用于优雅关闭服务器和数据库
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 先关闭HTTP服务器
	if err := srv.Shutdown(ctx); err != nil {
		pkg.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// 然后使用相同上下文优雅关闭数据库连接
	// 确保数据库连接在服务器停止接收新请求后有足够时间完成正在进行的操作
	if err := pkg.CloseDatabaseWithContext(ctx); err != nil {
		pkg.Error("Database shutdown error", zap.Error(err))
	}

	pkg.Info("Server exiting")
}

// 注册插件
func registerPlugins(router *gin.Engine) {
	// 设置路由引擎到PluginManager
	plugins.PluginManager.SetRouter(router)

	// 注册Hello插件
	helloPlugin := &examples.HelloPlugin{}
	if err := plugins.PluginManager.Register(helloPlugin); err != nil {
		pkg.Error("Failed to register plugin", zap.String("plugin", helloPlugin.Name()), zap.Error(err))
	} else {
		pkg.Info("Successfully registered plugin", zap.String("plugin", helloPlugin.Name()))
	}

	// 注册Note插件
	notePlugin := &note.NotePlugin{}
	if err := plugins.PluginManager.Register(notePlugin); err != nil {
		pkg.Error("Failed to register plugin", zap.String("plugin", notePlugin.Name()), zap.Error(err))
	} else {
		pkg.Info("Successfully registered plugin", zap.String("plugin", notePlugin.Name()))
	}

	// 注册FormatConverter插件
	formatConverter := &fc.FormatConverterPlugin{}
	if err := plugins.PluginManager.Register(formatConverter); err != nil {
		pkg.Error("Failed to register plugin", zap.String("plugin", formatConverter.Name()), zap.Error(err))
	} else {
		pkg.Info("Successfully registered plugin", zap.String("plugin", formatConverter.Name()))
	}

	// 统一注册所有插件路由（可选）
	// if err := plugins.PluginManager.RegisterAllRoutes(); err != nil {
	// 	pkg.Error("Failed to register all plugin routes", zap.Error(err))
	// }

	// 注册优化插件
	sampleOptimizedPlugin := examples.NewSampleOptimizedPlugin()
	if err := plugins.PluginManager.Register(sampleOptimizedPlugin); err != nil {
		pkg.Error("Failed to register plugin", zap.String("plugin", sampleOptimizedPlugin.Name()), zap.Error(err))
	} else {
		pkg.Info("Successfully registered plugin", zap.String("plugin", sampleOptimizedPlugin.Name()))
	}

	// 注册依赖插件
	sampleDependentPlugin := examples.NewSampleDependentPlugin()
	if err := plugins.PluginManager.Register(sampleDependentPlugin); err != nil {
		pkg.Error("Failed to register plugin", zap.String("plugin", sampleDependentPlugin.Name()), zap.Error(err))
	} else {
		pkg.Info("Successfully registered plugin", zap.String("plugin", sampleDependentPlugin.Name()))
	}

	// 所有插件注册完成，输出确认日志
	pkg.Info("插件已全部注册运行成功")
}
