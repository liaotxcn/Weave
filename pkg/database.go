package pkg

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"weave/config"
	"weave/pkg/metrics"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// InitDatabase 初始化数据库连接
func InitDatabase() error {
	// 加载配置
	if config.Config.Database.Host == "" {
		if err := config.LoadConfig(); err != nil {
			Error("Failed to load config in InitDatabase", zap.Error(err))
			return err
		}
	}

	var dsn string
	var dialector gorm.Dialector

	// 根据数据库驱动类型构建连接字符串
	switch config.Config.Database.Driver {
	case "postgres":
		// PostgreSQL连接字符串
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
			config.Config.Database.Host,
			config.Config.Database.Port,
			config.Config.Database.Username,
			config.Config.Database.Password,
			config.Config.Database.DBName,
		)
		dialector = postgres.Open(dsn)
	case "mysql":
		fallthrough
	default:
		// MySQL连接字符串
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&timeout=10s&readTimeout=10s&writeTimeout=10s&collation=utf8mb4_unicode_ci&tls=false",
			config.Config.Database.Username,
			config.Config.Database.Password,
			config.Config.Database.Host,
			config.Config.Database.Port,
			config.Config.Database.DBName,
			config.Config.Database.Charset,
		)
		dialector = mysql.Open(dsn)
	}

	// 根据环境设置日志级别
	logLevel := logger.Error
	if config.Config.Logger.Development {
		logLevel = logger.Warn // 生产环境使用Warn级别
	}

	// 配置自定义日志器
	customLogger := logger.New(
		log.New(os.Stdout, "[gorm] ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             800 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  config.Config.Logger.Development,
		},
	)

	// 连接重试机制
	maxRetries := 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		// 创建带有性能监控的GORM配置
		gormConfig := &gorm.Config{
			Logger: customLogger,
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true, // 使用单数表名
			},
		}

		// 添加GORM性能监控插件
		DB, lastErr = gorm.Open(dialector, gormConfig)
		if lastErr == nil {
			// 记录连接建立指标
			metrics.RecordDatabaseQuery("connect", "system", 0)
			break
		}
		Debug("Database connection attempt failed, retrying...", zap.Int("attempt", i+1), zap.Int("max_attempts", maxRetries), zap.Error(lastErr))
		time.Sleep(1 * time.Second) // 等待一秒后重试
	}
	if lastErr != nil {
		return fmt.Errorf("failed to connect database after %d retries: %w", maxRetries, lastErr)
	}

	// 获取底层数据库连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// 设置优化连接池参数
	sqlDB.SetMaxIdleConns(5)                   // 减少空闲连接数，降低启动资源消耗
	sqlDB.SetMaxOpenConns(50)                  // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大生命周期
	sqlDB.SetConnMaxIdleTime(15 * time.Minute) // 添加连接最大空闲时间

	// 快速连接检查
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// 启动数据库连接监控（异步）
	go func() {
		// 根据环境配置监测时间间隔
		monitorInterval := 5 * time.Minute // 生产环境(5分钟)
		if config.Config.Logger.Development {
			monitorInterval = 1 * time.Minute // 开发环境(1分钟)
		}
		ticker := time.NewTicker(monitorInterval)
		for range ticker.C {
			stats := sqlDB.Stats()
			metrics.UpdateDatabaseConnections(stats.OpenConnections)
		}
	}()

	// 输出数据库连接成功日志
	dbType := "MySQL"
	if config.Config.Database.Driver == "postgres" {
		dbType = "PostgreSQL"
	}
	Info("Database connection established successfully", zap.String("type", dbType), zap.String("host", config.Config.Database.Host), zap.Int("port", config.Config.Database.Port), zap.String("database", config.Config.Database.DBName))
	return nil
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() error {
	// 使用默认上下文，5秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return CloseDatabaseWithContext(ctx)
}

// CloseDatabaseWithContext 使用上下文控制的方式优雅关闭数据库连接
func CloseDatabaseWithContext(ctx context.Context) error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// 记录关闭前的连接状态
	stats := sqlDB.Stats()
	Info("Starting database graceful shutdown", zap.Int("idle_connections", stats.Idle), zap.Int("open_connections", stats.OpenConnections))

	// 开始关闭过程
	startTime := time.Now()

	// 首先设置最大空闲连接为0，防止新的空闲连接创建
	sqlDB.SetMaxIdleConns(0)

	// 设置最大打开连接数为当前活跃连接数的估计值，允许现有连接完成但不接受新连接
	sqlDB.SetMaxOpenConns(stats.OpenConnections)

	// 关闭数据库连接
	err = sqlDB.Close()

	elapsed := time.Since(startTime)
	if err != nil {
		Error("Database connection close failed", zap.Duration("elapsed", elapsed), zap.Error(err))
		return fmt.Errorf("database close failed after %v: %w", elapsed, err)
	}

	Info("Database connections closed successfully", zap.Duration("elapsed", elapsed))

	// 清除全局DB变量
	DB = nil

	return nil
}
