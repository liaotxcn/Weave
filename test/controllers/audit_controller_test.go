package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"weave/models"
)

func TestAuditControllerGetAuditLogsTenantIsolation(t *testing.T) {
	t.Skip()
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

	ac := newTestAuditController(db)
	r := gin.New()
	// Set tenant_id=1 via middleware
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", uint(1))
		c.Next()
	})
	r.GET("/audit", ac.GetAuditLogs)

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
