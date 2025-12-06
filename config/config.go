package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
)

// Config 应用程序配置结构
var Config struct {
	// 服务器配置
	Server struct {
		Port       int
		InstanceID string // 实例标识，用于多实例部署
	}

	// 数据库配置
	Database struct {
		Driver   string
		Host     string
		Port     int
		Username string
		Password string
		DBName   string
		Charset  string
	}

	// 日志配置
	Logger struct {
		Level       string
		OutputPath  string
		ErrorPath   string
		Development bool
	}

	// JWT配置
	JWT struct {
		Secret             string
		AccessTokenExpiry  int // 访问令牌过期时间（分钟）
		RefreshTokenExpiry int // 刷新令牌过期时间（小时）
	}

	// CSRF配置
	CSRF struct {
		Enabled        bool
		CookieName     string
		HeaderName     string
		TokenLength    int
		CookieMaxAge   int // 秒
		CookiePath     string
		CookieDomain   string
		CookieSecure   bool
		CookieHttpOnly bool
		CookieSameSite string
	}

	// 数据库迁移配置
	AutoMigrate bool

	// 插件配置
	Plugins struct {
		Dir            string
		WatcherEnabled bool
		ScanInterval   int // 秒
		HotReload      bool
	}

	// Prometheus配置
	Prometheus struct {
		Enabled           bool
		MetricsPath       string
		EnableGoMetrics   bool
		EnableHTTPMetrics bool
	}

	// 邮件服务配置
	Email struct {
		SMTPServer string
		SMTPPort   int
		Username   string
		Password   string
		From       string
	}
}

// 重置默认配置到初始值
func resetDefaults() {
	// 服务器配置
	Config.Server.Port = 8081
	Config.Server.InstanceID = "weave-default"

	// 数据库配置（非敏感字段默认值）
	Config.Database.Driver = "mysql"
	Config.Database.Host = "localhost"
	Config.Database.Port = 3306
	Config.Database.DBName = "weave"
	Config.Database.Charset = "utf8mb4"
	// 敏感字段（数据库用户名和密码）将通过环境变量或配置文件设置
	Config.Database.Username = ""
	Config.Database.Password = ""

	// 日志配置
	Config.Logger.Level = "info"
	Config.Logger.OutputPath = "stdout"
	Config.Logger.ErrorPath = "stderr"
	Config.Logger.Development = false

	// JWT配置
	Config.JWT.Secret = ""                 // 敏感信息，将通过环境变量或配置文件设置
	Config.JWT.AccessTokenExpiry = 60      // 60分钟
	Config.JWT.RefreshTokenExpiry = 24 * 7 // 7天

	// CSRF配置
	Config.CSRF.Enabled = true
	Config.CSRF.CookieName = "XSRF-TOKEN"
	Config.CSRF.HeaderName = "X-CSRF-Token"
	Config.CSRF.TokenLength = 32
	Config.CSRF.CookieMaxAge = 3600 * 24 * 7 // 7天
	Config.CSRF.CookiePath = "/"
	Config.CSRF.CookieDomain = ""
	Config.CSRF.CookieSecure = false   // 开发环境下为false
	Config.CSRF.CookieHttpOnly = false // 必须为false以便前端可以读取
	Config.CSRF.CookieSameSite = "Lax"

	// 数据库迁移配置
	Config.AutoMigrate = true

	// 插件配置
	Config.Plugins.Dir = "./plugins"
	Config.Plugins.WatcherEnabled = true
	Config.Plugins.ScanInterval = 5 // 5秒
	Config.Plugins.HotReload = true

	// Prometheus配置
	Config.Prometheus.Enabled = true
	Config.Prometheus.MetricsPath = "/metrics"
	Config.Prometheus.EnableGoMetrics = true
	Config.Prometheus.EnableHTTPMetrics = true

	// 邮件服务配置默认值
	Config.Email.SMTPServer = "smtp.qq.com"
	Config.Email.SMTPPort = 587
	Config.Email.Username = ""
	Config.Email.Password = ""
	Config.Email.From = ""
}

func init() {
	resetDefaults()
}

// ValidateConfig 验证配置的有效性
func ValidateConfig() error {
	// 1. 检查必要的敏感配置项
	if Config.Database.Username == "" {
		return fmt.Errorf("数据库用户名未配置，请设置DB_USERNAME环境变量或在配置文件中指定")
	}

	if Config.Database.Password == "" {
		return fmt.Errorf("数据库密码未配置，请设置DB_PASSWORD环境变量或在配置文件中指定")
	}

	if Config.JWT.Secret == "" {
		return fmt.Errorf("JWT密钥未配置，请设置JWT_SECRET环境变量或在配置文件中指定")
	}

	// 2. 验证服务器配置
	if Config.Server.Port <= 0 || Config.Server.Port > 65535 {
		return fmt.Errorf("无效的服务器端口: %d，端口必须在1-65535之间", Config.Server.Port)
	}

	// 3. 验证数据库配置
	supportedDrivers := map[string]bool{"mysql": true, "postgres": true, "postgresql": true}
	if !supportedDrivers[Config.Database.Driver] {
		return fmt.Errorf("不支持的数据库驱动: %s，支持的驱动有: mysql, postgres, postgresql", Config.Database.Driver)
	}

	if Config.Database.Port <= 0 || Config.Database.Port > 65535 {
		return fmt.Errorf("无效的数据库端口: %d，端口必须在1-65535之间", Config.Database.Port)
	}

	if Config.Database.DBName == "" {
		return fmt.Errorf("数据库名称未配置")
	}

	// 4. 验证日志配置
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true, "fatal": true}
	if !validLogLevels[Config.Logger.Level] {
		return fmt.Errorf("无效的日志级别: %s，有效级别为: debug, info, warn, error, fatal", Config.Logger.Level)
	}

	// 5. 验证JWT配置
	if Config.JWT.AccessTokenExpiry <= 0 {
		return fmt.Errorf("无效的访问令牌过期时间: %d，必须大于0分钟", Config.JWT.AccessTokenExpiry)
	}

	if Config.JWT.RefreshTokenExpiry <= 0 {
		return fmt.Errorf("无效的刷新令牌过期时间: %d，必须大于0小时", Config.JWT.RefreshTokenExpiry)
	}

	// 6. 验证CSRF配置
	if Config.CSRF.TokenLength < 16 {
		return fmt.Errorf("CSRF令牌长度过小: %d，建议至少16个字符", Config.CSRF.TokenLength)
	}

	validSameSiteValues := map[string]bool{"Strict": true, "Lax": true, "None": true}
	if !validSameSiteValues[Config.CSRF.CookieSameSite] {
		return fmt.Errorf("无效的Cookie SameSite值: %s，有效值为: Strict, Lax, None", Config.CSRF.CookieSameSite)
	}

	// 7. 验证插件配置
	if Config.Plugins.Dir == "" {
		return fmt.Errorf("插件目录未配置")
	}

	// 检查插件目录是否存在
	if _, err := os.Stat(Config.Plugins.Dir); os.IsNotExist(err) {
		// 创建插件目录
		if err := os.MkdirAll(Config.Plugins.Dir, 0755); err != nil {
			return fmt.Errorf("创建插件目录失败: %w", err)
		}
	}

	if Config.Plugins.ScanInterval <= 0 {
		return fmt.Errorf("无效的插件扫描间隔: %d，必须大于0秒", Config.Plugins.ScanInterval)
	}

	// 8. 验证Prometheus配置
	if Config.Prometheus.MetricsPath != "" && Config.Prometheus.MetricsPath[0] != '/' {
		return fmt.Errorf("Prometheus指标路径必须以斜杠开头: %s", Config.Prometheus.MetricsPath)
	}

	return nil
}

// convertToBool 将interface{}转换为bool
func convertToBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	case int:
		return v != 0
	case float64:
		return v != 0
	}
	return false
}

// mapToPrometheusConfig 将map映射到Prometheus配置
func mapToPrometheusConfig(configMap map[string]interface{}) {
	if enabled, ok := configMap["enabled"]; ok {
		Config.Prometheus.Enabled = convertToBool(enabled)
	}
	if metricsPath, ok := configMap["metricsPath"].(string); ok {
		Config.Prometheus.MetricsPath = metricsPath
	}
	if enableGoMetrics, ok := configMap["enableGoMetrics"]; ok {
		Config.Prometheus.EnableGoMetrics = convertToBool(enableGoMetrics)
	}
	if enableHTTPMetrics, ok := configMap["enableHTTPMetrics"]; ok {
		Config.Prometheus.EnableHTTPMetrics = convertToBool(enableHTTPMetrics)
	}
}

// SanitizeConfig 清理配置中的敏感信息，用于日志输出
func SanitizeConfig() map[string]interface{} {
	// 创建配置的安全副本用于日志输出
	sanitized := map[string]interface{}{
		"Server": map[string]interface{}{
			"Port": Config.Server.Port,
		},
		"Database": map[string]interface{}{
			"Driver":   Config.Database.Driver,
			"Host":     Config.Database.Host,
			"Port":     Config.Database.Port,
			"Username": Config.Database.Username,
			"Password": "***", // 隐藏密码
			"DBName":   Config.Database.DBName,
			"Charset":  Config.Database.Charset,
		},
		"Logger": map[string]interface{}{
			"Level":       Config.Logger.Level,
			"OutputPath":  Config.Logger.OutputPath,
			"ErrorPath":   Config.Logger.ErrorPath,
			"Development": Config.Logger.Development,
		},
		"JWT": map[string]interface{}{
			"Secret":             "***", // 隐藏密钥
			"AccessTokenExpiry":  Config.JWT.AccessTokenExpiry,
			"RefreshTokenExpiry": Config.JWT.RefreshTokenExpiry,
		},
		"CSRF": map[string]interface{}{
			"Enabled":        Config.CSRF.Enabled,
			"CookieName":     Config.CSRF.CookieName,
			"HeaderName":     Config.CSRF.HeaderName,
			"TokenLength":    Config.CSRF.TokenLength,
			"CookieMaxAge":   Config.CSRF.CookieMaxAge,
			"CookiePath":     Config.CSRF.CookiePath,
			"CookieDomain":   Config.CSRF.CookieDomain,
			"CookieSecure":   Config.CSRF.CookieSecure,
			"CookieHttpOnly": Config.CSRF.CookieHttpOnly,
			"CookieSameSite": Config.CSRF.CookieSameSite,
		},
		"AutoMigrate": Config.AutoMigrate,
		"Plugins": map[string]interface{}{
			"Dir":            Config.Plugins.Dir,
			"WatcherEnabled": Config.Plugins.WatcherEnabled,
			"ScanInterval":   Config.Plugins.ScanInterval,
			"HotReload":      Config.Plugins.HotReload,
		},
		"Prometheus": map[string]interface{}{
			"Enabled":           Config.Prometheus.Enabled,
			"MetricsPath":       Config.Prometheus.MetricsPath,
			"EnableGoMetrics":   Config.Prometheus.EnableGoMetrics,
			"EnableHTTPMetrics": Config.Prometheus.EnableHTTPMetrics,
		},
	}

	return sanitized
}

// GetAbsConfigFilePath 获取配置文件的绝对路径
func GetAbsConfigFilePath() (string, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/config.yaml"
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

// LoadConfigWithViper 使用Viper从配置文件和环境变量加载配置
func LoadConfigWithViper() error {
	// 在每次加载前重置默认值，避免跨测试用例状态污染
	resetDefaults()

	// 创建Viper实例
	v := viper.New()

	// 设置默认值
	v.SetDefault("server.port", Config.Server.Port)
	v.SetDefault("server.instanceID", Config.Server.InstanceID)
	v.SetDefault("database.driver", Config.Database.Driver)
	v.SetDefault("database.host", Config.Database.Host)
	v.SetDefault("database.port", Config.Database.Port)
	v.SetDefault("database.dbname", Config.Database.DBName)
	v.SetDefault("database.charset", Config.Database.Charset)
	v.SetDefault("logger.level", Config.Logger.Level)
	v.SetDefault("logger.outputPath", Config.Logger.OutputPath)
	v.SetDefault("logger.errorPath", Config.Logger.ErrorPath)
	v.SetDefault("logger.development", Config.Logger.Development)
	v.SetDefault("jwt.accessTokenExpiry", Config.JWT.AccessTokenExpiry)
	v.SetDefault("jwt.refreshTokenExpiry", Config.JWT.RefreshTokenExpiry)
	v.SetDefault("csrf.enabled", Config.CSRF.Enabled)
	v.SetDefault("csrf.cookieName", Config.CSRF.CookieName)
	v.SetDefault("csrf.headerName", Config.CSRF.HeaderName)
	v.SetDefault("csrf.tokenLength", Config.CSRF.TokenLength)
	v.SetDefault("csrf.cookieMaxAge", Config.CSRF.CookieMaxAge)
	v.SetDefault("csrf.cookiePath", Config.CSRF.CookiePath)
	v.SetDefault("csrf.cookieDomain", Config.CSRF.CookieDomain)
	v.SetDefault("csrf.cookieSecure", Config.CSRF.CookieSecure)
	v.SetDefault("csrf.cookieHttpOnly", Config.CSRF.CookieHttpOnly)
	v.SetDefault("csrf.cookieSameSite", Config.CSRF.CookieSameSite)
	v.SetDefault("autoMigrate", Config.AutoMigrate)
	v.SetDefault("plugins.dir", Config.Plugins.Dir)
	v.SetDefault("plugins.watcherEnabled", Config.Plugins.WatcherEnabled)
	v.SetDefault("plugins.scanInterval", Config.Plugins.ScanInterval)
	v.SetDefault("plugins.hotReload", Config.Plugins.HotReload)
	v.SetDefault("prometheus.enabled", Config.Prometheus.Enabled)
	v.SetDefault("prometheus.metricsPath", Config.Prometheus.MetricsPath)
	v.SetDefault("prometheus.enableGoMetrics", Config.Prometheus.EnableGoMetrics)
	v.SetDefault("prometheus.enableHTTPMetrics", Config.Prometheus.EnableHTTPMetrics)
	v.SetDefault("email.smtpServer", Config.Email.SMTPServer)
	v.SetDefault("email.smtpPort", Config.Email.SMTPPort)

	// 配置文件路径
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/config.yaml"
	}

	// 设置配置文件
	v.SetConfigFile(configPath)

	// 设置环境变量前缀
	v.SetEnvPrefix("WEAVE")
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 如果配置文件不存在，只记录警告，不返回错误
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 映射配置到结构体
	Config.Server.Port = v.GetInt("server.port")
	Config.Server.InstanceID = v.GetString("server.instanceID")
	Config.Database.Driver = v.GetString("database.driver")
	Config.Database.Host = v.GetString("database.host")
	Config.Database.Port = v.GetInt("database.port")
	Config.Database.Username = v.GetString("database.username")
	Config.Database.Password = v.GetString("database.password")
	Config.Database.DBName = v.GetString("database.dbname")
	Config.Database.Charset = v.GetString("database.charset")
	Config.Logger.Level = v.GetString("logger.level")
	Config.Logger.OutputPath = v.GetString("logger.outputPath")
	Config.Logger.ErrorPath = v.GetString("logger.errorPath")
	Config.Logger.Development = v.GetBool("logger.development")
	Config.JWT.Secret = v.GetString("jwt.secret")
	Config.JWT.AccessTokenExpiry = v.GetInt("jwt.accessTokenExpiry")
	Config.JWT.RefreshTokenExpiry = v.GetInt("jwt.refreshTokenExpiry")
	Config.CSRF.Enabled = v.GetBool("csrf.enabled")
	Config.CSRF.CookieName = v.GetString("csrf.cookieName")
	Config.CSRF.HeaderName = v.GetString("csrf.headerName")
	Config.CSRF.TokenLength = v.GetInt("csrf.tokenLength")
	Config.CSRF.CookieMaxAge = v.GetInt("csrf.cookieMaxAge")
	Config.CSRF.CookiePath = v.GetString("csrf.cookiePath")
	Config.CSRF.CookieDomain = v.GetString("csrf.cookieDomain")
	Config.CSRF.CookieSecure = v.GetBool("csrf.cookieSecure")
	Config.CSRF.CookieHttpOnly = v.GetBool("csrf.cookieHttpOnly")
	Config.CSRF.CookieSameSite = v.GetString("csrf.cookieSameSite")
	Config.AutoMigrate = v.GetBool("autoMigrate")
	Config.Plugins.Dir = v.GetString("plugins.dir")
	Config.Plugins.WatcherEnabled = v.GetBool("plugins.watcherEnabled")
	Config.Plugins.ScanInterval = v.GetInt("plugins.scanInterval")
	Config.Plugins.HotReload = v.GetBool("plugins.hotReload")
	Config.Prometheus.Enabled = v.GetBool("prometheus.enabled")
	Config.Prometheus.MetricsPath = v.GetString("prometheus.metricsPath")
	Config.Prometheus.EnableGoMetrics = v.GetBool("prometheus.enableGoMetrics")
	Config.Prometheus.EnableHTTPMetrics = v.GetBool("prometheus.enableHTTPMetrics")
	Config.Email.SMTPServer = v.GetString("email.smtpServer")
	Config.Email.SMTPPort = v.GetInt("email.smtpPort")
	Config.Email.Username = v.GetString("email.username")
	Config.Email.Password = v.GetString("email.password")
	Config.Email.From = v.GetString("email.from")

	// 验证配置
	return ValidateConfig()
}

// LoadConfig 从配置文件和环境变量加载配置
func LoadConfig() error {
	// 使用Viper加载配置
	return LoadConfigWithViper()
}
