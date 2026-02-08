package models

import (
	"time"
)

// EmailVerificationCode 邮箱验证码模型
type EmailVerificationCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"size:100;not null;index" json:"email"`
	Code      string    `gorm:"size:60;not null" json:"code"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	TenantID  uint      `gorm:"index" json:"tenant_id"`
}

// TableName 指定表名
func (EmailVerificationCode) TableName() string {
	return "email_verification_codes"
}
