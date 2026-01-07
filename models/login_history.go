package models

import (
	"time"
)

// LoginHistory 登录历史模型
// 记录用户登录的详细信息，包括登录时间、IP地址、登录状态等
// 这种方式比日志形式更便于查询和分析用户登录行为
// 对于安全审计和异常检测非常有用

type LoginHistory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:50;not null" json:"username"` // 登录用户名
	IPAddress string    `gorm:"size:50" json:"ip_address"`        // 登录IP地址
	Success   bool      `gorm:"not null" json:"success"`          // 登录是否成功
	Message   string    `gorm:"size:255" json:"message"`          // 登录结果消息/失败原因
	UserAgent string    `gorm:"type:text" json:"user_agent"`      // 用户代理信息
	TenantID  uint      `gorm:"index" json:"tenant_id"`
	LoginTime time.Time `json:"login_time"` // 登录时间
}
