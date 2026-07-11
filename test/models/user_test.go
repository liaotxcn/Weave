package models

import (
	"errors"
	"testing"
	"time"

	"weave/models"
	"weave/pkg"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// 初始化测试数据库（使用内存SQLite）
func setupTestDB(t *testing.T) {
	if pkg.DB != nil {
		_ = pkg.CloseDatabase()
		pkg.DB = nil
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("failed to open sqlite test db: %v", err)
	}
	pkg.DB = db
	sqlDB, err := pkg.DB.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	if err := models.MigrateTables(pkg.DB); err != nil {
		t.Fatalf("Failed to migrate tables: %v", err)
	}
}

// 清理测试数据
func cleanupTestDB(t *testing.T) {
	if pkg.DB != nil {
		_ = pkg.CloseDatabase()
		pkg.DB = nil
	}
}

// TestUserCreate 测试创建用户
func TestUserCreate(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建测试用户
	user := models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存用户到数据库
	err := pkg.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 验证用户是否创建成功（ID不为0）
	if user.ID == 0 {
		t.Fatal("User ID is 0, creation failed")
	}

	// 清理
	defer pkg.DB.Delete(&user)
}

// TestMigrateTables 验证迁移后表是否存在
func TestMigrateTables(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// 使用SQLite元数据检查表是否存在
	tables := []string{"user", "tool", "tool_history", "note", "login_history", "audit_logs"}
	for _, tbl := range tables {
		var count int64
		// sqlite_master 查询检查表存在
		res := pkg.DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&count)
		if res.Error != nil {
			t.Fatalf("error checking table %s existence: %v", tbl, res.Error)
		}
		if count == 0 {
			t.Fatalf("table %s does not exist after migration", tbl)
		}
	}
}

// TestUserUpdate 测试更新用户信息
func TestUserUpdate(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 先创建一个测试用户
	user := models.User{
		Username: "updateuser",
		Email:    "update@example.com",
		Password: "password123",
	}

	// 保存用户到数据库
	err := pkg.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create user for update test: %v", err)
	}

	// 清理
	defer pkg.DB.Delete(&user)

	// 更新用户信息（更新现有字段，避免使用不存在的列）
	newEmail := "updated@example.com"

	err = pkg.DB.Model(&user).Updates(map[string]interface{}{
		"email": newEmail,
	}).Error
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// 重新查询用户以验证更新
	var updatedUser models.User
	err = pkg.DB.First(&updatedUser, user.ID).Error
	if err != nil {
		t.Fatalf("Failed to find user after update: %v", err)
	}

	if updatedUser.Email != newEmail {
		t.Errorf("Expected email %s, got %s", newEmail, updatedUser.Email)
	}
}

// TestUserDelete 测试删除用户
func TestUserDelete(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 先创建一个测试用户
	user := models.User{
		Username: "deleteuser",
		Email:    "delete@example.com",
		Password: "password123",
	}

	// 保存用户到数据库
	err := pkg.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create user for delete test: %v", err)
	}

	// 记录用户ID用于后续验证
	userId := user.ID

	// 删除用户
	err = pkg.DB.Delete(&user).Error
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// 验证用户是否已被删除
	var deletedUser models.User
	err = pkg.DB.First(&deletedUser, userId).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("Expected record not found error after delete, got %v", err)
	}
}
