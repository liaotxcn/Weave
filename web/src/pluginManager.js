// 插件管理器

class PluginManager {
  constructor() {
    this.plugins = {}
  }

  // 注册插件
  registerPlugin(pluginName, plugin) {
    if (!pluginName || !plugin) {
      console.error('插件名称和插件对象不能为空')
      return
    }

    // 检查插件是否已经存在
    if (this.plugins[pluginName]) {
      console.warn(`插件 ${pluginName} 已经存在，将被覆盖`)
    }

    // 初始化插件 - 处理同步和异步两种情况
    if (plugin.initialize) {
      const initResult = plugin.initialize()
      // 检查是否是Promise
      if (initResult && typeof initResult.then === 'function') {
        initResult.catch(error => {
          console.error(`插件 ${pluginName} 初始化失败:`, error)
        })
      }
    }

    this.plugins[pluginName] = plugin

  }

  // 获取插件
  getPlugin(pluginName) {
    return this.plugins[pluginName]
  }

  // 获取所有插件
  getAllPlugins() {
    return this.plugins
  }

  // 卸载插件
  unregisterPlugin(pluginName) {
    const plugin = this.plugins[pluginName]
    if (plugin && plugin.destroy) {
      plugin.destroy()
    }
    delete this.plugins[pluginName]

  }
}

// 创建单例实例
export default new PluginManager()