package models

import (
	"time"
)

// Note 笔记模型
type Note struct {
	ID          uint      `gorm:"primaryKey" json:"id"`          // 使用MySQL自增ID
	UserID      uint      `gorm:"not null;index" json:"user_id"` // 添加用户ID字段，建立索引提高查询效率
	TenantID    uint      `gorm:"index" json:"tenant_id"`
	Title       string    `gorm:"size:255;not null;index" json:"title"` // 添加索引
	Content     string    `gorm:"type:text;not null" json:"content"`
	CreatedTime time.Time `gorm:"index" json:"created_time"` // 添加索引
	UpdatedTime time.Time `json:"updated_time"`
}
