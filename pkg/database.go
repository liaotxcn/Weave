package pkg

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"weave/config"
	"weave/pkg/metrics"

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
		config.LoadConfig()
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
	logLevel := logger.Info
	if !config.Config.Logger.Development {
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
		log.Printf("Database connection attempt failed, retrying... attempt=%d/%d, error=%v",
			i+1, maxRetries, lastErr)
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
			idle := stats.Idle
			open := stats.OpenConnections
			metrics.UpdateDatabaseConnections(open)
			log.Printf("Database connection stats: idle=%d, open=%d", idle, open)
		}
	}()

	// 输出数据库连接成功日志
	dbType := "MySQL"
	if config.Config.Database.Driver == "postgres" {
		dbType = "PostgreSQL"
	}
	log.Printf("数据库连接成功 - database_type: %s, driver: %s, host: %s, port: %d, database: %s",
		dbType,
		config.Config.Database.Driver,
		config.Config.Database.Host,
		config.Config.Database.Port,
		config.Config.Database.DBName,
	)
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
// 支持超时控制，等待正在进行的事务完成
func CloseDatabaseWithContext(ctx context.Context) error {
	if DB == nil {
		log.Printf("Database connection already closed or not initialized")
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Failed to get database instance during shutdown: %v", err)
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// 记录关闭前的连接状态
	stats := sqlDB.Stats()
	log.Printf("Starting database graceful shutdown - idle_connections: %d, open_connections: %d, in_use: %d, idle_closed: %d",
		stats.Idle,
		stats.OpenConnections,
		stats.InUse,
		stats.MaxIdleClosed,
	)

	// 开始关闭过程
	startTime := time.Now()

	// 首先设置最大空闲连接为0，防止新的空闲连接创建
	sqlDB.SetMaxIdleConns(0)

	// 设置最大打开连接数为当前活跃连接数的估计值，允许现有连接完成但不接受新连接
	// 注意：Go的sql.DBStats没有Active字段，我们使用OpenConnections作为上限
	sqlDB.SetMaxOpenConns(stats.OpenConnections)

	log.Printf("Waiting for active database connections to complete")

	// 关闭数据库连接
	err = sqlDB.Close()

	elapsed := time.Since(startTime)
	if err != nil {
		log.Printf("Database connection close failed after %v: %v", elapsed, err)
		return fmt.Errorf("database close failed after %v: %w", elapsed, err)
	}

	log.Printf("Database connections closed successfully after %v", elapsed)

	// 清除全局DB变量
	DB = nil

	return nil
}
