package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weave/controllers"
	"weave/plugins"
	"weave/plugins/core"
)

func setupMemoryDBForHealth(t *testing.T) *gorm.DB {
	return setupTestDB(t)
}

type hcTestPlugin struct{ pm *core.PluginManager }

func (p *hcTestPlugin) Name() string                                               { return "hc_demo" }
func (p *hcTestPlugin) Description() string                                        { return "health test plugin" }
func (p *hcTestPlugin) Version() string                                            { return "1.0.0" }
func (p *hcTestPlugin) GetDependencies() []string                                  { return nil }
func (p *hcTestPlugin) GetConflicts() []string                                     { return nil }
func (p *hcTestPlugin) Init() error                                                { return nil }
func (p *hcTestPlugin) Shutdown() error                                            { return nil }
func (p *hcTestPlugin) OnEnable() error                                            { return nil }
func (p *hcTestPlugin) OnDisable() error                                           { return nil }
func (p *hcTestPlugin) GetRoutes() []core.Route                                    { return nil }
func (p *hcTestPlugin) GetDefaultMiddlewares() []gin.HandlerFunc                   { return nil }
func (p *hcTestPlugin) SetPluginManager(manager *core.PluginManager)               { p.pm = manager }
func (p *hcTestPlugin) RegisterRoutes(router *gin.Engine)                          {}
func (p *hcTestPlugin) Execute(params map[string]interface{}) (interface{}, error) { return nil, nil }

func TestGetHealth_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = setupMemoryDBForHealth(t)

	hc := controllers.HealthController{}
	r := gin.New()
	r.GET("/health", hc.GetHealth)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %#v", body["status"])
	}
	db := body["database"].(map[string]interface{})
	if healthy, ok := db["healthy"].(bool); !ok || !healthy {
		t.Fatalf("expected database healthy=true, got %#v", db)
	}
	pluginsInfo := body["plugins"].(map[string]interface{})
	if cnt, ok := pluginsInfo["pluginCount"].(float64); !ok || int(cnt) != 0 {
		t.Fatalf("expected pluginCount=0, got %#v", pluginsInfo)
	}
}

func TestPluginHealthCheck_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = setupMemoryDBForHealth(t)

	hc := controllers.HealthController{}
	r := gin.New()
	r.GET("/health/plugins/:name", hc.PluginHealthCheck)

	req, _ := http.NewRequest(http.MethodGet, "/health/plugins/unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if body["status"] != "error" || body["message"] != "插件不存在" {
		t.Fatalf("unexpected response: %#v", body)
	}
}

func TestPluginHealthCheck_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = setupMemoryDBForHealth(t)

	// Register a test plugin
	_ = plugins.PluginManager.Unregister("hc_demo")
	if err := plugins.PluginManager.Register(&hcTestPlugin{}); err != nil {
		t.Fatalf("register plugin error: %v", err)
	}
	defer func() { _ = plugins.PluginManager.Unregister("hc_demo") }()

	hc := controllers.HealthController{}
	r := gin.New()
	r.GET("/health/plugins/:name", hc.PluginHealthCheck)

	req, _ := http.NewRequest(http.MethodGet, "/health/plugins/hc_demo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if body["name"] != "hc_demo" || body["status"] != "enabled" || body["enabled"] != true || body["healthy"] != true {
		t.Fatalf("unexpected plugin health response: %#v", body)
	}
}
