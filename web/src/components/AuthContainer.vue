<template>
  <div class="auth-container">
    <!-- 背景装饰元素 -->
    <div class="auth-bg-pattern"></div>
    
    <div class="auth-card">
      <!-- 品牌展示区域 - 使用Element Plus卡片 -->
      <el-card class="brand-card" shadow="never" :body-style="{ padding: 0 }">
        <div class="brand">
          <div class="logo-container">
            <el-avatar shape="circle" size="large" class="brand-logo">
              <img src="/logo.png" alt="Weave Logo" />
            </el-avatar>
            <h1 class="brand-title">Weave</h1>
          </div>
        </div>
      </el-card>
      
      <!-- 标签切换区域 - 使用Element Plus按钮和Tabs组件思路 -->
      <div class="tabs">
        <el-button 
          type="primary" 
          :plain="!showLogin" 
          :icon="UserFilled" 
          @click="switchToLogin"
          class="tab-btn"
        >
          登录
        </el-button>
        <el-button 
          type="primary" 
          :plain="showLogin" 
          :icon="Plus" 
          @click="switchToRegister"
          class="tab-btn"
        >
          注册
        </el-button>
        <el-tabs v-model="activeTab" class="tab-indicator-container" :show-tabbar="false">
          <el-tab-pane label="登录" name="login"></el-tab-pane>
          <el-tab-pane label="注册" name="register"></el-tab-pane>
        </el-tabs>
      </div>
      
      <!-- 表单区域 -->
      <div class="form-area">
        <transition name="form-switch" mode="out-in" appear>
          <Login
            v-if="showLogin"
            @switch-to-register="switchToRegister"
            @login-success="handleLoginSuccess"
          />
          <Register
            v-else
            @switch-to-login="switchToLogin"
            @register-success="handleRegisterSuccess"
          />
        </transition>
      </div>
      
      <!-- 页脚信息 -->
      <div class="auth-footer">
        <p>© {{ new Date().getFullYear() }} Weave - 插件开发/服务聚合平台</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import Login from './Login.vue'
import Register from './Register.vue'
import { authService } from '../services/auth.js'
import api from '../services/auth.js'
// 导入Element Plus图标
import { UserFilled, Plus } from '@element-plus/icons-vue'

// 定义props和emits
const emit = defineEmits(['auth-success'])

// 状态管理
const showLogin = ref(true) // 默认显示登录表单
// 计算属性用于标签切换
const activeTab = computed({
  get() {
    return showLogin.value ? 'login' : 'register'
  },
  set(val) {
    showLogin.value = val === 'login'
  }
})

// 初始化时检查用户是否已登录，并预热CSRF Cookie
onMounted(() => {
  // 预热CSRF：访问健康检查接口以便后端设置XSRF-TOKEN Cookie
  api.get('/health', { withCredentials: true }).catch(() => {})

  if (authService.isAuthenticated()) {
    // 用户已登录，通知父组件
    emit('auth-success')
  }
})

// 切换到注册表单
const switchToRegister = () => {
  showLogin.value = false
}

// 切换到登录表单
const switchToLogin = () => {
  showLogin.value = true
}

// 处理登录成功
const handleLoginSuccess = () => {
  // 登录成功后，通知父组件
  emit('auth-success')
}

// 处理注册成功
const handleRegisterSuccess = () => {
  // 注册成功后，自动切换到登录表单
  showLogin.value = true
}
</script>

<style scoped>
/* 背景容器 */
.auth-container {
  width: 100%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--primary-500) 0%, var(--primary-700) 100%);
  padding: 24px;
  position: relative;
  overflow: hidden;
}

/* 背景装饰图案 */
.auth-bg-pattern {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-image: 
    radial-gradient(circle at 20% 30%, rgba(255, 255, 255, 0.1) 0%, transparent 25%),
    radial-gradient(circle at 80% 70%, rgba(255, 255, 255, 0.1) 0%, transparent 30%);
  z-index: 1;
}

/* 主卡片 */
.auth-card {
  width: 100%;
  max-width: 480px;
  background: var(--bg-primary);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  overflow: hidden;
  position: relative;
  z-index: 2;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.auth-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

/* 品牌卡片 */
.brand-card {
  border: none;
  margin-bottom: 16px;
}

/* 品牌区域 */
.brand {
  background: linear-gradient(135deg, var(--primary-600) 0%, var(--primary-700) 100%);
  color: var(--bg-primary);
  padding: 32px 32px 24px;
  text-align: center;
  border-radius: var(--radius-xl);
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.2);
}

.logo-container {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 16px;
}

.brand-logo {
  border: 3px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.brand-title {
  font-size: var(--font-size-2xl);
  font-weight: var(--font-weight-bold);
  margin: 0;
  background: linear-gradient(135deg, var(--bg-primary) 0%, var(--primary-100) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: -0.02em;
}

.brand-subtitle {
  font-size: var(--font-size-sm);
  margin: 0;
  font-weight: var(--font-weight-medium);
  border-radius: var(--radius-full);
  padding: 6px 20px;
  font-size: 14px;
}

/* 标签切换区域 */
.tabs {
  position: relative;
  display: flex;
  gap: 12px;
  padding: 20px 24px 16px;
  background: var(--bg-primary);
  justify-content: center;
}

.tab-btn {
  flex: 1;
  max-width: 180px;
  border-radius: var(--radius-lg);
  font-weight: var(--font-weight-semibold);
  padding: 12px 20px;
  transition: all 0.3s ease;
}

.tab-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(99, 102, 241, 0.3);
}

/* 标签指示器 */
.tab-indicator-container {
  position: absolute;
  bottom: 0;
  left: 24px;
  right: 24px;
  width: calc(100% - 48px);
}

:deep(.el-tabs__nav-wrap) {
  display: none;
}

/* 表单区域 */
.form-area {
  padding: 24px 24px;
  min-height: 360px;
}

/* 表单切换过渡动画 */
.form-switch-enter-active,
.form-switch-leave-active {
  transition: opacity var(--transition-normal), transform var(--transition-normal);
}

.form-switch-enter-from {
  opacity: 0;
  transform: translateX(10px) scale(0.98);
}

.form-switch-enter-to {
  opacity: 1;
  transform: translateX(0) scale(1);
}

.form-switch-leave-from {
  opacity: 1;
  transform: translateX(0) scale(1);
}

.form-switch-leave-to {
  opacity: 0;
  transform: translateX(-10px) scale(0.98);
}

/* 表单统一风格 */
:deep(form) {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

:deep(.form-group) {
  display: flex;
  flex-direction: column;
  gap: 6px;
  position: relative;
}

:deep(.form-group label) {
  font-weight: var(--font-weight-medium);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  letter-spacing: 0.02em;
}

:deep(.form-group input) {
  width: 100%;
  padding: 12px 16px;
  border: 2px solid var(--border-light);
  border-radius: var(--radius-lg);
  font-size: var(--font-size-base);
  transition: all var(--transition-normal);
  background: var(--bg-primary);
  color: var(--text-primary);
}

:deep(.form-group input:focus) {
  outline: none;
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
  background: var(--bg-primary);
}

:deep(.form-group input:placeholder-shown) {
  color: var(--text-muted);
}

:deep(.form-group input::placeholder) {
  color: var(--text-muted);
  opacity: 1;
}

/* 提交按钮 */
:deep(button[type="submit"]) {
  padding: 12px 16px;
  border: none;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--primary-600) 0%, var(--primary-700) 100%);
  color: var(--bg-primary);
  cursor: pointer;
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-semibold);
  transition: all var(--transition-normal);
  position: relative;
  overflow: hidden;
}

:deep(button[type="submit"])::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 0;
  height: 0;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.2);
  transform: translate(-50%, -50%);
  transition: width 0.6s ease, height 0.6s ease;
}

:deep(button[type="submit"]:hover::before) {
  width: 300px;
  height: 300px;
}

:deep(button[type="submit"]:hover) {
  transform: translateY(-1px);
  box-shadow: 0 8px 20px rgba(99, 102, 241, 0.4);
}

:deep(button[type="submit"]:active) {
  transform: translateY(0);
}

:deep(button[type="submit"]:disabled) {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

/* 错误消息 */
:deep(.error-message) {
  background: var(--error-100);
  color: var(--error-700);
  border: 1px solid var(--error);
  padding: 10px 12px;
  border-radius: var(--radius-lg);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  display: flex;
  align-items: center;
  gap: 8px;
  position: relative;
}

:deep(.error-message)::before {
  content: '';
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 20 20' fill='%23dc2626'%3E%3Cpath fill-rule='evenodd' d='M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z' clip-rule='evenodd'/%3E%3C/svg%3E") no-repeat center center;
  background-size: contain;
}

/* 切换提示 */
:deep(.switch-tip) {
  text-align: center;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  margin-top: 16px;
}

:deep(.link-btn) {
  font-size: var(--font-size-sm);
  background: none;
  border: none;
  color: var(--primary-600);
  cursor: pointer;
  font-weight: var(--font-weight-medium);
  padding: 2px 6px;
  border-radius: var(--radius);
  transition: all var(--transition-fast);
  text-decoration: none;
}

:deep(.link-btn:hover) {
  background: var(--primary-50);
  color: var(--primary-700);
}

/* 页脚 */
.auth-footer {
  padding: 20px 24px;
  background: linear-gradient(135deg, var(--primary-500) 0%, var(--primary-700) 100%);
  border-top: 3px solid var(--primary-400);
  text-align: center;
  position: relative;
  overflow: hidden;
}

/* 页脚装饰元素 */
.auth-footer::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: radial-gradient(circle at 20% 80%, rgba(255, 255, 255, 0.15) 0%, transparent 20%),
              radial-gradient(circle at 80% 20%, rgba(255, 255, 255, 0.1) 0%, transparent 15%);
  z-index: 1;
}

.auth-footer p {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--bg-primary);
  font-weight: var(--font-weight-medium);
  letter-spacing: 0.02em;
  position: relative;
  z-index: 2;
  display: inline-block;
  padding: 4px 8px;
  border-radius: var(--radius);
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(4px);
  transition: all var(--transition-normal);
}

.auth-footer p:hover {
  background: rgba(255, 255, 255, 0.2);
  transform: translateY(-1px);
}

/* 响应式设计 */
@media (max-width: 480px) {
  .auth-container {
    padding: 16px;
  }
  
  .brand {
    padding: 24px 20px 20px;
  }
  
  .form-area {
    padding: 20px 20px;
  }
  
  .auth-card {
    max-width: 100%;
    border-radius: var(--radius-lg);
  }
}
</style>