<template>
  <div class="plugin-renderer">
    <div v-if="loading" class="plugin-loading fade-in">
      <div class="spinner"></div>
      <span>加载插件中...</span>
    </div>
    <div v-else-if="error" class="plugin-error fade-in">
      <div class="error-icon">⚠️</div>
      <h3>加载失败</h3>
      <p>{{ error }}</p>
      <button @click="refreshPlugin" class="retry-btn">重试</button>
    </div>
    <div v-else-if="pluginComponent" class="plugin-content fade-in">
      <div class="plugin-toolbar">
        <div class="plugin-meta">
          <span class="plugin-name">{{ pluginInfo?.name || '插件' }}</span>
          <span v-if="pluginInfo?.description" class="plugin-sep">·</span>
          <span v-if="pluginInfo?.description" class="plugin-desc">{{ pluginInfo.description }}</span>
          <span v-if="pluginInfo?.version" class="plugin-version">v{{ pluginInfo.version }}</span>
        </div>
        <div class="plugin-actions">
          <button class="toolbar-btn" @click="refreshPlugin">刷新</button>
        </div>
      </div>
      <div class="plugin-body">
        <component :is="pluginComponent" :plugin="plugin" ref="pluginContainer"></component>
      </div>
    </div>
    <div v-else class="plugin-empty fade-in">
      <div class="empty-icon">📦</div>
      <p>请选择一个插件/服务</p>
    </div>
  </div>
</template>

<script setup>
import * as VueRuntimeDOM from 'vue'
import { ref, watch, onMounted, defineComponent } from 'vue'
import { compile as compileTemplate } from '@vue/compiler-dom'

const props = defineProps({
  pluginName: {
    type: String,
    required: true
  },
  pluginManager: {
    type: Object,
    required: true
  }
})

const plugin = ref(null)
const pluginComponent = ref(null)
const pluginContainer = ref(null)
const loading = ref(false)
const error = ref(null)
const pluginInfo = ref(null)

// 辅助函数：更新组件数据
const updateComponentData = async (pluginInstance, component) => {
  if (pluginInstance.loadNotesFromAPI && typeof pluginInstance.loadNotesFromAPI === 'function') {
    await pluginInstance.loadNotesFromAPI()
    if (component.$data.notes !== undefined && pluginInstance.getAllNotes) {
      component.$data.notes = [...(pluginInstance.getAllNotes() || [])]
    }
  }
}

// 加载插件
const loadPlugin = async () => {
  if (!props.pluginName || !props.pluginManager) {
    plugin.value = null
    pluginComponent.value = null
    return
  }

  loading.value = true
  error.value = null

  try {
    // 从pluginManager获取插件实例
    plugin.value = props.pluginManager.getPlugin(props.pluginName)
    if (!plugin.value) {
      throw new Error(`Plugin ${props.pluginName} not found`)
    }

    // 调用插件的初始化方法
    if (typeof plugin.value.initialize === 'function') {
      await plugin.value.initialize()
    }

    // 获取插件的渲染结果
    const renderResult = plugin.value.render()

    // 动态创建Vue组件
    if (renderResult && renderResult.template) {
      // 创建方法映射
      const pluginMethods = {}
      if (renderResult.methods) {
        Object.keys(renderResult.methods).forEach(key => {
          if (typeof renderResult.methods[key] === 'function') {
            pluginMethods[key] = function(...args) {
              const result = renderResult.methods[key].apply(this, args)
              if (result && typeof result.then === 'function') {
                return result.then(async (resolvedResult) => {
                  await updateComponentData(plugin.value, this)
                  return resolvedResult
                })
              } else {
                updateComponentData(plugin.value, this)
                return result
              }
            }
          }
        })
      }

      // 运行时编译模板（使用 function 模式），失败时回退到 template
      let renderFn = null
      try {
        if (compileTemplate) {
          const { code } = compileTemplate(renderResult.template, { mode: 'function' })
          renderFn = new Function('Vue', code)(VueRuntimeDOM)
        }
      } catch (e) {
        console.warn('Runtime compile failed, fallback to template option:', e)
      }

      pluginComponent.value = defineComponent({
        name: `${props.pluginName}-component`,
        props: { plugin: Object },
        // 兼容：优先使用 render 函数，否则使用 template 字符串
        ...(renderFn ? { render: renderFn } : { template: renderResult.template }),
        data() {
          if (renderResult.data && typeof renderResult.data === 'function') {
            const dataResult = renderResult.data.call(plugin.value)
            if (plugin.value.getAllNotes && !dataResult.notes) {
              dataResult.notes = [...(plugin.value.getAllNotes() || [])]
            }
            return dataResult
          }
          return {}
        },
        methods: {
          escapeHtml: (text) => { const div = document.createElement('div'); div.textContent = text; return div.innerHTML },
          formatDate: (dateString) => { try { return new Date(dateString).toLocaleString() } catch (e) { return dateString } },
          ...pluginMethods
        },
        computed: renderResult.computed || {},
        watch: renderResult.watch || {},
        created() {
          const pluginInstance = this.plugin
          if (pluginInstance) {
            ;['addNote', 'updateNote', 'deleteNote', 'getAllNotes', 'loadNotesFromAPI'].forEach(methodName => {
              if (typeof pluginInstance[methodName] === 'function') {
                this[methodName] = pluginInstance[methodName].bind(pluginInstance)
              }
            })
          }
        },
        mounted() {
          if (this.loadNotesFromAPI && typeof this.loadNotesFromAPI === 'function') {
            this.loadNotesFromAPI().then(() => {
              if (this.getAllNotes && typeof this.getAllNotes === 'function') {
                this.notes = [...(this.getAllNotes() || [])]
              }
            })
          }
        }
      })
    } else {
      throw new Error('插件未返回有效的模板')
    }

    // 处理插件样式
    if (renderResult.css) {
      loadPluginCSS(renderResult.css)
    }
  } catch (err) {
    console.error('加载插件失败:', err)
    error.value = `加载插件失败: ${err.message || '未知错误'}`
    pluginComponent.value = null
  } finally {
    loading.value = false
  }
}

// 加载插件CSS
const loadPluginCSS = (css) => {
  if (!css) return
  const styleId = `plugin-css-${props.pluginName}`
  let styleElement = document.getElementById(styleId)
  if (!styleElement) {
    styleElement = document.createElement('style')
    styleElement.id = styleId
    document.head.appendChild(styleElement)
  }
  styleElement.textContent = css
}

// 监听pluginName变化
watch(() => props.pluginName, () => { loadPlugin() }, { immediate: true })

// 导出方法供父组件使用
defineExpose({ refreshPlugin: loadPlugin })

// 组件挂载时加载插件
onMounted(() => { loadPlugin() })
const refreshPlugin = () => { loadPlugin() }
</script>

<style scoped>
.plugin-renderer {
  width: 100%;
  min-height: 100%;
  padding: 24px;
  box-sizing: border-box;
  position: relative;
}

/* 加载状态 */
.plugin-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  color: var(--color-text-tertiary, #6b7280);
  gap: 20px;
  padding: 40px;
  background: linear-gradient(135deg, var(--bg-secondary, #f8fafc), rgba(99, 102, 241, 0.03));
  border-radius: 16px;
  border: 1px solid var(--border-light, #e5e7eb);
  animation: fadeInUp 0.4s ease-out;
}

.spinner {
  width: 48px;
  height: 48px;
  border: 3.5px solid rgba(99, 102, 241, 0.15);
  border-top-color: var(--primary-500, #6366f1);
  border-radius: 50%;
  animation: spin 0.9s linear infinite;
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.2);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 错误状态 */
.plugin-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  color: var(--error-600, #dc2626);
  gap: 16px;
  padding: 40px;
  text-align: center;
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.04), rgba(252, 165, 165, 0.02));
  border-radius: 16px;
  border: 1.5px solid rgba(239, 68, 68, 0.15);
  animation: fadeInUp 0.4s ease-out;
}

.error-icon {
  font-size: 3.5rem;
  filter: drop-shadow(0 6px 8px rgba(239, 68, 68, 0.15));
  animation: shake 0.5s ease-in-out;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  20%, 60% { transform: translateX(-8px); }
  40%, 80% { transform: translateX(8px); }
}

.plugin-error h3 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--error-700, #b91c1c);
  letter-spacing: -0.01em;
}

.plugin-error p {
  margin: 0;
  max-width: 480px;
  line-height: 1.7;
  font-size: 14px;
  color: var(--error-600, #dc2626);
}

.retry-btn {
  background: linear-gradient(135deg, #ef4444, #dc2626);
  color: white;
  border: none;
  padding: 10px 24px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  margin-top: 12px;
  font-weight: 600;
  font-size: 14px;
  box-shadow: 0 4px 14px rgba(239, 68, 68, 0.25);
  letter-spacing: 0.01em;
}

.retry-btn:hover {
  background: linear-gradient(135deg, #dc2626, #b91c1c);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(239, 68, 68, 0.35);
}

.retry-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 10px rgba(239, 68, 68, 0.25);
}

/* 插件内容区域 */
.plugin-content {
  width: 100%;
  min-height: 400px;
  overflow: auto;
  background: white;
  border-radius: 16px;
  border: 1px solid var(--border-light, #e5e7eb);
  box-shadow: var(--shadow-card, 0 1px 3px rgba(0,0,0,0.05));
  display: flex;
  flex-direction: column;
  animation: fadeInUp 0.45s cubic-bezier(0.4, 0, 0.2, 1);
}

/* 工具栏 */
.plugin-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color, #f1f5f9);
  background: linear-gradient(180deg, #ffffff, #f8fafc);
  position: sticky;
  top: 0;
  z-index: 10;
  backdrop-filter: blur(8px);
}

.plugin-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.plugin-meta .plugin-name {
  color: var(--primary-700, #4338ca);
  font-weight: 700;
  font-size: 17px;
  letter-spacing: -0.01em;
}

.plugin-meta .plugin-sep {
  color: var(--color-text-tertiary, #9ca3af);
  margin: 0 4px;
  font-weight: 300;
}

.plugin-meta .plugin-desc {
  color: var(--color-text-secondary, #64748b);
  font-size: 13px;
  font-weight: 500;
}

.plugin-meta .plugin-version {
  margin-left: 8px;
  color: white;
  font-size: 11px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--primary-500, #6366f1), var(--primary-600, #4f46e5));
  border-radius: 8px;
  padding: 3px 10px;
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.2);
  letter-spacing: 0.02em;
}

.plugin-actions .toolbar-btn {
  background: linear-gradient(135deg, var(--primary-500, #6366f1), var(--primary-600, #4f46e5));
  color: #fff;
  border: none;
  padding: 8px 18px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 600;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 3px 10px rgba(99, 102, 241, 0.25);
  letter-spacing: 0.01em;
}

.toolbar-btn:hover {
  background: linear-gradient(135deg, var(--primary-600, #4f46e5), var(--primary-700, #4338ca));
  transform: translateY(-2px);
  box-shadow: 0 6px 18px rgba(99, 102, 241, 0.35);
}

.toolbar-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.25);
}

.plugin-body {
  padding: 24px;
  overflow: auto;
  height: 100%;
  flex: 1;
}

/* 空状态 */
.plugin-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  color: var(--text-muted, #6b7280);
  gap: 18px;
  padding: 50px;
  background: linear-gradient(135deg, var(--bg-secondary, #f8fafc), rgba(99, 102, 241, 0.02));
  border-radius: 16px;
  border: 1px dashed var(--border-medium, #cbd5e1);
  animation: fadeInUp 0.4s ease-out;
}

.empty-icon {
  font-size: 4rem;
  opacity: 0.55;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  50% { transform: translateY(-12px) rotate(3deg); }
}

.plugin-empty p {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  color: var(--color-text-secondary, #64748b);
  letter-spacing: -0.01em;
}

/* 滚动条样式 */
.plugin-content::-webkit-scrollbar,
.plugin-body::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.plugin-content::-webkit-scrollbar-thumb,
.plugin-body::-webkit-scrollbar-thumb {
  background: linear-gradient(180deg, rgba(99, 102, 241, 0.3), rgba(139, 92, 246, 0.3));
  border-radius: 10px;
  border: 2px solid transparent;
  background-clip: content-box;
}

.plugin-content::-webkit-scrollbar-thumb:hover,
.plugin-body::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, rgba(99, 102, 241, 0.5), rgba(139, 92, 246, 0.5));
  background-clip: content-box;
}

/* 淡入动画 */
.fade-in {
  animation: fadeInUp 0.35s ease-out both;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .plugin-loading,
  .plugin-error,
  .plugin-empty,
  .plugin-content {
    min-height: 320px;
    padding: 20px;
    border-radius: 12px;
  }

  .spinner {
    width: 38px;
    height: 38px;
  }

  .error-icon,
  .empty-icon {
    font-size: 3rem;
  }

  .plugin-error h3 {
    font-size: 19px;
  }

  .plugin-toolbar {
    padding: 12px 16px;
    flex-wrap: wrap;
    gap: 10px;
  }

  .plugin-body {
    padding: 18px;
  }
}

@media (max-width: 480px) {
  .plugin-renderer {
    padding: 16px;
  }

  .plugin-loading,
  .plugin-error,
  .plugin-empty,
  .plugin-content {
    min-height: 280px;
    padding: 16px;
    border-radius: 10px;
  }

  .spinner {
    width: 32px;
    height: 32px;
  }

  .error-icon,
  .empty-icon {
    font-size: 2.5rem;
  }

  .plugin-error h3 {
    font-size: 17px;
  }

  .plugin-error p {
    font-size: 13px;
  }

  .toolbar-btn {
    padding: 7px 14px;
    font-size: 12px;
  }

  .plugin-body {
    padding: 14px;
  }
}
</style>