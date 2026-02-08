package email

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/big"
	"net/smtp"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"weave/models"
	"weave/pkg"
	"weave/utils"

	"gorm.io/gorm"
)

// EmailConfig 邮件服务器配置
type EmailConfig struct {
	SMTPServer string
	SMTPPort   int
	Username   string
	Password   string
	From       string
}

// EmailService 邮件服务
type EmailService struct {
	config EmailConfig
}

// NewEmailService 创建新的邮件服务实例
func NewEmailService(config EmailConfig) *EmailService {
	return &EmailService{config: config}
}

// isValidEmail 验证邮箱地址格式是否正确
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// GenerateVerificationCode 生成6位数字验证码
func (s *EmailService) GenerateVerificationCode() (string, error) {
	result := make([]string, 6)
	for i := 0; i < 6; i++ {
		// 生成0-9之间的随机数
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = strconv.Itoa(int(num.Int64()))
	}
	return strings.Join(result, ""), nil
}

// SendVerificationCode 发送验证码到指定邮箱
func (s *EmailService) SendVerificationCode(email, code string) error {
	// 验证邮箱地址格式
	if !isValidEmail(email) {
		return fmt.Errorf("invalid email address format")
	}

	subject := "Weave 验证码"

	// 对验证码进行HTML转义，防止XSS风险
	escapedCode := template.HTMLEscapeString(code)

	// 读取邮件模板
	body, _ := s.loadEmailTemplate(escapedCode)

	return s.SendEmail(email, subject, body)
}

// loadEmailTemplate 加载邮件模板
func (s *EmailService) loadEmailTemplate(code string) (string, error) {
	// 模板文件路径
	templatePath := filepath.Join("services", "email", "email.html")

	// 读取模板文件
	content, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	// 替换模板中的验证码占位符
	templateContent := string(content)
	templateContent = strings.Replace(templateContent, "{{.Code}}", code, -1)

	return templateContent, nil
}

// SendEmail 发送邮件
func (s *EmailService) SendEmail(to, subject, body string) error {
	// 构建邮件头
	header := make(map[string]string)
	header["From"] = s.config.From
	header["To"] = to
	header["Subject"] = "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	// 构建邮件内容
	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// 连接SMTP服务器
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.SMTPServer)
	serverAddr := fmt.Sprintf("%s:%d", s.config.SMTPServer, s.config.SMTPPort)

	return smtp.SendMail(serverAddr, auth, s.config.From, []string{to}, []byte(message))
}

// CreateVerificationCode 创建并保存验证码记录
func (s *EmailService) CreateVerificationCode(email string, tenantID uint) (string, *models.EmailVerificationCode, error) {
	// 生成验证码
	code, err := s.GenerateVerificationCode()
	if err != nil {
		return "", nil, err
	}

	// 对验证码进行哈希处理
	hashedCode, err := utils.HashPassword(code)
	if err != nil {
		return "", nil, err
	}

	// 创建验证码记录
	verificationCode := &models.EmailVerificationCode{
		Email:     email,
		Code:      hashedCode,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute), // 5分钟有效期
		Used:      false,
		TenantID:  tenantID,
	}

	return code, verificationCode, nil
}

// VerifyCode 验证邮箱验证码
func (s *EmailService) VerifyCode(email, code string, tenantID uint) (bool, error) {
	// 查找最新的未使用且未过期的验证码记录
	var verificationCode models.EmailVerificationCode
	result := pkg.DB.Where("email = ? AND used = false AND expires_at > ? AND tenant_id = ?",
		email, time.Now(), tenantID).Order("created_at DESC").First(&verificationCode)

	if result.Error != nil {
		return false, result.Error
	}

	// 验证验证码是否匹配
	if !utils.CheckPasswordHash(code, verificationCode.Code) {
		return false, fmt.Errorf("invalid verification code")
	}

	// 标记验证码为已使用
	verificationCode.Used = true
	if err := pkg.DB.Save(&verificationCode).Error; err != nil {
		return false, err
	}

	return true, nil
}

// GetLastVerificationTime 获取用户最近一次获取验证码的时间
func (s *EmailService) GetLastVerificationTime(email string, tenantID uint) (time.Time, error) {
	var verificationCode models.EmailVerificationCode
	result := pkg.DB.Where("email = ? AND tenant_id = ?", email, tenantID).Order("created_at DESC").First(&verificationCode)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// 如果没有找到记录，返回零时间
			return time.Time{}, nil
		}
		return time.Time{}, result.Error
	}

	return verificationCode.CreatedAt, nil
}

// CheckRateLimit 检查获取验证码的频率限制
func (s *EmailService) CheckRateLimit(email string, tenantID uint) (bool, error) {
	lastTime, err := s.GetLastVerificationTime(email, tenantID)
	if err != nil {
		return false, err
	}

	// 若距离上次发送验证码不足60秒，则限制再次发送
	if !lastTime.IsZero() && time.Since(lastTime) < 60*time.Second {
		return false, nil
	}

	// 检查24小时内的总发送次数
	var count int64
	pkg.DB.Model(&models.EmailVerificationCode{}).
		Where("email = ? AND tenant_id = ? AND created_at > ?",
			email, tenantID, time.Now().Add(-24*time.Hour)).
		Count(&count)

	if count >= 15 {
		return false, nil
	}

	return true, nil
}
