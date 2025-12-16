<script setup>
import { ref, onMounted, computed } from 'vue'
import PluginRenderer from './components/PluginRenderer.vue'
import AuthContainer from './components/AuthContainer.vue'
import AIAssistant from './components/AIAssistant.vue'
import PluginItem from './components/PluginItem.vue'
import MenuItem from './components/MenuItem.vue'
import pluginManager from './pluginManager.js'
import HelloPlugin from './plugins/HelloPlugin.js'
import NotePlugin from './plugins/NotePlugin.js'
import { authService } from './services/auth.js'
import UserCenter from './components/UserCenter.vue'
import TeamsCenter from './components/TeamsCenter.vue'
// 导入需要的 Element Plus 图标
import { Search } from '@element-plus/icons-vue'

// 导入共享样式
import './styles/shared.css'
import './styles/patterns.css'
import './styles/animations.css'
const appVersion = '1.0.0'

// 认证状态
const isAuthenticated = ref(false)
const currentUser = ref(null)
const showMenu = ref(false)

// 可用插件列表
const availablePlugins = ref([])
const selectedPlugin = ref(null)
const selectedSection = ref('plugins')
const pluginRendererRef = ref(null)

// 侧边栏菜单数据
const menuItems = [
  { icon: '🏢', label: 'Services', value: 'services' },
  { icon: '🔌', label: 'Plugins', value: 'plugins' },
  { icon: '👥', label: '团队', value: 'teams' },
  { icon: '👤', label: '个人', value: 'personal' },
  { icon: '🔒', label: '安全中心', value: 'security' }
]
// 新增：侧栏搜索关键词与过滤列表
const pluginKeyword = ref('')
const filteredPlugins = computed(() => {
  const kw = pluginKeyword.value.trim().toLowerCase()
  if (!kw) return availablePlugins.value
  return availablePlugins.value.filter(p => {
    const name = (p.name || '').toLowerCase()
    const desc = (p.info?.description || '').toLowerCase()
    return name.includes(kw) || desc.includes(kw)
  })
})

// 引用插件渲染器（重复声明已移除）

// 初始化应用
onMounted(() => {
  // 检查用户是否已登录
  checkAuthentication()
  
  // 注册插件
  registerPlugins()
  
  // 获取可用插件信息
  updateAvailablePlugins()
})

// 注册插件
const registerPlugins = () => {
  // 注册Hello插件
  const helloPlugin = new HelloPlugin()
  pluginManager.registerPlugin('hello', helloPlugin)
  
  // 注册Note插件
  const notePlugin = new NotePlugin()
  pluginManager.registerPlugin('note', notePlugin)
  
  // 这里可以注册更多插件
}

// 更新可用插件列表
const updateAvailablePlugins = () => {
  const plugins = pluginManager.getAllPlugins()
  availablePlugins.value = Object.keys(plugins).map(key => {
    const plugin = plugins[key]
    return {
      name: key,
      info: plugin.getInfo()
    }
  })
  
  // 默认选择第一个插件
  if (availablePlugins.value.length > 0 && !selectedPlugin.value) {
    selectedPlugin.value = availablePlugins.value[0].name
  }
}

// 选择插件
const selectPlugin = (pluginName) => {
  selectedSection.value = 'plugins'
  selectedPlugin.value = pluginName
}

// 检查用户认证状态
const checkAuthentication = () => {
  const authStatus = authService.isAuthenticated()

  isAuthenticated.value = authStatus
  if (authStatus) {
    currentUser.value = authService.getCurrentUser()

  }
}

// 处理认证成功
const handleAuthSuccess = () => {
  checkAuthentication()
  // 登录成功后刷新当前插件，使其加载用户数据
  if (pluginRendererRef.value && typeof pluginRendererRef.value.refreshPlugin === 'function') {
    pluginRendererRef.value.refreshPlugin()
  }
}

// 处理用户登出
const handleLogout = () => {
  authService.logout()
  isAuthenticated.value = false
  currentUser.value = null
  selectedPlugin.value = null
}
const handleMenuSelect = (key) => {
  if (key === 'services') {
    selectedSection.value = 'services'
    selectedPlugin.value = null
  } else if (key === 'plugins') {
    selectedSection.value = 'plugins'
    // 如果没有选中插件，默认选择第一个
    if (!selectedPlugin.value && availablePlugins.value.length > 0) {
      selectedPlugin.value = availablePlugins.value[0].name
    }
  } else if (key === 'teams') {
    selectedSection.value = 'teams'
    selectedPlugin.value = null
  } else if (key === 'personal') {
    selectedSection.value = 'personal'
    selectedPlugin.value = null
  } else if (key === 'security') {
    selectedSection.value = 'security'
    selectedPlugin.value = null
  } else if (key === 'logout') {
    handleLogout()
  }
  showMenu.value = false
}
</script>

<template>
  <div class="app">
    <!-- 用户未登录时显示登录/注册界面 -->
    <AuthContainer v-if="!isAuthenticated" @auth-success="handleAuthSuccess" />
    
    <!-- 用户已登录时显示主应用界面 -->
    <template v-else>
      <el-header class="app-header">
        <div class="header-content">
          <!-- 左侧品牌区域 -->
          <div class="brand-section">
            <div class="brand-container">
              <el-avatar 
                size="64px" 
                class="brand-avatar"
                src="/logo.png"
              />
              <div class="brand-text">
                <h1 class="brand-name">Weave</h1>
              </div>
            </div>
          </div>
          
          <!-- 右侧用户信息区域 -->
          <div class="user-info">
            <el-dropdown
              @command="handleMenuSelect"
              trigger="click"
              placement="bottom"
            >
              <div class="user-dropdown-trigger">
                <el-avatar 
                  size="40px" 
                  class="user-avatar"
                  :src="currentUser?.avatar || ''"
                  icon="User"
                />
                <div class="user-info-text">
                  <span class="user-name">{{ currentUser?.username }}</span>
                  <span class="user-role-tag">
                    <el-tag size="small" type="primary" effect="plain">用户</el-tag>
                  </span>
                </div>
                <el-icon class="arrow-icon">
                  <ArrowDown />
                </el-icon>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item 
                    command="logout" 
                    class="logout-item"
                    icon="SwitchButton"
                  >
                    退出登录
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </el-header>
      
      <main class="app-main">
        <aside class="sidebar">
          <!-- 主菜单 -->
          <nav class="sidebar-nav">
            <el-menu
              class="menu-list"
              @select="handleMenuSelect"
              background-color="transparent"
              text-color="var(--text-secondary)"
              active-text-color="var(--primary)"
              router="false"
              unique-opened
            >
              <el-menu-item
                v-for="(item, index) in menuItems"
                :key="item.value"
                :index="item.value"
                :class="{ active: selectedSection === item.value }"
              >
                <span class="menu-icon">{{ item.icon }}</span>
                <span>{{ item.label }}</span>
              </el-menu-item>
            </el-menu>
          </nav>
        </aside>
        
        <!-- 右侧内容区域 -->
        <div class="content-area">
          <header class="content-header">
            <h2 class="section-title">
              <span class="title-icon">
                {{ 
                  selectedSection === 'services' ? '🏢' :
                  selectedSection === 'plugins' ? '🔌' :
                  selectedSection === 'teams' ? '👥' :
                  selectedSection === 'personal' ? '👤' :
                  selectedSection === 'security' ? '🔒' :
                  '📄'
                }}
              </span>
              <span class="title-text">
                {{ 
                  selectedSection === 'services' ? 'Services' :
                  selectedSection === 'plugins' ? 'Plugins' :
                  selectedSection === 'teams' ? '团队' :
                  selectedSection === 'personal' ? '个人' :
                  selectedSection === 'security' ? '安全中心' :
                  '内容'
                }}
              </span>
            </h2>
          </header>
          
          <main class="content-body">
            <transition name="content-switch" mode="out-in" appear>
              <!-- Services 内容 -->
              <div v-if="selectedSection === 'services'" class="services-content">
                <div class="service-card">
                  <h3>🏢 Service</h3>
                  <p>集研发、聚合、管理为一体</p>
                </div>
              </div>
              
              <!-- Plugins 内容 -->
              <div v-else-if="selectedSection === 'plugins'" class="plugins-content">
                <div class="plugins-layout">
                  <!-- 插件列表区域 -->
                  <div class="plugins-sidebar">
                    <el-card shadow="hover" class="plugin-list-card">
                      <template #header>
                        <div class="plugins-header">
                          <h3 class="plugin-list-title">插件列表</h3>
                        </div>
                      </template>
                      <div class="plugins-tools">
                        <el-input
                          v-model="pluginKeyword"
                          placeholder="搜索插件..."
                          clearable
                          size="small"
                          class="plugins-search"
                        >
                          <template #prefix>
                            <el-icon class="el-input__icon"><Search /></el-icon>
                          </template>
                        </el-input>
                        <div class="plugins-count">共 {{ availablePlugins.length }}，匹配 {{ filteredPlugins.length }}</div>
                      </div>
                      <el-scrollbar class="plugins-scrollbar">
                        <div class="plugins-list">
                          <PluginItem
                            v-for="pluginInfo in filteredPlugins"
                            :key="pluginInfo.name"
                            :name="pluginInfo.name"
                            :description="pluginInfo.info?.description"
                            :badge-count="pluginInfo.info?.noteCount"
                            :is-active="selectedPlugin === pluginInfo.name"
                            @select="selectPlugin"
                          />
                        </div>
                      </el-scrollbar>
                    </el-card>
                  </div>
                  
                  <!-- 插件内容区域 -->
                  <div class="plugins-main">
                    <el-card shadow="hover" class="plugin-content-card">
                      <div v-if="selectedPlugin" class="plugin-renderer-container">
                        <PluginRenderer 
                          :plugin-name="selectedPlugin"
                          :plugin-manager="pluginManager"
                          ref="pluginRendererRef"
                        />
                      </div>
                      <div v-else class="no-plugin-selected">
                        <el-empty
                          description="请选择一个插件"
                          image-size="120"
                        >
                          <template #image>
                            <div class="empty-icon">🔌</div>
                          </template>
                          <template #description>
                            <div class="empty-description">
                              <h3>请选择一个插件</h3>
                              <p>从左侧插件列表中选择要使用的插件</p>
                            </div>
                          </template>
                        </el-empty>
                      </div>
                    </el-card>
                  </div>
                </div>
              </div>
              
              <!-- Teams 内容 -->
              <div v-else-if="selectedSection === 'teams'" class="teams-content">
                <TeamsCenter />
              </div>
              
              <!-- Personal 内容 -->
              <div v-else-if="selectedSection === 'personal'" class="personal-content">
                <UserCenter 
                  :current-user="currentUser"
                  @updated-user="currentUser = $event"
                />
              </div>
              
              <!-- Security 内容 -->
              <div v-else-if="selectedSection === 'security'" class="security-content">
                <div class="security-card">
                  <h3>🔒 安全中心</h3>
                  <p>权限管控、认证加密、沙盒环境等</p>
                </div>
              </div>
            </transition>
          </main>
        </div>
      </main>
      
      <el-footer class="app-footer">
        <div class="footer-content">
          <!-- <div class="footer-left"> -->
          <div class="footer-right">
            <span class="version">Weave v{{ appVersion }}</span>
            <el-link 
              href="https://github.com/liaotxcn/Weave" 
              target="_blank" 
              type="primary"
              :underline="false"
              class="github-link"
            >
              <!-- 使用Element Plus GitHub图标 -->
              <el-icon class="github-icon"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path></svg></el-icon>
            </el-link>
          </div>
        </div>
      </el-footer>
      
      <!-- AI智能助手（仅在登录后显示） -->
      <AIAssistant v-if="isAuthenticated" />
    </template>
  </div>
</template>

<style scoped>
/* 应用主样式 */
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

/* 头部样式 - 优化版本 */
.app-header {
  background: linear-gradient(135deg, 
    rgba(99, 102, 241, 1) 0%, 
    rgba(79, 70, 229, 1) 25%,
    rgba(67, 56, 202, 1) 75%,
    rgba(55, 48, 163, 1) 100%
  );
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  color: white;
  padding: var(--space-1.5) 0; /* 增加上下内边距，提高背景高度 */
  box-shadow: 
    0 4px 20px rgba(0, 0, 0, 0.1),
    0 2px 6px rgba(0, 0, 0, 0.08);
  position: sticky;
  top: 0;
  z-index: 100;
  transition: all 0.3s ease;
}

/* 头部装饰效果 */
.app-header::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, 
    rgba(255, 255, 255, 0.8) 0%, 
    rgba(255, 255, 255, 0.3) 100%
  );
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 var(--space-6);
  min-height: 88px; /* 增加最小高度，提高整体背景高度 */
  position: relative;
  z-index: 1;
}

/* 品牌区域样式 */
.brand-section {
  display: flex;
  align-items: center;
}

.brand-container {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  position: relative;
  padding: var(--space-2) 0;
}

.brand-avatar {
  width: 72px;
  height: 72px;
  border-radius: 16px;
  transition: all 0.3s ease;
  box-shadow: 
    0 4px 12px rgba(0, 0, 0, 0.15),
    0 2px 4px rgba(0, 0, 0, 0.1);
  border: 2px solid rgba(255, 255, 255, 0.1) !important;
  background: #f1f5f9;
}

.brand-avatar:hover {
  transform: scale(1.05);
  box-shadow: 
    0 6px 18px rgba(0, 0, 0, 0.2),
    0 3px 6px rgba(0, 0, 0, 0.15);
}

.brand-name {
  font-size: 2.4rem;
  font-weight: 400;
  margin: 0;
  color: var(--el-text-color-primary);
  letter-spacing: 0.01em;
  text-shadow: none;
  position: relative;
  transition: all 0.3s ease;
  font-family: var(--el-font-family);
}

.brand-name::after {
  content: '';
  position: absolute;
  bottom: -4px;
  left: 0;
  width: 0;
  height: 2px;
  background: var(--el-color-primary);
  transition: width 0.3s ease;
}

.brand-container:hover .brand-name::after {
  width: 100%;
}

.brand-highlight {
  position: relative;
  background: linear-gradient(135deg, 
    #ffd700 0%, 
    #ffb800 50%, 
    #ffa500 100%
  );
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  font-weight: 600;
  margin-left: 4px;
  text-shadow: none;
  font-family: var(--el-font-family);
}

.brand-subtitle {
  font-size: var(--font-size-xs);
  opacity: 0;
  margin: 4px 0 0;
  color: rgba(0, 0, 0, 0.8);
  letter-spacing: 0.01em;
  font-weight: var(--font-weight-medium);
  transform: translateY(-4px);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.1);
  line-height: 1.2;
}

.app-header:hover .brand-subtitle {
  opacity: 1;
  transform: translateY(0);
}

/* 用户信息区域样式 */
.user-info {
  display: flex;
  align-items: center;
}

.user-dropdown-trigger {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-4);
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 50px;
  color: white;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: var(--font-size-sm);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.user-dropdown-trigger:hover {
  background: rgba(255, 255, 255, 0.15);
  border-color: rgba(255, 255, 255, 0.3);
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15);
}

.user-avatar {
  border: 2px solid rgba(255, 255, 255, 0.3);
  transition: all 0.3s ease;
}

.user-dropdown-trigger:hover .user-avatar {
  border-color: rgba(255, 255, 255, 0.5);
  transform: scale(1.05);
}

.user-info-text {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 0;
}

.user-name {
  font-weight: var(--font-weight-medium);
  color: white;
  font-size: var(--font-size-sm);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-role-tag {
  margin-top: 2px;
}

:deep(.el-tag) {
  font-size: var(--font-size-xs);
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
  color: white;
}

.arrow-icon {
  font-size: 14px;
  color: white;
  transition: transform 0.3s ease;
}

.user-dropdown-trigger:hover .arrow-icon {
  transform: translateY(2px);
}

/* 下拉菜单样式优化 */
:deep(.el-dropdown-item__icon) {
  font-size: 16px;
  margin-right: 8px;
}

:deep(.el-dropdown-menu) {
  border-radius: var(--radius-lg);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  border: 1px solid rgba(0, 0, 0, 0.08);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  padding: var(--space-1) 0;
  min-width: 140px;
  background: rgba(255, 255, 255, 0.95);
}

:deep(.el-dropdown-item) {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-4);
  transition: all 0.2s ease;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  position: relative;
  overflow: hidden;
  height: auto;
  line-height: 1.4;
}

:deep(.el-dropdown-item::before) {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--primary);
  transform: scaleY(0);
  transition: transform 0.2s ease;
}

:deep(.el-dropdown-item:hover) {
  background: linear-gradient(90deg, 
    rgba(99, 102, 241, 0.1) 0%, 
    rgba(99, 102, 241, 0.05) 100%
  );
  color: var(--primary);
}

:deep(.el-dropdown-item:hover::before) {
  transform: scaleY(1);
}

:deep(.el-dropdown-item.logout-item) {
  color: var(--error);
}

:deep(.el-dropdown-item.logout-item:hover) {
  background: linear-gradient(90deg, 
    rgba(239, 68, 68, 0.1) 0%, 
    rgba(239, 68, 68, 0.05) 100%
  );
  color: var(--error);
}

:deep(.el-dropdown-item.logout-item:hover::before) {
  background: var(--error);
}

/* 主内容区域样式 */
.app-main {
  flex: 1;
  display: flex;
  min-height: 0;
  background: var(--color-background);
  /* 为固定侧边栏留出空间 */
  padding-left: 280px;
}

/* 侧边栏样式 */
.sidebar {
  width: 280px;
  background: white;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.04);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: fixed;
  top: 0;
  left: 0;
  height: 100vh;
  z-index: 10;
  overflow-y: auto;
  /* 考虑顶部导航栏的高度 */
  padding-top: 72px; /* 与app-header的高度保持一致 */
}

/* 添加侧边栏装饰元素 */
.sidebar::after {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.1), transparent);
  border-radius: 0 0 0 100%;
  z-index: 0;
}

.sidebar-nav {
  padding: var(--space-4) 0;
  border-bottom: 1px solid var(--border);
  position: relative;
  z-index: 1;
}

.menu-list {
  list-style: none;
  margin: 0;
  padding: var(--space-2);
  border-radius: var(--radius-md);
  margin: 0 var(--space-3);
  overflow: hidden;
  background: var(--background);
  backdrop-filter: blur(8px);
  border: none !important;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

/* Element Plus 菜单组件样式优化 */
:deep(.el-menu) {
  background: transparent !important;
  border-right: none !important;
}

:deep(.el-menu-item) {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4) !important;
  border: none !important;
  background: transparent !important;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  text-align: left;
  border-radius: var(--radius-md);
  margin-bottom: var(--space-1) !important;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

:deep(.el-menu-item:hover) {
  background: linear-gradient(90deg, 
    rgba(99, 102, 241, 0.1) 0%, 
    rgba(99, 102, 241, 0.05) 100% 
  ) !important;
  color: var(--primary) !important;
  transform: translateX(2px);
}

:deep(.el-menu-item.is-active) {
  background: linear-gradient(90deg, 
    rgba(99, 102, 241, 0.15) 0%, 
    rgba(99, 102, 241, 0.08) 100% 
  ) !important;
  color: var(--primary) !important;
  font-weight: var(--font-weight-semibold);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.15);
}

:deep(.el-menu-item .menu-icon) {
  font-size: 1.3em;
  width: 28px;
  text-align: center;
  transition: transform 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 28px;
  background: rgba(99, 102, 241, 0.05);
  border-radius: var(--radius-md);
}

:deep(.el-menu-item:hover .menu-icon) {
  transform: scale(1.1);
  background: rgba(99, 102, 241, 0.15);
}

:deep(.el-menu-item.is-active .menu-icon) {
  background: rgba(99, 102, 241, 0.2);
}

/* 优化侧边栏整体阴影和边界 */
.sidebar:hover {
  box-shadow: 0 0 25px rgba(0, 0, 0, 0.06);
}

/* 插件内容区域样式 */
.plugins-content {
  height: 100%;
}

.plugins-layout {
  display: flex;
  height: 100%;
  gap: var(--space-4);
}

.plugins-sidebar {
  width: 320px;
  min-width: 280px;
  display: flex;
  flex-direction: column;
}

/* Element Plus 卡片样式优化 */
.plugin-list-card {
  height: calc(100vh - 200px);
  display: flex;
  flex-direction: column;
}

.plugin-content-card {
  height: calc(100vh - 200px);
  display: flex;
  flex-direction: column;
}

.plugin-list-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.plugins-tools {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

:deep(.el-input) {
  width: 100%;
}

.plugins-count {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.plugins-scrollbar {
  flex: 1;
  overflow: hidden;
}

.plugins-list {
  padding: var(--space-1);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.plugins-main {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.plugin-renderer-container {
  height: 100%;
  width: 100%;
  overflow: auto;
}

/* 空状态样式 */
.no-plugin-selected {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-icon {
  font-size: 4rem;
  opacity: 0.5;
}

.empty-description h3 {
  margin: 0 0 var(--space-2) 0;
  font-size: var(--font-size-lg);
  color: var(--color-text-primary);
}

.empty-description p {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .plugins-layout {
    flex-direction: column;
  }
  
  .plugins-sidebar {
    width: 100%;
    max-height: 300px;
  }
  
  .plugins-main {
    max-height: none;
  }
}

@media (max-width: 768px) {
  .plugins-sidebar {
    padding: var(--space-3);
  }
  
  .plugins-main {
    padding: var(--space-3);
  }
}

/* 原有的插件子菜单样式保留用于向后兼容 */
.plugin-submenu {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.submenu-header {
  padding: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.submenu-header h3 {
  margin: 0 0 var(--space-3) 0;
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.sidebar-tools {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.sidebar-search {
  width: 100%;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  transition: all 0.2s ease;
}

.sidebar-search:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.sidebar-count {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.plugin-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-2);
}

/* 插件列表项样式由PluginItem组件提供 */


/* 内容区域样式 */
.content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.content-header {
  padding: var(--space-5) var(--space-6) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  background: linear-gradient(135deg, 
    rgba(255, 255, 255, 1) 0%, 
    rgba(248, 250, 252, 1) 100%
  );
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
  position: relative;
  overflow: hidden;
}

.content-header::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  height: 3px;
  width: 0;
  background: linear-gradient(90deg, 
    rgba(99, 102, 241, 1) 0%, 
    rgba(139, 92, 246, 1) 100%
  );
  transition: width 0.5s cubic-bezier(0.4, 0, 0.2, 1);
}

.content-header:hover::before {
  width: 100%;
}

.section-title {
  margin: 0;
  font-size: 1.5rem;
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  position: relative;
  z-index: 1;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.title-icon {
  font-size: 1.8rem;
  padding: var(--space-2);
  background: linear-gradient(135deg, 
    rgba(99, 102, 241, 0.15) 0%, 
    rgba(139, 92, 246, 0.15) 100%
  );
  border-radius: var(--radius-lg);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
}

.title-text {
  background: linear-gradient(135deg, 
    rgba(99, 102, 241, 1) 0%, 
    rgba(139, 92, 246, 1) 100%
  );
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  position: relative;
}

.title-text::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  width: 100%;
  height: 2px;
  background: linear-gradient(90deg, 
    rgba(99, 102, 241, 0.3) 0%, 
    rgba(139, 92, 246, 0.3) 100%
  );
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.content-header:hover .section-title {
  transform: translateX(5px);
}

.content-header:hover .title-icon {
  transform: scale(1.1) rotate(3deg);
  background: linear-gradient(135deg, 
    rgba(99, 102, 241, 0.25) 0%, 
    rgba(139, 92, 246, 0.25) 100%
  );
}

.content-header:hover .title-text::after {
  transform: scaleX(1);
}

.content-body {
  flex: 1;
  padding: var(--space-6);
  overflow-y: auto;
}

/* 内容卡片样式 */
.services-content,
.plugins-content,
.security-content {
  height: 100%;
}

.service-card,
.security-card {
  background: white;
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid var(--color-border);
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  text-align: center;
}

.service-card h3,
.security-card h3 {
  margin: 0 0 var(--space-3) 0;
  font-size: 1.25rem;
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.service-card p,
.security-card p {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--font-size-base);
}

.no-plugin-selected {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-tertiary);
}

.empty-state {
  text-align: center;
  padding: var(--space-8);
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: var(--space-4);
  display: block;
  opacity: 0.6;
}

.empty-state h3 {
  margin: 0 0 var(--space-2) 0;
  font-size: 1.25rem;
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.empty-state p {
  margin: 0;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-sm);
}

/* 过渡动画 */
.content-switch-enter-active,
.content-switch-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.content-switch-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.content-switch-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

/* 底部样式 */
.app-footer {
  background: white;
  border-top: 1px solid var(--color-border);
  padding: var(--space-4) 0;
  margin-top: auto;
  box-shadow: 0 -2px 10px rgba(0, 0, 0, 0.03);
}

.footer-content {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 var(--space-5);
}

.footer-right {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.version {
  font-size: var(--font-size-sm);
  color: var(--color-text-tertiary);
  font-weight: var(--font-weight-medium);
  letter-spacing: 0.2px;
}

:deep(.github-link) {
  display: flex;
  align-items: center;
  color: var(--color-text-tertiary);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  text-decoration: none;
}

:deep(.github-link:hover) {
  color: var(--color-primary);
  transform: translateY(-2px);
}

:deep(.github-icon) {
  font-size: 1.5rem;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

:deep(.github-link:hover .github-icon) {
  transform: scale(1.2) rotate(5deg);
  filter: drop-shadow(0 2px 4px rgba(99, 102, 241, 0.3));
}

/* 响应式设计 */
@media (max-width: 768px) {
  .sidebar {
    width: 240px;
  }
  
  .content-body {
    padding: var(--space-4);
  }
  
  .brand-text h1 {
    font-size: 1.5rem;
  }
  
  .brand-text p {
    display: none;
  }
}

@media (max-width: 640px) {
  .app-main {
    flex-direction: column;
  }
  
  .sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--color-border);
  }
  
  .sidebar-nav {
    padding: var(--space-2) 0;
  }
  
  .menu-list {
    display: flex;
    overflow-x: auto;
    padding: 0 var(--space-2);
  }
  
  .content-header {
    padding: var(--space-3) var(--space-4) var(--space-2);
  }
  
  .content-body {
    padding: var(--space-3);
  }
}
</style>