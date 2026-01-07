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

func setupMemoryDBForTeam(t *testing.T) *gorm.DB {
	return setupTestDB(t)
}

func TestCreateTeam_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = setupMemoryDBForTeam(t)

	tc := controllers.TeamController{}
	r := gin.New()
	// Inject authenticated context
	r.Use(func(c *gin.Context) { c.Set("tenant_id", uint(10)); c.Set("user_id", uint(99)); c.Next() })
	r.POST("/teams", tc.CreateTeam)

	payload := `{"name":"alpha","description":"alpha team"}`
	req, _ := http.NewRequest(http.MethodPost, "/teams", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var team models.Team
	if err := json.Unmarshal(w.Body.Bytes(), &team); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if team.ID == 0 || team.Name != "alpha" || team.TenantID != 10 || team.OwnerID != 99 {
		t.Fatalf("unexpected created team: %#v", team)
	}
}

func TestCreateTeam_DuplicateName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupMemoryDBForTeam(t)
	// Seed one team in tenant 5
	seed := models.Team{Name: "alpha", TenantID: 5, OwnerID: 7}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed team error: %v", err)
	}

	tc := controllers.TeamController{}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", uint(5)); c.Set("user_id", uint(9)); c.Next() })
	r.POST("/teams", tc.CreateTeam)

	payload := `{"name":"alpha"}`
	req, _ := http.NewRequest(http.MethodPost, "/teams", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if body["code"] != "CONFLICT" {
		t.Fatalf("expected code 'CONFLICT', got %#v", body["code"])
	}
}
