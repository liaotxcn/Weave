package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:50;not null;unique" json:"username"`
	Password  string    `gorm:"size:100;not null" json:"password,omitempty"`
	Email     string    `gorm:"size:100;unique" json:"email"`
	TenantID  uint      `gorm:"index" json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Tool 工具模型
type Tool struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null;unique" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Icon        string    `gorm:"size:255" json:"icon"`
	PluginName  string    `gorm:"size:100;not null" json:"plugin_name"`
	IsEnabled   bool      `gorm:"default:true" json:"is_enabled"`
	TenantID    uint      `gorm:"index" json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToolHistory 工具使用历史模型
type ToolHistory struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	UserID   uint      `json:"user_id"`
	ToolID   uint      `json:"tool_id"`
	TenantID uint      `gorm:"index" json:"tenant_id"`
	UsedAt   time.Time `json:"used_at"`
	Params   string    `gorm:"type:text" json:"params"`
	Result   string    `gorm:"type:text" json:"result"`
}

// 迁移数据表(依赖顺序)
func MigrateTables(db *gorm.DB) error {
	if err := db.AutoMigrate(&User{}, &Tool{}, &EmailVerificationCode{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&Team{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&Note{}, &LoginHistory{}, &AuditLog{}, &ToolHistory{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&TeamMember{}); err != nil {
		return err
	}
	return nil
}
