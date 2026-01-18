package watcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"weave/config"
	"weave/pkg"
	"weave/pkg/metrics"
	"weave/plugins/loader"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// Plugin 定义watcher包中使用的最小插件接口
type Plugin interface {
	Name() string
}

// PluginManager 定义插件管理器接口，避免循环依赖
type PluginManager interface {
	ReloadPlugin(name string) error
	GetPlugin(name string) (Plugin, bool)
	Unregister(name string) error
	Register(plugin Plugin) error
}

// PluginWatcher 插件文件监控器
type PluginWatcher struct {
	watcher      *fsnotify.Watcher
	pluginDir    string
	manager      PluginManager
	loader       *loader.PluginLoader
	logger       *pkg.Logger
	mu           sync.RWMutex
	watchedFiles map[string]time.Time
	scanInterval time.Duration
	running      bool
	stopChan     chan struct{}
	processChan  chan string
}

// NewPluginWatcher 创建插件监控器
func NewPluginWatcher(pluginDir string, manager PluginManager, logger *pkg.Logger) (*PluginWatcher, error) {
	// 创建fsnotify监控器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("创建文件监控器失败: %w", err)
	}

	// 从配置中获取扫描间隔，默认为5秒
	scanInterval := 5
	if config.Config.Plugins.ScanInterval > 0 {
		scanInterval = config.Config.Plugins.ScanInterval
	}

	// 创建插件加载器
	pluginLoader := loader.NewPluginLoader(logger)

	pw := &PluginWatcher{
		watcher:      watcher,
		pluginDir:    pluginDir,
		manager:      manager,
		loader:       pluginLoader,
		logger:       logger,
		watchedFiles: make(map[string]time.Time),
		scanInterval: time.Duration(scanInterval) * time.Second,
		running:      false,
		stopChan:     make(chan struct{}),
		processChan:  make(chan string, 100),
	}

	// 确保插件目录存在
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("创建插件目录失败: %w", err)
	}

	return pw, nil
}

// Start 启动监控器
func (pw *PluginWatcher) Start() error {
	pw.mu.Lock()
	if pw.running {
		pw.mu.Unlock()
		return nil
	}
	pw.running = true
	pw.mu.Unlock()

	// 启动监控循环
	go pw.watchLoop()
	// 启动定期扫描（额外保障）
	go pw.scanLoop()
	// 启动文件处理队列
	go pw.processQueue()

	// 初始扫描插件目录
	pw.scanPluginDir()

	// 添加插件目录到监控
	if err := pw.watcher.Add(pw.pluginDir); err != nil {
		pw.logger.Error("添加插件目录到监控失败", zap.Error(err))
		pw.Stop()
		return fmt.Errorf("添加插件目录到监控失败: %w", err)
	}

	pw.logger.Debug("插件文件监控器已启动", zap.String("pluginDir", pw.pluginDir))
	return nil
}

// Stop 停止监控器
func (pw *PluginWatcher) Stop() {
	pw.mu.Lock()
	if !pw.running {
		pw.mu.Unlock()
		return
	}
	pw.running = false
	close(pw.stopChan)
	pw.mu.Unlock()

	// 关闭监控器
	pw.watcher.Close()
	pw.logger.Info("插件文件监控器已停止")
}

// watchLoop 监控文件系统事件
func (pw *PluginWatcher) watchLoop() {
	for {
		select {
		case event, ok := <-pw.watcher.Events:
			if !ok {
				return
			}

			// 忽略临时文件和目录
			if isTempFile(event.Name) || isDirectory(event.Name) {
				continue
			}

			// 只处理.go文件
			if filepath.Ext(event.Name) != ".go" {
				continue
			}

			pw.logger.Debug("检测到文件变更",
				zap.String("path", event.Name),
				zap.String("event", event.Op.String()))

			// 防抖处理（避免短时间内多次触发）
			pw.mu.Lock()
			pw.watchedFiles[event.Name] = time.Now()
			pw.mu.Unlock()

			// 将文件加入处理队列（带延迟）
			go func(path string) {
				time.Sleep(500 * time.Millisecond)
				pw.processChan <- path
			}(event.Name)

		case err, ok := <-pw.watcher.Errors:
			if !ok {
				return
			}
			pw.logger.Error("文件监控错误", zap.Error(err))

		case <-pw.stopChan:
			return
		}
	}
}

// scanLoop 定期扫描插件目录
func (pw *PluginWatcher) scanLoop() {
	ticker := time.NewTicker(pw.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pw.scanPluginDir()

		case <-pw.stopChan:
			return
		}
	}
}

// processQueue 处理文件变更队列
func (pw *PluginWatcher) processQueue() {
	for {
		select {
		case path := <-pw.processChan:
			// 再次检查文件是否存在（可能已被删除）
			if _, err := os.Stat(path); os.IsNotExist(err) {
				pw.handlePluginRemoval(path)
				continue
			}

			// 检查文件修改时间，避免重复处理
			pw.mu.RLock()
			lastProcessed, exists := pw.watchedFiles[path]
			pw.mu.RUnlock()

			fileInfo, err := os.Stat(path)
			if err != nil {
				pw.logger.Error("获取文件信息失败", zap.String("path", path), zap.Error(err))
				continue
			}

			// 如果是新文件或文件确实被修改了
			if !exists || fileInfo.ModTime().After(lastProcessed) {
				pw.handlePluginChange(path)
			}

		case <-pw.stopChan:
			return
		}
	}
}

// scanPluginDir 扫描插件目录
func (pw *PluginWatcher) scanPluginDir() {
	files, err := ioutil.ReadDir(pw.pluginDir)
	if err != nil {
		pw.logger.Error("扫描插件目录失败", zap.Error(err))
		return
	}

	currentFiles := make(map[string]bool)

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".go" {
			continue
		}

		path := filepath.Join(pw.pluginDir, file.Name())
		currentFiles[path] = true

		// 检查是否是新文件
		pw.mu.RLock()
		_, exists := pw.watchedFiles[path]
		pw.mu.RUnlock()

		if !exists {
			pw.logger.Debug("发现新的插件文件", zap.String("path", path))
			pw.processChan <- path
		}
	}

	// 检查是否有文件被删除
	pw.mu.Lock()
	for path := range pw.watchedFiles {
		if !currentFiles[path] {
			pw.logger.Debug("检测到插件文件被删除", zap.String("path", path))
			pw.handlePluginRemoval(path)
			delete(pw.watchedFiles, path)
		}
	}
	pw.mu.Unlock()
}

// handlePluginChange 处理插件文件变更
func (pw *PluginWatcher) handlePluginChange(path string) {
	// 简化处理，打印日志并调用插件管理器的重载方法
	pluginName := getPluginNameFromPath(path)

	pw.logger.Debug("处理插件文件变更",
			zap.String("path", path),
			zap.String("pluginName", pluginName))

	// 检查插件是否已注册
	if _, exists := pw.manager.GetPlugin(pluginName); exists {
		// 重新加载现有插件
		if config.Config.Plugins.HotReload {
			if err := pw.manager.ReloadPlugin(pluginName); err != nil {
				pw.logger.Error("重新加载插件失败",
					zap.String("pluginName", pluginName),
					zap.Error(err))
				metrics.RecordPluginError(pluginName, "hot_reload_failed")
			} else {
				pw.logger.Debug("插件已成功重新加载", zap.String("pluginName", pluginName))
				metrics.RecordPluginReload(pluginName, true)
			}
		} else {
			pw.logger.Debug("热重载功能已禁用", zap.String("pluginName", pluginName))
		}
	} else {
		// 注册新插件
		if config.Config.Plugins.HotReload {
			pw.logger.Debug("发现新插件，尝试动态加载", zap.String("pluginName", pluginName))
			pw.tryLoadNewPlugin(pluginName)
		} else {
			pw.logger.Debug("热重载功能已禁用，无法加载新插件", zap.String("pluginName", pluginName))
		}
	}

	// 更新文件处理时间
	pw.mu.Lock()
	pw.watchedFiles[path] = time.Now()
	pw.mu.Unlock()
}

// handlePluginRemoval 处理插件文件删除
func (pw *PluginWatcher) handlePluginRemoval(path string) {
	pluginName := getPluginNameFromPath(path)

	pw.logger.Debug("处理插件文件删除",
			zap.String("path", path),
			zap.String("pluginName", pluginName))

	// 检查插件是否已注册
	if _, exists := pw.manager.GetPlugin(pluginName); exists {
		// 注销插件
		if err := pw.manager.Unregister(pluginName); err != nil {
			pw.logger.Error("注销插件失败",
				zap.String("pluginName", pluginName),
				zap.Error(err))
		} else {
			pw.logger.Debug("插件已成功注销", zap.String("pluginName", pluginName))
		}
	}
}

// getPluginNameFromPath 从文件路径中提取插件名称
// 使用文件名（不含扩展名）作为插件名称
func getPluginNameFromPath(path string) string {
	baseName := filepath.Base(path)
	return baseName[:len(baseName)-len(filepath.Ext(baseName))]
}

// isTempFile 判断是否为临时文件
func isTempFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".swp" || ext == ".swo" || ext == "~" ||
		filepath.Base(path)[0] == '.'
}

// isDirectory 判断是否为目录
func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

// tryLoadNewPlugin 尝试动态加载新插件
func (pw *PluginWatcher) tryLoadNewPlugin(pluginName string) {
	// 检查.so文件是否存在
	soPath := loader.GetPluginPath(pw.pluginDir, pluginName)
	if _, err := os.Stat(soPath); os.IsNotExist(err) {
		pw.logger.Debug("插件编译文件不存在，跳过加载",
				zap.String("pluginName", pluginName),
				zap.String("expectedPath", soPath))
		metrics.RecordPluginError(pluginName, "plugin_file_not_found")
		return
	}

	// 尝试加载插件
	pluginInstance, err := pw.loader.LoadPlugin(soPath, pluginName)
	if err != nil {
		pw.logger.Error("动态加载插件失败",
			zap.String("pluginName", pluginName),
			zap.Error(err))
		metrics.RecordPluginError(pluginName, "dynamic_load_failed")
		return
	}

	// 注册插件
	if err := pw.manager.Register(pluginInstance); err != nil {
		pw.logger.Error("注册插件失败",
			zap.String("pluginName", pluginName),
			zap.Error(err))
		metrics.RecordPluginError(pluginName, "hot_register_failed")
		// 加载失败，卸载插件
		pw.loader.UnloadPlugin(pluginName)
		return
	}

	pw.logger.Debug("插件已成功动态加载并注册", zap.String("pluginName", pluginName))
}

// PluginManifest 插件清单结构，用于描述插件信息
type PluginManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Version           string   `json:"version"`
	Author            string   `json:"author"`
	Dependencies      []string `json:"dependencies"`
	Conflicts         []string `json:"conflicts"`
	EntryPoint        string   `json:"entry_point"`
	BuildTags         []string `json:"build_tags"`
	RequiredGoVersion string   `json:"required_go_version"`
}

// LoadPluginManifest 加载插件清单文件
func LoadPluginManifest(manifestPath string) (*PluginManifest, error) {
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("读取插件清单失败: %w", err)
	}

	manifest := &PluginManifest{}
	if err := json.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf("解析插件清单失败: %w", err)
	}

	return manifest, nil
}
