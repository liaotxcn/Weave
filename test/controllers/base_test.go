package controllers_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"weave/models"
	"weave/pkg"
)

// setupTestDB 创建并配置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("gorm open error: %v", err)
	}

	// 使用models包中定义的标准表迁移顺序
	if err := models.MigrateTables(db); err != nil {
		t.Fatalf("migrate tables error: %v", err)
	}

	// 设置全局DB实例
	pkg.DB = db

	return db
}
