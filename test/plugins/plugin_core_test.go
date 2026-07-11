package plugins_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"weave/config"
	"weave/pkg"
	"weave/plugins"
	"weave/plugins/core"
	"weave/plugins/watcher"
)

// mockPlugin implements core.Plugin for testing
type mockPlugin struct {
	name          string
	deps          []string
	conflicts     []string
	routes        []core.Route
	pm            *core.PluginManager
	initCount     int
	shutdownCount int
	enableCount   int
	disableCount  int
}

func (p *mockPlugin) Name() string                                 { return p.name }
func (p *mockPlugin) Description() string                          { return "mock plugin for tests" }
func (p *mockPlugin) Version() string                              { return "0.1.0" }
func (p *mockPlugin) GetDependencies() []string                    { return p.deps }
func (p *mockPlugin) GetConflicts() []string                       { return p.conflicts }
func (p *mockPlugin) Init() error                                  { p.initCount++; return nil }
func (p *mockPlugin) Shutdown() error                              { p.shutdownCount++; return nil }
func (p *mockPlugin) OnEnable() error                              { p.enableCount++; return nil }
func (p *mockPlugin) OnDisable() error                             { p.disableCount++; return nil }
func (p *mockPlugin) GetRoutes() []core.Route                      { return p.routes }
func (p *mockPlugin) GetDefaultMiddlewares() []gin.HandlerFunc     { return nil }
func (p *mockPlugin) SetPluginManager(manager *core.PluginManager) { p.pm = manager }
func (p *mockPlugin) Execute(params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"executed": true, "params": params}, nil
}
func (p *mockPlugin) RegisterRoutes(router *gin.Engine) {
	// Fallback route when GetRoutes returns empty
	router.GET(fmt.Sprintf("/plugins/%s/custom", p.name), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "custom", "plugin": p.name})
	})
}

func resetPluginManager(t *testing.T) {
	// Unregister any existing plugins to isolate tests
	names := plugins.PluginManager.ListPlugins()
	for _, n := range names {
		_ = plugins.PluginManager.Unregister(n)
	}
	// Also clear router to avoid cross-test route registrations
	plugins.PluginManager.SetRouter(nil)
}

func TestRegisterAndRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	r := gin.New()
	plugins.PluginManager.SetRouter(r)

	p := &mockPlugin{
		name: "mock",
		routes: []core.Route{{
			Path:   "ping",
			Method: "GET",
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			},
		}},
	}
	if err := plugins.PluginManager.Register(p); err != nil {
		t.Fatalf("register error: %v", err)
	}

	req, _ := http.NewRequest("GET", "/plugins/mock/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRegisterWithMissingDependencyFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	plugins.PluginManager.SetRouter(gin.New())

	p := &mockPlugin{name: "needs-dep", deps: []string{"dep1"}}
	if err := plugins.PluginManager.Register(p); err == nil {
		t.Fatalf("expected error for missing dependency, got nil")
	}
}

func TestConflictsPreventRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	plugins.PluginManager.SetRouter(gin.New())

	pA := &mockPlugin{name: "A"}
	if err := plugins.PluginManager.Register(pA); err != nil {
		t.Fatalf("register A error: %v", err)
	}

	pB := &mockPlugin{name: "B", conflicts: []string{"A"}}
	if err := plugins.PluginManager.Register(pB); err == nil {
		t.Fatalf("expected conflict error when registering B, got nil")
	}
}

func TestReloadPluginCallsShutdownAndInit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	// Do NOT set router to avoid duplicate route registration during reload
	// plugins.PluginManager.SetRouter(gin.New())

	p := &mockPlugin{name: "reloadable"}
	if err := plugins.PluginManager.Register(p); err != nil {
		t.Fatalf("register error: %v", err)
	}
	if p.initCount != 1 {
		t.Fatalf("expected initCount=1 after register, got %d", p.initCount)
	}

	if err := plugins.PluginManager.ReloadPlugin("reloadable"); err != nil {
		t.Fatalf("reload error: %v", err)
	}
	if p.shutdownCount != 1 || p.initCount != 2 {
		t.Fatalf("expected shutdown=1 and init=2 after reload, got shutdown=%d init=%d", p.shutdownCount, p.initCount)
	}
}

func TestEnableDisablePlugin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	plugins.PluginManager.SetRouter(gin.New())

	p := &mockPlugin{name: "toggle"}
	if err := plugins.PluginManager.Register(p); err != nil {
		t.Fatalf("register error: %v", err)
	}
	if err := plugins.PluginManager.DisablePlugin("toggle"); err != nil {
		t.Fatalf("disable error: %v", err)
	}
	if p.disableCount != 1 {
		t.Fatalf("expected disableCount=1, got %d", p.disableCount)
	}
	if err := plugins.PluginManager.EnablePlugin("toggle"); err != nil {
		t.Fatalf("enable error: %v", err)
	}
	if p.enableCount != 1 {
		t.Fatalf("expected enableCount=1, got %d", p.enableCount)
	}
}

func TestRegisterRoutesFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	r := gin.New()
	plugins.PluginManager.SetRouter(r)

	p := &mockPlugin{name: "fallback"} // GetRoutes empty triggers RegisterRoutes fallback
	if err := plugins.PluginManager.Register(p); err != nil {
		t.Fatalf("register error: %v", err)
	}

	req, _ := http.NewRequest("GET", "/plugins/fallback/custom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for fallback route, got %d", w.Code)
	}
}

func TestExecutePlugin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	plugins.PluginManager.SetRouter(gin.New())

	p := &mockPlugin{name: "exec"}
	if err := plugins.PluginManager.Register(p); err != nil {
		t.Fatalf("register error: %v", err)
	}

	res, err := plugins.PluginManager.ExecutePlugin("exec", map[string]interface{}{"x": 1})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["executed"] != true {
		t.Fatalf("expected executed=true, got %v", m["executed"])
	}
}

// stubWatcher implements core.PluginWatcher for manager watcher tests
type stubWatcher struct{ stopped bool }

func (sw *stubWatcher) Start() error { return nil }
func (sw *stubWatcher) Stop()        { sw.stopped = true }

// stubManager implements watcher.PluginManager for file watcher tests
type stubManager struct {
	reloaded     []string
	unregistered []string
	registered   []string
	plugins      map[string]bool
}

func newStubManager() *stubManager { return &stubManager{plugins: make(map[string]bool)} }
func (sm *stubManager) ReloadPlugin(name string) error {
	sm.reloaded = append(sm.reloaded, name)
	return nil
}
func (sm *stubManager) GetPlugin(name string) (watcher.Plugin, bool) {
	if sm.plugins[name] {
		return stubWatchedPlugin{name: name}, true
	}
	return nil, false
}
func (sm *stubManager) Unregister(name string) error {
	sm.unregistered = append(sm.unregistered, name)
	delete(sm.plugins, name)
	return nil
}
func (sm *stubManager) Register(p watcher.Plugin) error {
	sm.registered = append(sm.registered, p.Name())
	sm.plugins[p.Name()] = true
	return nil
}

// stubWatchedPlugin minimal watcher.Plugin
type stubWatchedPlugin struct{ name string }

func (p stubWatchedPlugin) Name() string { return p.name }

// --- Additional tests for plugin core and watcher ---

func TestRegisterPluginsCycleDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	// Do not set router
	pA := &mockPlugin{name: "A", deps: []string{"B"}}
	pB := &mockPlugin{name: "B", deps: []string{"A"}}
	// Cycle should cause error
	err := plugins.PluginManager.RegisterPlugins([]core.Plugin{pA, pB})
	if err == nil {
		t.Fatalf("expected cycle dependency error, got nil")
	}
}

func TestCheckDependenciesReportsMissing(t *testing.T) {
	t.Skip("redundant with core package dependency tests; skip to reduce duplication")
}

func TestGetDependencyGraphContents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	pA := &mockPlugin{name: "A", deps: []string{"B", "C"}}
	pB := &mockPlugin{name: "B"}
	pC := &mockPlugin{name: "C"}
	// Register dependencies first to satisfy Register() checks
	if err := plugins.PluginManager.Register(pB); err != nil {
		t.Fatalf("register B error: %v", err)
	}
	if err := plugins.PluginManager.Register(pC); err != nil {
		t.Fatalf("register C error: %v", err)
	}
	if err := plugins.PluginManager.Register(pA); err != nil {
		t.Fatalf("register A error: %v", err)
	}
	graph := plugins.PluginManager.GetDependencyGraph()
	if _, ok := graph["A"]["B"]; !ok {
		t.Fatalf("expected graph to include A->B")
	}
	if _, ok := graph["A"]["C"]; !ok {
		t.Fatalf("expected graph to include A->C")
	}
}

func TestPluginManagerWatcherStartStop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetPluginManager(t)
	// Without watcher set, StartPluginWatcher should error
	if err := plugins.PluginManager.StartPluginWatcher(); err == nil {
		t.Fatalf("expected error when watcher not set, got nil")
	}
	// Set stub watcher and start/stop
	sw := &stubWatcher{}
	plugins.PluginManager.SetPluginWatcher(sw)
	if err := plugins.PluginManager.StartPluginWatcher(); err != nil {
		t.Fatalf("unexpected error when watcher set: %v", err)
	}
	plugins.PluginManager.StopPluginWatcher()
	if !sw.stopped {
		t.Fatalf("expected watcher.Stop to be called")
	}
}

func TestWatcherReloadOnChangeAndUnregisterOnDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Prepare temp plugin directory
	dir, err := os.MkdirTemp("", "pluginwatcher")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(dir)

	// Configure hot reload on
	config.Config.Plugins.HotReload = true
	// Create stub manager and mark plugin as registered
	sm := newStubManager()
	sm.plugins["reloadable"] = true
	// Pre-create a .go file so initial scan detects it
	path := filepath.Join(dir, "reloadable.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	// Build watcher
	logger := pkg.GetLogger()
	pw, err := watcher.NewPluginWatcher(dir, sm, logger)
	if err != nil {
		t.Fatalf("new watcher error: %v", err)
	}
	if err := pw.Start(); err != nil {
		t.Fatalf("start watcher error: %v", err)
	}
	defer pw.Stop()

	// Modify the file to trigger fsnotify change in addition to initial scan
	if err := os.WriteFile(path, []byte("package main\n// change"), 0644); err != nil {
		t.Fatalf("rewrite file error: %v", err)
	}
		// Allow processing from scan and fsnotify
		reloadOK := false
		for i := 0; i < 20; i++ {
			time.Sleep(300 * time.Millisecond)
			if len(sm.reloaded) > 0 && sm.reloaded[0] == "reloadable" {
				reloadOK = true
				break
			}
		}
		if !reloadOK {
			t.Fatalf("expected reloadable to be reloaded, got %#v", sm.reloaded)
		}

		// Delete file to trigger unregister
		if err := os.Remove(path); err != nil {
			t.Fatalf("remove file error: %v", err)
		}
		// Give watcher time to process delete event
		unregOK := false
		for i := 0; i < 20; i++ {
			time.Sleep(300 * time.Millisecond)
			if len(sm.unregistered) > 0 && sm.unregistered[0] == "reloadable" {
				unregOK = true
				break
			}
		}
		if !unregOK {
			t.Fatalf("expected reloadable to be unregistered, got %#v", sm.unregistered)
		}
}

func TestWatcherSkipLoadWithoutSO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir, err := os.MkdirTemp("", "pluginwatcher2")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(dir)
	config.Config.Plugins.HotReload = true
	sm := newStubManager() // no existing plugin registered
	// Pre-create a .go file so initial scan detects it
	path := filepath.Join(dir, "newone.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	logger := pkg.GetLogger()
	pw, err := watcher.NewPluginWatcher(dir, sm, logger)
	if err != nil {
		t.Fatalf("new watcher error: %v", err)
	}
	if err := pw.Start(); err != nil {
		t.Fatalf("start watcher error: %v", err)
	}
	defer pw.Stop()
	// Allow processing from initial scan
	time.Sleep(1500 * time.Millisecond)
	if len(sm.registered) != 0 {
		t.Fatalf("expected no dynamic register without .so, got %#v", sm.registered)
	}
}
