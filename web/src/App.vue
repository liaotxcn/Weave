<script setup>
import { ref, onMounted, computed } from 'vue'
import PluginRenderer from './components/PluginRenderer.vue'
import AuthContainer from './components/AuthContainer.vue'
import AIAssistant from './components/AIAssistant.vue'
import PluginItem from './components/PluginItem.vue'
import pluginManager from './pluginManager.js'
import HelloPlugin from './plugins/HelloPlugin.js'
import NotePlugin from './plugins/NotePlugin.js'
import { authService } from './services/auth.js'
import UserCenter from './components/UserCenter.vue'
import TeamsCenter from './components/TeamsCenter.vue'
// 导入需要的 Element Plus 图标
import { Search } from '@element-plus/icons-vue'

import './styles/app.css'
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
                  <div class="plugins-sidebar card-base">
                    <div class="sidebar-header">
                      <h3 class="plugin-list-title">📦 插件列表</h3>
                    </div>
                    <div class="plugins-tools">
                      <div class="search-box el-input-reset">
                        <el-input
                          v-model="pluginKeyword"
                          placeholder="搜索插件..."
                          clearable
                          :prefix-icon="Search"
                          class="plugins-search"
                        />
                      </div>
                      <div class="plugins-stats">
                        <span>共</span>
                        <span class="stats-num">{{ availablePlugins.length }}</span>
                        <span>个，匹配</span>
                        <span class="stats-num highlight">{{ filteredPlugins.length }}</span>
                      </div>
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
                  </div>

                  <!-- 插件内容区域 -->
                  <div class="plugins-main card-base">
                    <div v-if="selectedPlugin" class="plugin-renderer-container">
                      <PluginRenderer
                        :plugin-name="selectedPlugin"
                        :plugin-manager="pluginManager"
                        ref="pluginRendererRef"
                      />
                    </div>
                    <div v-else class="no-plugin-selected">
                      <div class="empty-state">
                        <div class="empty-icon">🔌</div>
                        <h3>请选择一个插件</h3>
                        <p>从左侧插件列表中选择要使用的插件</p>
                      </div>
                    </div>
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
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
</style>