package controllers

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"weave/config"
	"weave/models"
	"weave/pkg"
	"weave/services/email"
	"weave/utils"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
type UserController struct {
	emailService *email.EmailService
}

// NewUserController 创建用户控制器实例
func NewUserController() *UserController {
	// 从应用配置中加载邮件服务配置
	emailConfig := email.EmailConfig{
		SMTPServer: config.Config.Email.SMTPServer,
		SMTPPort:   config.Config.Email.SMTPPort,
		Username:   config.Config.Email.Username,
		Password:   config.Config.Email.Password,
		From:       config.Config.Email.From,
	}
	return &UserController{
		emailService: email.NewEmailService(emailConfig),
	}
}

// Register 用户注册
func (uc *UserController) Register(c *gin.Context) {
	// 定义注册请求结构体
	var registerRequest struct {
		Username        string `json:"username" binding:"required,min=3,max=50"`
		Password        string `json:"password" binding:"required,min=6"`
		ConfirmPassword string `json:"confirm_password" binding:"required,min=6"`
		Email           string `json:"email" binding:"required,email"`
	}

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		err := pkg.NewValidationError("Invalid registration data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 检查两次输入的密码是否一致
	if registerRequest.Password != registerRequest.ConfirmPassword {
		err := pkg.NewValidationError("Passwords do not match", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 检查用户名是否已存在
	var existingUser models.User
	result := pkg.DB.Where("username = ?", registerRequest.Username).First(&existingUser)
	if result.Error == nil {
		err := pkg.NewConflictError("Username already exists", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 检查邮箱是否已存在
	result = pkg.DB.Where("email = ?", registerRequest.Email).First(&existingUser)
	if result.Error == nil {
		err := pkg.NewConflictError("Email already registered", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 对密码进行哈希处理
	passwordHash, err := utils.HashPassword(registerRequest.Password)
	if err != nil {
		err := pkg.NewInternalError("Failed to encrypt password", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 创建新用户
	newUser := models.User{
		Username: registerRequest.Username,
		Password: passwordHash,
		Email:    registerRequest.Email,
	}

	result = pkg.DB.Create(&newUser)
	if result.Error != nil {
		dbErr := pkg.NewDatabaseError("Failed to register user", result.Error)
		dbErr.WithDetails(map[string]interface{}{
			"username": registerRequest.Username,
			"email":    registerRequest.Email,
		})
		c.JSON(pkg.GetHTTPStatus(dbErr), gin.H{"code": string(dbErr.Code), "message": dbErr.Message})
		return
	}

	// 不返回密码信息
	newUser.Password = ""
	c.JSON(http.StatusCreated, gin.H{"message": "注册成功", "user": newUser})
}

// SendVerificationCodeRequest 发送验证码请求结构
type SendVerificationCodeRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"omitempty,email"` // 保留兼容性
}

// LoginWithCodeRequest 验证码登录请求结构
type LoginWithCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

// SendVerificationCode 发送邮箱验证码
func (uc *UserController) SendVerificationCode(c *gin.Context) {
	var req SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("请输入有效的用户名", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 获取租户ID
	tenantID := c.GetUint("tenant_id")

	// 检查是否有权限（这里假设未登录用户也可以获取验证码，只是需要租户ID）
	// 在实际应用中，可能需要更复杂的权限控制

	// 查找用户是否存在
	var user models.User
	result := pkg.DB.Where("username = ? AND tenant_id = ?", req.Username, tenantID).First(&user)
	if result.Error != nil {
		err := pkg.NewNotFoundError("用户不存在", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 使用用户注册时的邮箱
	userEmail := user.Email

	// 检查频率限制
	canSend, err := uc.emailService.CheckRateLimit(userEmail, tenantID)
	if err != nil {
		err := pkg.NewInternalError("Failed to check rate limit", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	if !canSend {
		err := pkg.NewValidationError("验证码发送过于频繁，请稍后再试", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 创建验证码记录
	originalCode, verificationCode, err := uc.emailService.CreateVerificationCode(userEmail, tenantID)
	if err != nil {
		err := pkg.NewInternalError("Failed to create verification code", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 保存验证码到数据库
	if err := pkg.DB.Create(verificationCode).Error; err != nil {
		err := pkg.NewDatabaseError("Failed to save verification code", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 异步发送邮件
	go func() {
		// 发送验证码邮件，使用原始验证码
		err := uc.emailService.SendVerificationCode(userEmail, originalCode)
		if err != nil {
			// 记录发送失败日志，但不影响用户体验
			fmt.Printf("Failed to send verification email to %s: %v\n", userEmail, err)
		}
	}()

	// 记录操作日志
	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "send_verification_code",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", user.ID),
		OldValue:     nil,
		NewValue: map[string]interface{}{
			"username": req.Username,
			"email":    userEmail,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送到您的邮箱，请查收"})
}

// LoginWithVerificationCode 使用邮箱验证码登录
func (uc *UserController) LoginWithVerificationCode(c *gin.Context) {
	var req LoginWithCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 记录绑定失败的登录尝试
		recordLoginHistory(req.Email, c.ClientIP(), c.Request.UserAgent(), false, "请求参数验证失败: "+err.Error(), 0)
		err := pkg.NewValidationError("Invalid request data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 获取租户ID
	tenantID := c.GetUint("tenant_id")

	// 验证验证码
	isValid, err := uc.emailService.VerifyCode(req.Email, req.Code, tenantID)
	if err != nil {
		// 记录验证失败的登录尝试
		recordLoginHistory(req.Email, c.ClientIP(), c.Request.UserAgent(), false, "验证码验证失败: "+err.Error(), tenantID)
		err := pkg.NewAuthError("验证码错误或已过期", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	if !isValid {
		// 记录验证码无效的登录尝试
		recordLoginHistory(req.Email, c.ClientIP(), c.Request.UserAgent(), false, "验证码无效", tenantID)
		err := pkg.NewAuthError("验证码错误或已过期", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 查找用户
	var user models.User
	result := pkg.DB.Where("email = ? AND tenant_id = ?", req.Email, tenantID).First(&user)
	if result.Error != nil {
		// 记录用户不存在的登录尝试
		recordLoginHistory(req.Email, c.ClientIP(), c.Request.UserAgent(), false, "用户不存在", tenantID)
		err := pkg.NewAuthError("User not found", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 生成访问令牌和刷新令牌
	accessToken, err := utils.GenerateToken(user.ID, user.TenantID)
	if err != nil {
		// 记录生成token失败的情况
		recordLoginHistory(req.Email, c.ClientIP(), c.Request.UserAgent(), false, "生成访问令牌失败: "+err.Error(), user.TenantID)
		err := pkg.NewInternalError("Failed to generate access token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.TenantID)
	if err != nil {
		// 记录生成刷新令牌失败的情况
		recordLoginHistory(req.Email, c.ClientIP(), c.Request.UserAgent(), false, "生成刷新令牌失败: "+err.Error(), user.TenantID)
		err := pkg.NewInternalError("Failed to generate refresh token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 记录登录成功
	recordLoginHistory(req.Email, c.ClientIP(), c.Request.UserAgent(), true, "邮箱验证码登录成功", user.TenantID)

	// 记录登录操作的审计日志
	loginUser := user
	loginUser.Password = "[REDACTED]"
	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "login_with_code",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", user.ID),
		OldValue:     nil,
		NewValue: map[string]interface{}{
			"email":      user.Email,
			"ip_address": c.ClientIP(),
			"success":    true,
		},
	})

	// 不返回密码信息
	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"message": "登录成功", "access_token": accessToken, "refresh_token": refreshToken, "user": user})
}

// Login 用户登录（需要用户名、密码和邮箱验证码，邮箱从用户注册信息中获取）
func (uc *UserController) Login(c *gin.Context) {
	// 定义登录请求结构体
	var loginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required,len=6"`
	}

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		// 记录绑定失败的登录尝试
		recordLoginHistory(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), false, "请求参数验证失败: "+err.Error(), 0)
		err := pkg.NewValidationError("请输入用户名、密码和验证码", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 获取租户ID
	tenantID := c.GetUint("tenant_id")

	// 查找用户
	var user models.User
	result := pkg.DB.Where("username = ? AND tenant_id = ?", loginRequest.Username, tenantID).First(&user)
	if result.Error != nil {
		// 记录用户不存在的登录尝试
		recordLoginHistory(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), false, "用户名或密码错误", tenantID)
		err := pkg.NewAuthError("用户名或密码错误", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 验证密码
	if !utils.CheckPasswordHash(loginRequest.Password, user.Password) {
		// 记录密码错误的登录尝试
		recordLoginHistory(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), false, "用户名或密码错误", user.TenantID)
		err := pkg.NewAuthError("用户名或密码错误", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 验证邮箱验证码（使用用户注册邮箱）
	isValid, err := uc.emailService.VerifyCode(user.Email, loginRequest.Code, tenantID)
	if err != nil || !isValid {
		// 记录验证码验证失败的登录尝试
		recordLoginHistory(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), false, "验证码验证失败: "+err.Error(), user.TenantID)
		err := pkg.NewAuthError("验证码错误或已过期", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 生成访问令牌和刷新令牌（包含tenant_id）
	accessToken, err := utils.GenerateToken(user.ID, user.TenantID)
	if err != nil {
		// 记录生成token失败的情况
		recordLoginHistory(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), false, "生成访问令牌失败: "+err.Error(), user.TenantID)
		err := pkg.NewInternalError("Failed to generate access token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.TenantID)
	if err != nil {
		// 记录生成刷新令牌失败的情况
		recordLoginHistory(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), false, "生成刷新令牌失败: "+err.Error(), user.TenantID)
		err := pkg.NewInternalError("Failed to generate refresh token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 记录登录成功
	recordLoginHistory(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), true, "登录成功", user.TenantID)

	// 记录登录操作的审计日志
	loginUser := user
	loginUser.Password = "[REDACTED]"

	// 记录多因素认证登录的审计日志
	action := "login_multi_factor"
	newValue := map[string]interface{}{
		"username":   user.Username,
		"email":      user.Email,
		"ip_address": c.ClientIP(),
		"success":    true,
	}

	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       action,
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", user.ID),
		OldValue:     nil,
		NewValue:     newValue,
	})

	// 不返回密码信息
	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"message": "登录成功", "access_token": accessToken, "refresh_token": refreshToken, "user": user})
}

// RefreshToken 刷新访问令牌
func (uc *UserController) RefreshToken(c *gin.Context) {
	// 定义刷新令牌请求结构体
	var refreshRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(&refreshRequest); err != nil {
		err := pkg.NewValidationError("Refresh token is required", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 验证刷新令牌（获取userID与tenantID）
	userID, tenantID, err := utils.VerifyRefreshToken(refreshRequest.RefreshToken)
	if err != nil {
		err := pkg.NewAuthError("Invalid refresh token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 查找用户
	var user models.User
	result := pkg.DB.First(&user, userID)
	if result.Error != nil {
		err := pkg.NewNotFoundError("User not found", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 生成新的访问令牌（保持相同tenant_id）
	accessToken, err := utils.GenerateToken(userID, tenantID)
	if err != nil {
		err := pkg.NewInternalError("Failed to generate access token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 生成新的刷新令牌（保持相同tenant_id）
	refreshToken, err := utils.GenerateRefreshToken(userID, tenantID)
	if err != nil {
		err := pkg.NewInternalError("Failed to generate refresh token", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 不返回密码信息
	user.Password = ""
	c.JSON(http.StatusOK, gin.H{"message": "令牌刷新成功", "access_token": accessToken, "refresh_token": refreshToken, "user": user})
}

// recordLoginHistory 记录登录历史
func recordLoginHistory(username, ipAddress, userAgent string, success bool, message string, tenantID uint) {
	loginHistory := models.LoginHistory{
		Username:  username,
		IPAddress: ipAddress,
		Success:   success,
		Message:   message,
		UserAgent: userAgent,
		TenantID:  tenantID,
		LoginTime: time.Now(),
	}

	// 异步记录登录历史，不阻塞主流程
	go func() {
		if err := pkg.DB.Create(&loginHistory).Error; err != nil {
			// 记录失败不应影响主流程，可以记录到日志中
			fmt.Printf("Failed to record login history: %v\n", err)
		}
	}()
}

// GetUsers 获取所有用户
func (uc *UserController) GetUsers(c *gin.Context) {
	var users []models.User
	tenantID := c.GetUint("tenant_id")
	// 根据需要预加载关联数据，避免N+1查询问题
	result := pkg.DB.Where("tenant_id = ?", tenantID).Find(&users)
	if result.Error != nil {
		err := pkg.NewDatabaseError("Failed to fetch users", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser 获取单个用户
func (uc *UserController) GetUser(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	var user models.User
	// 根据API需求预加载关联数据，这里根据常见使用场景选择预加载审计日志
	result := pkg.DB.Where("id = ? AND tenant_id = ?", id, tenantID).
		Preload("AuditLogs", func(db *gorm.DB) *gorm.DB {
			// 只预加载最近30天的审计日志
			return db.Where("created_at > ?", time.Now().AddDate(0, 0, -30)).Order("created_at DESC").Limit(100)
		}).First(&user)
	if result.Error != nil {
		err := pkg.NewNotFoundError("User not found", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser 创建用户
func (uc *UserController) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		err := pkg.NewValidationError("Invalid user data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 绑定租户ID，防止跨租户创建
	user.TenantID = c.GetUint("tenant_id")

	// 创建用户前先记录审计日志（不包含密码）
	logUser := user
	logUser.Password = "[REDACTED]"

	result := pkg.DB.Create(&user)
	if result.Error != nil {
		err := pkg.NewDatabaseError("Failed to create user", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 记录创建用户的审计日志
	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "create",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", user.ID),
		OldValue:     nil,
		NewValue:     logUser,
	})

	// 返回用户信息（不包含密码）
	user.Password = ""
	c.JSON(http.StatusCreated, user)
}

// UpdateUser 更新用户
func (uc *UserController) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	// 获取原始用户信息
	var oldUser models.User
	result := pkg.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&oldUser)
	if result.Error != nil {
		err := pkg.NewNotFoundError("User not found", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 记录原始值（不包含密码）
	auditOldUser := oldUser
	auditOldUser.Password = "[REDACTED]"

	// 绑定新的用户信息
	var newUser models.User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		err := pkg.NewValidationError("Invalid user data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 防止跨租户变更
	newUser.TenantID = tenantID
	newUser.ID = oldUser.ID // 确保ID不变

	// 如果没有更新密码，则保留原密码
	if newUser.Password == "" {
		newUser.Password = oldUser.Password
	}

	result = pkg.DB.Save(&newUser)
	if result.Error != nil {
		err := pkg.NewDatabaseError("Failed to update user", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 记录更新用户的审计日志
	auditNewUser := newUser
	auditNewUser.Password = "[REDACTED]"
	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "update",
		ResourceType: "user",
		ResourceID:   id,
		OldValue:     auditOldUser,
		NewValue:     auditNewUser,
	})

	// 返回更新后的用户信息（不包含密码）
	newUser.Password = ""
	c.JSON(http.StatusOK, newUser)
}

// DeleteUser 删除用户
func (uc *UserController) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetUint("tenant_id")

	// 先获取要删除的用户信息，用于审计日志
	var user models.User
	result := pkg.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&user)
	if result.Error != nil {
		err := pkg.NewNotFoundError("User not found", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 记录要删除的用户信息（不包含密码）
	auditUser := user
	auditUser.Password = "[REDACTED]"

	// 执行删除操作
	result = pkg.DB.Delete(&user)
	if result.Error != nil {
		err := pkg.NewDatabaseError("Failed to delete user", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 记录删除用户的审计日志
	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "delete",
		ResourceType: "user",
		ResourceID:   id,
		OldValue:     auditUser,
		NewValue:     nil,
	})

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ChangePasswordRequest 修改密码请求结构
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword 修改用户密码
func (uc *UserController) ChangePassword(c *gin.Context) {
	// 获取当前用户ID和租户ID
	currentUserID := c.GetUint("user_id")
	tenantID := c.GetUint("tenant_id")

	// 绑定请求参数
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		err := pkg.NewValidationError("Invalid request data", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 获取当前用户信息
	var user models.User
	result := pkg.DB.Where("id = ? AND tenant_id = ?", currentUserID, tenantID).First(&user)
	if result.Error != nil {
		err := pkg.NewNotFoundError("User not found", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 验证当前密码是否正确
	if !utils.CheckPasswordHash(req.CurrentPassword, user.Password) {
		err := pkg.NewValidationError("Current password is incorrect", nil)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 哈希新密码
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		err := pkg.NewInternalError("Failed to hash new password", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 更新密码
	user.Password = hashedPassword
	result = pkg.DB.Save(&user)
	if result.Error != nil {
		err := pkg.NewDatabaseError("Failed to update password", result.Error)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 记录修改密码的审计日志
	_ = pkg.AuditLogFromContext(c, pkg.AuditLogOptions{
		Action:       "change_password",
		ResourceType: "user",
		ResourceID:   fmt.Sprintf("%d", currentUserID),
		OldValue:     map[string]interface{}{"user_id": currentUserID},
		NewValue:     map[string]interface{}{"user_id": currentUserID, "password_changed": true},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
