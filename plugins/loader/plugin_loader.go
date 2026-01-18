// Package loader 提供插件动态加载功能
package loader

import (
	"fmt"
	"path/filepath"
	"plugin"
	"sync"

	"weave/pkg"
	"weave/plugins/core"

	"go.uber.org/zap"
)

// PluginLoader 负责动态加载和卸载插件
type PluginLoader struct {
	loadedPlugins map[string]*plugin.Plugin
	mutex         sync.RWMutex
	logger        *pkg.Logger
}

// NewPluginLoader 创建插件加载器实例
func NewPluginLoader(logger *pkg.Logger) *PluginLoader {
	return &PluginLoader{
		loadedPlugins: make(map[string]*plugin.Plugin),
		mutex:         sync.RWMutex{},
		logger:        logger,
	}
}

// LoadPlugin 动态加载插件
// 参数:
// - pluginPath: 插件文件路径(.so文件)
// - pluginName: 插件名称
// 返回值:
// - core.Plugin: 加载的插件实例
// - error: 加载过程中的错误
func (pl *PluginLoader) LoadPlugin(pluginPath string, pluginName string) (core.Plugin, error) {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	// 检查插件是否已经加载
	if _, exists := pl.loadedPlugins[pluginName]; exists {
		// 先卸载已加载的插件
		if err := pl.UnloadPlugin(pluginName); err != nil {
			pl.logger.Warn("卸载已加载的插件失败", zap.String("plugin", pluginName), zap.Error(err))
		}
	}

	// 加载插件
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("加载插件失败: %w", err)
	}

	// 查找插件入口点
	symbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("查找插件入口点失败: %w", err)
	}

	// 类型断言，确保它是一个函数
	constructor, ok := symbol.(func() core.Plugin)
	if !ok {
		return nil, fmt.Errorf("插件入口点类型错误")
	}

	// 创建插件实例
	pluginInstance := constructor()

	// 验证插件名称
	if pluginInstance.Name() != pluginName {
		return nil, fmt.Errorf("插件名称不匹配: 期望 %s, 实际 %s", pluginName, pluginInstance.Name())
	}

	// 保存插件引用
	pl.loadedPlugins[pluginName] = p
	pl.logger.Debug("插件加载成功", zap.String("plugin", pluginName), zap.String("path", pluginPath))

	return pluginInstance, nil
}

// UnloadPlugin 卸载插件
func (pl *PluginLoader) UnloadPlugin(pluginName string) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	// 检查插件是否已加载
	_, exists := pl.loadedPlugins[pluginName]
	if !exists {
		return nil
	}

	// Go标准库的plugin包不提供显式关闭插件的机制
	// 插件会在进程结束时自动卸载

	// 从映射中删除
	delete(pl.loadedPlugins, pluginName)
	pl.logger.Debug("插件卸载成功", zap.String("plugin", pluginName))

	return nil
}

// GetLoadedPlugin 检查插件是否已加载
func (pl *PluginLoader) GetLoadedPlugin(pluginName string) bool {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()

	_, exists := pl.loadedPlugins[pluginName]
	return exists
}

// GetPluginPath 获取插件的绝对路径
func GetPluginPath(pluginDir string, pluginName string) string {
	return filepath.Join(pluginDir, fmt.Sprintf("%s.so", pluginName))
}
