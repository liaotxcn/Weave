package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weave/controllers"
	"weave/models"
)

func setupMemoryDBForTool(t *testing.T) *gorm.DB {
	return setupTestDB(t)
}

func TestGetTools_TenantIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupMemoryDBForTool(t)

	t1 := models.Tool{Name: "toolA", Description: "A", PluginName: "p1", IsEnabled: true, TenantID: 1}
	t2 := models.Tool{Name: "toolB", Description: "B", PluginName: "p2", IsEnabled: true, TenantID: 2}
	if err := db.Create(&t1).Error; err != nil {
		t.Fatalf("seed t1 error: %v", err)
	}
	if err := db.Create(&t2).Error; err != nil {
		t.Fatalf("seed t2 error: %v", err)
	}

	tc := controllers.ToolController{}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", uint(1)); c.Next() })
	r.GET("/tools", tc.GetTools)

	req, _ := http.NewRequest(http.MethodGet, "/tools", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var tools []models.Tool
	if err := json.Unmarshal(w.Body.Bytes(), &tools); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if len(tools) != 1 || tools[0].TenantID != 1 || tools[0].Name != "toolA" {
		t.Fatalf("expected 1 tool for tenant 1, got %#v", tools)
	}
}

func TestGetTool_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = setupMemoryDBForTool(t)

	tc := controllers.ToolController{}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", uint(1)); c.Next() })
	r.GET("/tools/:id", tc.GetTool)

	req, _ := http.NewRequest(http.MethodGet, "/tools/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if body["message"] != "Tool not found" {
		t.Fatalf("expected message 'Tool not found', got %#v", body["message"])
	}
}

func TestCreateTool_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = setupMemoryDBForTool(t)

	tc := controllers.ToolController{}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", uint(3)); c.Next() })
	r.POST("/tools", tc.CreateTool)

	payload := `{"name":"toolC","description":"C","plugin_name":"pc","is_enabled":true}`
	req, _ := http.NewRequest(http.MethodPost, "/tools", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var created models.Tool
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if created.ID == 0 || created.Name != "toolC" || created.TenantID != 3 {
		t.Fatalf("unexpected created tool: %#v", created)
	}
}
