package core

import (
	"fmt"
	"sync"
	"time"

	"weave/middleware"
	"weave/pkg/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Route 定义路由结构
type Route struct {
	Path         string            // 路由路径（不包含插件前缀）
	Method       string            // HTTP方法
	Handler      gin.HandlerFunc   // 处理函数
	Middlewares  []gin.HandlerFunc // 路由特定中间件
	Description  string            // 路由描述
	AuthRequired bool              // 是否需要认证
	Tags         []string          // 路由标签
	Params       map[string]string // 参数说明
} // 路由结构定义

// Plugin 插件接口定义
type Plugin interface {
	// 基础信息接口
	Name() string              // 返回插件名称
	Description() string       // 返回插件描述
	Version() string           // 返回插件版本
	GetDependencies() []string // 返回插件依赖的其他插件名称
	GetConflicts() []string    // 返回与当前插件冲突的插件名称

	// 生命周期接口
	Init() error      // 初始化插件
	Shutdown() error  // 关闭插件
	OnEnable() error  // 插件启用时调用（热重载相关）
	OnDisable() error // 插件禁用时调用（热重载相关）

	// 路由注册接口
	// 新版接口：提供路由定义，由PluginManager统一注册
	GetRoutes() []Route // 获取插件路由定义
	// 旧版接口：为了兼容现有插件保留
	RegisterRoutes(router *gin.Engine) // 注册插件路由

	// 执行功能接口
	Execute(params map[string]interface{}) (interface{}, error) // 执行插件功能

	// 插件配置接口（可选）
	GetDefaultMiddlewares() []gin.HandlerFunc // 获取插件默认中间件
	SetPluginManager(manager *PluginManager)  // 设置插件管理器引用
}

// PluginInfo 存储插件信息和路由元数据
type PluginInfo struct {
	Plugin       Plugin   // 插件实例
	Routes       []Route  // 插件路由
	Dependencies []string // 依赖的插件名称列表
	Conflicts    []string // 冲突的插件名称列表
	IsRegistered bool     // 路由是否已注册
	IsEnabled    bool     // 插件是否启用
}

// PluginWatcher 定义插件监控器接口
type PluginWatcher interface {
	Start() error
	Stop()
}

// PluginManager 插件管理器结构体类型
type PluginManager struct {
	plugins   map[string]PluginInfo // 存储插件信息和路由
	router    *gin.Engine           // 路由引擎引用
	mutex     *sync.RWMutex         // 读写锁，保证线程安全
	watcher   PluginWatcher         // 插件文件监控器
	logger    *zap.Logger           // 日志记录器
	pluginDir string                // 插件目录路径
}

// SetPluginWatcher 设置插件监控器实例
// 用于解决循环依赖问题，允许外部创建并注入PluginWatcher
func (pm *PluginManager) SetPluginWatcher(watcher PluginWatcher) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.watcher = watcher
}

// GlobalPluginManager 全局插件管理器实例
var GlobalPluginManager = &PluginManager{
	plugins:   make(map[string]PluginInfo),
	router:    nil,
	mutex:     &sync.RWMutex{},
	watcher:   nil,
	logger:    nil,       // 将在SetLogger中设置
	pluginDir: "plugins", // 默认插件目录
}

// SetRouter 设置路由引擎
func (pm *PluginManager) SetRouter(router *gin.Engine) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.router = router
}

// Register 注册插件
func (pm *PluginManager) Register(plugin Plugin) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("插件 '%s' 已存在", name)
	}

	// 设置插件管理器引用
	plugin.SetPluginManager(pm)

	// 检查冲突插件
	conflicts := plugin.GetConflicts()
	for _, conflictName := range conflicts {
		if _, exists := pm.plugins[conflictName]; exists {
			return fmt.Errorf("插件 '%s' 与已注册的插件 '%s' 冲突", name, conflictName)
		}
	}

	// 检查依赖插件
	dependencies := plugin.GetDependencies()
	for _, depName := range dependencies {
		if _, exists := pm.plugins[depName]; !exists {
			return fmt.Errorf("依赖的插件未注册: %s", depName)
		}
	}

	// 初始化插件
	if err := plugin.Init(); err != nil {
		return fmt.Errorf("插件 '%s' 初始化失败: %w", name, err)
	}

	// 创建插件信息
	info := PluginInfo{
		Plugin:       plugin,
		Routes:       plugin.GetRoutes(),
		Dependencies: dependencies,
		Conflicts:    conflicts,
		IsRegistered: false,
		IsEnabled:    true, // 默认为启用状态
	}

	pm.plugins[name] = info

	// 如果路由引擎已设置，自动注册路由
	if pm.router != nil {
		if err := pm.registerPluginRoutes(name); err != nil {
			return fmt.Errorf("插件 '%s' 路由注册失败: %w", name, err)
		}
	}

	return nil
}

// EnablePlugin 启用插件
func (pm *PluginManager) EnablePlugin(name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	info, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("插件 '%s' 不存在", name)
	}

	if info.IsEnabled {
		return nil // 已经是启用状态
	}

	// 检查依赖是否可用
	for _, depName := range info.Dependencies {
		depInfo, exists := pm.plugins[depName]
		if !exists || !depInfo.IsEnabled {
			return fmt.Errorf("依赖的插件 '%s' 未启用", depName)
		}
	}

	startTime := time.Now()
	success := true

	// 调用插件的OnEnable方法
	if err := info.Plugin.OnEnable(); err != nil {
		success = false
		metrics.RecordPluginError(name, "enable_failed")
		return fmt.Errorf("插件 '%s' 启用回调失败: %w", name, err)
	}

	// 启用插件
	info.IsEnabled = true
	pm.plugins[name] = info

	// 如果路由引擎已设置，注册路由
	if pm.router != nil && !info.IsRegistered {
		if err := pm.registerPluginRoutes(name); err != nil {
			success = false
			metrics.RecordPluginError(name, "route_registration_failed")
			return fmt.Errorf("插件 '%s' 路由注册失败: %w", name, err)
		}
	}

	// 记录插件执行时间和结果
	duration := time.Since(startTime)
	metrics.RecordPluginExecution(name, success, duration)
	metrics.RecordPluginMethodCall(name, "OnEnable", success)

	return nil
}

// DisablePlugin 禁用插件
func (pm *PluginManager) DisablePlugin(name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	info, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("插件 '%s' 不存在", name)
	}

	if !info.IsEnabled {
		return nil // 已经是禁用状态
	}

	// 检查是否有其他插件依赖当前插件
	for pluginName, pluginInfo := range pm.plugins {
		if pluginName != name && pluginInfo.IsEnabled {
			for _, depName := range pluginInfo.Dependencies {
				if depName == name {
					return fmt.Errorf("插件 '%s' 被插件 '%s' 依赖，无法禁用", name, pluginName)
				}
			}
		}
	}

	startTime := time.Now()
	success := true

	// 调用插件的OnDisable方法
	if err := info.Plugin.OnDisable(); err != nil {
		success = false
		metrics.RecordPluginError(name, "disable_failed")
		return fmt.Errorf("插件 '%s' 禁用回调失败: %w", name, err)
	}

	// 禁用插件
	info.IsEnabled = false
	pm.plugins[name] = info

	// 注意：Gin不支持动态删除路由，这里只能标记为禁用
	// 在ExecutePlugin等方法中会检查IsEnabled状态

	// 记录插件执行时间和结果
	duration := time.Since(startTime)
	metrics.RecordPluginExecution(name, success, duration)
	metrics.RecordPluginMethodCall(name, "OnDisable", success)

	return nil
}

// ReloadPlugin 重新加载插件
// 注意：这是一个简化实现，在实际生产环境中可能需要结合插件文件监控等功能
func (pm *PluginManager) ReloadPlugin(name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	startTime := time.Now()
	success := true

	// 获取当前插件信息
	info, exists := pm.plugins[name]
	if !exists {
		success = false
		metrics.RecordPluginReload(name, success)
		return fmt.Errorf("插件 '%s' 不存在", name)
	}

	plugin := info.Plugin
	isEnabled := info.IsEnabled

	// 先禁用插件
	if isEnabled {
		info.IsEnabled = false
		pm.plugins[name] = info
	}

	// 关闭当前插件
	if err := plugin.Shutdown(); err != nil {
		success = false
		metrics.RecordPluginReload(name, success)
		metrics.RecordPluginError(name, "shutdown_during_reload_failed")
		return fmt.Errorf("插件 '%s' 关闭失败: %w", name, err)
	}

	// 从管理器中移除插件
	delete(pm.plugins, name)

	// 重新初始化插件
	if err := plugin.Init(); err != nil {
		success = false
		metrics.RecordPluginReload(name, success)
		metrics.RecordPluginError(name, "init_during_reload_failed")
		return fmt.Errorf("插件 '%s' 重新初始化失败: %w", name, err)
	}

	// 重新创建插件信息
	newInfo := PluginInfo{
		Plugin:       plugin,
		Routes:       plugin.GetRoutes(),
		Dependencies: plugin.GetDependencies(),
		Conflicts:    plugin.GetConflicts(),
		IsRegistered: false,
		IsEnabled:    isEnabled,
	}

	pm.plugins[name] = newInfo

	// 如果路由引擎已设置且插件被启用，重新注册路由
	if pm.router != nil && isEnabled {
		if err := pm.registerPluginRoutes(name); err != nil {
			success = false
			metrics.RecordPluginReload(name, success)
			metrics.RecordPluginError(name, "route_registration_during_reload_failed")
			return fmt.Errorf("插件 '%s' 路由重新注册失败: %w", name, err)
		}
	}

	// 记录插件执行时间和结果
	duration := time.Since(startTime)
	metrics.RecordPluginExecution(name, success, duration)
	metrics.RecordPluginReload(name, success)

	return nil
}

// GetPluginStatus 获取插件状态
func (pm *PluginManager) GetPluginStatus(name string) (string, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	info, exists := pm.plugins[name]
	if !exists {
		return "not_registered", false
	}

	if info.IsEnabled {
		return "enabled", true
	}
	return "disabled", true
}

// GetAllPluginsInfo 获取所有插件信息
func (pm *PluginManager) GetAllPluginsInfo() []PluginInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	infos := make([]PluginInfo, 0, len(pm.plugins))
	for _, info := range pm.plugins {
		infos = append(infos, info)
	}
	return infos
}

// registerPluginRoutes 注册单个插件的路由
func (pm *PluginManager) registerPluginRoutes(name string) error {
	info, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("插件 '%s' 不存在", name)
	}

	if pm.router == nil {
		return fmt.Errorf("路由引擎未初始化")
	}

	// 如果路由已经注册，先清理
	if info.IsRegistered {
		// 注意：Gin不支持动态删除路由，这里只能标记为未注册
		// 实际生产环境中可能需要重启服务或使用其他路由方案
		info.IsRegistered = false
	}

	plugin := info.Plugin
	pluginName := plugin.Name()

	// 创建插件路由组
	pluginGroup := pm.router.Group(fmt.Sprintf("/plugins/%s", pluginName))

	// 添加插件默认中间件
	if defaultMiddlewares := plugin.GetDefaultMiddlewares(); len(defaultMiddlewares) > 0 {
		pluginGroup.Use(defaultMiddlewares...)
	}

	// 获取插件路由
	routes := plugin.GetRoutes()

	// 如果没有通过GetRoutes提供路由，则回退到旧版的RegisterRoutes方法
	if len(routes) == 0 {
		plugin.RegisterRoutes(pm.router)
		info.IsRegistered = true
		pm.plugins[name] = info
		return nil
	}

	// 注册每个路由
	for _, route := range routes {
		// 创建路由处理函数链
		handlers := append(route.Middlewares, route.Handler)

		// 如果需要认证，则在处理链前添加认证中间件
		if route.AuthRequired {
			handlers = append([]gin.HandlerFunc{middleware.AuthMiddleware()}, handlers...)
		}

		// 根据HTTP方法注册路由
		switch route.Method {
		case "GET":
			pluginGroup.GET(route.Path, handlers...)
		case "POST":
			pluginGroup.POST(route.Path, handlers...)
		case "PUT":
			pluginGroup.PUT(route.Path, handlers...)
		case "DELETE":
			pluginGroup.DELETE(route.Path, handlers...)
		case "PATCH":
			pluginGroup.PATCH(route.Path, handlers...)
		case "OPTIONS":
			pluginGroup.OPTIONS(route.Path, handlers...)
		default:
			return fmt.Errorf("不支持的HTTP方法: %s", route.Method)
		}

		// 更新路由信息
		info.Routes = routes
	}

	info.IsRegistered = true
	pm.plugins[name] = info
	return nil
}

// Unregister 注销插件
func (pm *PluginManager) Unregister(name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	info, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("插件 '%s' 不存在", name)
	}

	// 关闭插件
	plugin := info.Plugin
	if err := plugin.Shutdown(); err != nil {
		return fmt.Errorf("插件 '%s' 关闭失败: %w", name, err)
	}

	// 标记路由为未注册
	info.IsRegistered = false

	// 从管理器中删除插件
	delete(pm.plugins, name)
	return nil
}

// GetPlugin 获取插件
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	info, exists := pm.plugins[name]
	if !exists {
		return nil, false
	}
	return info.Plugin, true
}

// ListPlugins 列出所有插件
func (pm *PluginManager) ListPlugins() []string {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}
	return names
}

// GetPluginInfo 获取插件详细信息
func (pm *PluginManager) GetPluginInfo(name string) (*PluginInfo, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	info, exists := pm.plugins[name]
	if !exists {
		return nil, false
	}
	return &info, true
}

// RegisterAllRoutes 注册所有插件的路由
func (pm *PluginManager) RegisterAllRoutes() error {
	pm.mutex.RLock()
	// 复制插件名称列表，避免在注册过程中锁定太久
	pluginNames := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		pluginNames = append(pluginNames, name)
	}
	pm.mutex.RUnlock()

	// 逐个注册插件路由
	for _, name := range pluginNames {
		pm.mutex.Lock()
		err := pm.registerPluginRoutes(name)
		pm.mutex.Unlock()

		if err != nil {
			return fmt.Errorf("注册插件 '%s' 路由失败: %w", name, err)
		}
	}

	return nil
}

// GetAllRoutes 获取所有插件的路由信息
func (pm *PluginManager) GetAllRoutes() map[string][]Route {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	allRoutes := make(map[string][]Route)
	for name, info := range pm.plugins {
		allRoutes[name] = info.Routes
	}
	return allRoutes
}

// ExecutePlugin 执行插件功能
func (pm *PluginManager) ExecutePlugin(name string, params map[string]interface{}) (interface{}, error) {
	pm.mutex.RLock()
	info, exists := pm.plugins[name]
	pm.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("插件 '%s' 不存在", name)
	}

	// 检查插件是否启用
	if !info.IsEnabled {
		return nil, fmt.Errorf("插件 '%s' 已被禁用", name)
	}

	startTime := time.Now()
	success := true

	// 调用插件的Execute方法
	result, err := info.Plugin.Execute(params)
	if err != nil {
		success = false
		metrics.RecordPluginError(name, "execute_failed")
	}

	// 记录插件执行时间和结果
	duration := time.Since(startTime)
	metrics.RecordPluginExecution(name, success, duration)
	metrics.RecordPluginMethodCall(name, "Execute", success)

	return result, err
}

// RegisterPlugins 批量注册插件，自动处理依赖顺序
func (pm *PluginManager) RegisterPlugins(plugins []Plugin) error {
	// 1. 构建依赖图
	dependencyGraph := make(map[string][]string)
	pluginMap := make(map[string]Plugin)

	for _, plugin := range plugins {
		name := plugin.Name()
		pluginMap[name] = plugin
		dependencyGraph[name] = plugin.GetDependencies()
	}

	// 2. 拓扑排序
	sortedNames, err := topologicalSort(dependencyGraph)
	if err != nil {
		return err
	}

	// 3. 按排序结果注册插件
	for _, name := range sortedNames {
		if err := pm.Register(pluginMap[name]); err != nil {
			return err
		}
	}

	return nil
}

// topologicalSort 执行拓扑排序
func topologicalSort(graph map[string][]string) ([]string, error) {
	// 我们的输入是：plugin -> [dependencies]
	// 为了确保“先依赖后使用者”，需要将边方向反转为：dependency -> plugin

	inDegree := make(map[string]int)
	adj := make(map[string][]string)
	pluginNodes := make(map[string]struct{})

	// 初始化节点集合与入度
	for node := range graph {
		pluginNodes[node] = struct{}{}
		inDegree[node] = 0
	}

	// 构建反向邻接表，并计算入度（插件的入度是它的依赖数量）
	for plugin, deps := range graph {
		for _, dep := range deps {
			// 依赖节点也需要出现在入度表中，以便正确释放其邻居
			if _, ok := inDegree[dep]; !ok {
				inDegree[dep] = 0
			}
			adj[dep] = append(adj[dep], plugin)
			inDegree[plugin]++
		}
	}

	// 将入度为0的所有节点（包括未在graph中的纯依赖节点）加入队列
	queue := []string{}
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	sorted := []string{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// 仅将真实插件节点（graph的键）加入排序结果；纯依赖节点只用于释放其邻居
		if _, isPlugin := pluginNodes[current]; isPlugin {
			sorted = append(sorted, current)
		}

		for _, neighbor := range adj[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 如果无法将所有插件节点加入结果，说明存在循环依赖
	if len(sorted) != len(pluginNodes) {
		return nil, fmt.Errorf("插件依赖关系存在循环依赖")
	}

	return sorted, nil
}

// CheckDependencies 检查所有插件的依赖关系
func (pm *PluginManager) CheckDependencies() []error {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var errors []error

	for name, info := range pm.plugins {
		for _, depName := range info.Dependencies {
			if _, exists := pm.plugins[depName]; !exists {
				errors = append(errors, fmt.Errorf("插件 '%s' 依赖的插件 '%s' 未注册", name, depName))
			}
		}

		for _, conflictName := range info.Conflicts {
			if _, exists := pm.plugins[conflictName]; exists {
				errors = append(errors, fmt.Errorf("插件 '%s' 与插件 '%s' 冲突", name, conflictName))
			}
		}
	}

	return errors
}

// GetDependencyGraph 获取插件依赖图
func (pm *PluginManager) GetDependencyGraph() map[string]map[string]bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	graph := make(map[string]map[string]bool)

	for name, info := range pm.plugins {
		graph[name] = make(map[string]bool)
		for _, depName := range info.Dependencies {
			graph[name][depName] = true
		}
	}

	return graph
}

// SetLogger 设置日志记录器
func (pm *PluginManager) SetLogger(logger *zap.Logger) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.logger = logger
}

// SetPluginDir 设置插件目录
func (pm *PluginManager) SetPluginDir(dir string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.pluginDir = dir
}

// StartPluginWatcher 启动插件监控器
// 注意：此方法需要外部提供PluginWatcher实例
func (pm *PluginManager) StartPluginWatcher() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 如果监控器已存在且正在运行，直接返回
	if pm.watcher != nil {
		return nil
	}

	// 注意：此方法已被重构，请在应用程序初始化时通过SetPluginWatcher方法设置监控器实例
	return fmt.Errorf("插件监控器未初始化，请先设置PluginWatcher实例")
}

// StopPluginWatcher 停止插件监控器
func (pm *PluginManager) StopPluginWatcher() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.watcher != nil {
		pm.watcher.Stop()
		pm.watcher = nil
	}
}
