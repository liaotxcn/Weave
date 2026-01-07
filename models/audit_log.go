package models

import (
	"time"
)

// AuditLog 安全审计日志模型
// 记录系统中所有关键操作的详细信息，用于安全审计和合规性检查
type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `json:"user_id"`                       // 操作用户ID，如果未登录则为0
	Username     string    `gorm:"size:50" json:"username"`       // 操作用户名
	Action       string    `gorm:"size:100" json:"action"`        // 操作类型，如create、update、delete、login、logout等
	ResourceType string    `gorm:"size:100" json:"resource_type"` // 资源类型，如user、tool、plugin等
	ResourceID   string    `gorm:"size:100" json:"resource_id"`   // 资源ID
	OldValue     string    `gorm:"type:text" json:"old_value"`    // 操作前的值（JSON格式）
	NewValue     string    `gorm:"type:text" json:"new_value"`    // 操作后的值（JSON格式）
	IPAddress    string    `gorm:"size:50" json:"ip_address"`     // 操作IP地址
	UserAgent    string    `gorm:"type:text" json:"user_agent"`   // 用户代理信息
	TenantID     uint      `gorm:"index" json:"tenant_id"`        // 租户ID，用于多租户环境
	CreatedAt    time.Time `json:"created_at"`                    // 操作时间
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}
