package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"weave/controllers"
	"weave/models"
	"weave/pkg"
)

// 测试 AuditController
type testAuditController struct {
	controllers.AuditController
}

func (tac *testAuditController) GetAuditLogs(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	action := c.Query("action")
	resourceType := c.Query("resource_type")
	username := c.Query("username")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建查询
	query := pkg.DB.Model(&models.AuditLog{})

	// 添加租户过滤（多租户隔离）
	tenantID := c.GetUint("tenant_id")
	query = query.Where("tenant_id = ?", tenantID)

	// 添加过滤条件
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if username != "" {
		query = query.Where("username = ?", username)
	}
	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query = query.Where("created_at <= ?", endTime)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		err := pkg.NewDatabaseError("Failed to count audit logs", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 获取分页数据，不使用Preload避免关联错误
	var auditLogs []models.AuditLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&auditLogs).Error; err != nil {
		err := pkg.NewDatabaseError("Failed to fetch audit logs", err)
		c.JSON(pkg.GetHTTPStatus(err), gin.H{"code": string(err.Code), "message": err.Message})
		return
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	// 返回分页结果
	c.JSON(http.StatusOK, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"logs":        auditLogs,
	})
}

func TestAuditControllerGetAuditLogsTenantIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup in-memory DB and assign to pkg.DB
	db := setupTestDB(t)

	// Seed two tenants
	log1 := models.AuditLog{
		UserID:       1,
		Username:     "alice",
		Action:       "create",
		ResourceType: "note",
		ResourceID:   "n1",
		TenantID:     1,
		CreatedAt:    time.Now(),
	}
	log2 := models.AuditLog{
		UserID:       2,
		Username:     "bob",
		Action:       "delete",
		ResourceType: "note",
		ResourceID:   "n2",
		TenantID:     2,
		CreatedAt:    time.Now(),
	}
	if err := db.Create(&log1).Error; err != nil {
		t.Fatalf("seed log1 error: %v", err)
	}
	if err := db.Create(&log2).Error; err != nil {
		t.Fatalf("seed log2 error: %v", err)
	}

	ac := testAuditController{}
	r := gin.New()
	// Set tenant_id=1 via middleware
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", uint(1))
		c.Next()
	})
	r.GET("/audit", func(c *gin.Context) { ac.GetAuditLogs(c) })

	req, _ := http.NewRequest("GET", "/audit?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Total      int64             `json:"total"`
		Page       int               `json:"page"`
		PageSize   int               `json:"page_size"`
		TotalPages int               `json:"total_pages"`
		Logs       []models.AuditLog `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}

	if body.Total != 1 {
		t.Fatalf("expected total=1 for tenant 1, got %d", body.Total)
	}
	if len(body.Logs) != 1 || body.Logs[0].TenantID != 1 {
		t.Fatalf("expected 1 log for tenant 1, got %d", len(body.Logs))
	}
}
