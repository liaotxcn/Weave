<template>
  <div class="register-container">
    <h2 class="form-title">注册</h2>
    <el-form class="auth-form" @submit.prevent="handleRegister" label-position="top" size="large">
      <!-- 用户名输入 -->
      <el-form-item label="用户名" :error="usernameInvalid && username ? '用户名至少需要3个字符' : ''" :validate-status="usernameInvalid && username ? 'error' : (username && !usernameInvalid ? 'success' : '')" required>
        <el-input 
          v-model="username" 
          placeholder="3-50个字符" 
          maxlength="50" 
          clearable
          @input="clearError"
          prefix-icon="User"
        />
        <div v-if="username && !usernameInvalid" class="input-hint input-hint-success">用户名可用</div>
      </el-form-item>
      
      <!-- 邮箱输入 -->
      <el-form-item label="邮箱" :error="emailInvalid && email ? '请输入有效的邮箱地址' : ''" :validate-status="emailInvalid && email ? 'error' : (email && !emailInvalid ? 'success' : '')" required>
        <el-input 
          v-model="email" 
          type="email" 
          placeholder="name@example.com" 
          clearable
          @input="clearError"
          prefix-icon="Message"
        />
        <div v-if="email && !emailInvalid" class="input-hint input-hint-success">邮箱格式正确</div>
      </el-form-item>
      
      <!-- 密码输入 -->
      <el-form-item label="密码" :error="password.length > 0 && password.length < 6 ? '密码至少需要6个字符' : ''" :validate-status="password.length > 0 && password.length < 6 ? 'error' : ''" required>
        <el-input 
          v-model="password" 
          type="password" 
          placeholder="至少6个字符" 
          show-password 
          clearable
          @input="clearError"
          prefix-icon="Lock"
        />
        
        <!-- 密码强度指示器 -->
        <div v-if="password" class="pw-strength">
          <div class="strength-label">密码强度：</div>
          <div class="strength-bar-container">
            <div class="strength-bar">
              <div 
                class="strength-progress" 
                :class="passwordLevel"
                :style="{ width: getStrengthWidth() }"
              ></div>
            </div>
            <span class="strength-text" :class="passwordLevel">{{ passwordLabel }}</span>
          </div>
          
          <!-- 密码强度提示 -->
          <div class="strength-hints">
            <div class="hint-item" :class="{ 'hint-passed': password.length >= 6 }">
              <span class="hint-icon">{{ password.length >= 6 ? '✓' : '•' }}</span>
              <span>至少6个字符</span>
            </div>
            <div class="hint-item" :class="{ 'hint-passed': /[A-Z]/.test(password) }">
              <span class="hint-icon">{{ /[A-Z]/.test(password) ? '✓' : '•' }}</span>
              <span>包含大写字母</span>
            </div>
            <div class="hint-item" :class="{ 'hint-passed': /[a-z]/.test(password) }">
              <span class="hint-icon">{{ /[a-z]/.test(password) ? '✓' : '•' }}</span>
              <span>包含小写字母</span>
            </div>
            <div class="hint-item" :class="{ 'hint-passed': /\d/.test(password) }">
              <span class="hint-icon">{{ /\d/.test(password) ? '✓' : '•' }}</span>
              <span>包含数字</span>
            </div>
            <div class="hint-item" :class="{ 'hint-passed': /[^\w]/.test(password) }">
              <span class="hint-icon">{{ /[^\w]/.test(password) ? '✓' : '•' }}</span>
              <span>包含特殊字符</span>
            </div>
          </div>
        </div>
      </el-form-item>
      
      <!-- 确认密码 -->
      <el-form-item label="确认密码" :error="passwordMismatch && confirmPassword ? '两次输入的密码不一致' : ''" :validate-status="passwordMismatch && confirmPassword ? 'error' : (confirmPassword && !passwordMismatch && password ? 'success' : '')" required>
        <el-input 
          v-model="confirmPassword" 
          type="password" 
          placeholder="请再次输入密码" 
          show-password 
          clearable
          @input="clearError"
          prefix-icon="Lock"
        />
        <div v-if="confirmPassword && !passwordMismatch && password" class="input-hint input-hint-success">密码一致</div>
      </el-form-item>
      
      <!-- 错误消息 -->
      <el-form-item>
        <el-alert v-if="errorMessage" type="error" :message="errorMessage" show-icon center :closable="false" />
      </el-form-item>
      
      <!-- 注册按钮 -->
      <el-form-item>
        <el-button type="primary" native-type="submit" :disabled="loading || !canRegister" :loading="loading" style="width: 100%;" size="large">
        注册
      </el-button>
      </el-form-item>
    </el-form>
    
    <p class="switch-tip">
      已有账号？
      <el-button type="text" @click="switchToLogin">返回登录</el-button>
    </p>
  </div>
</template>

<script>
import { ref, computed } from 'vue'
import { authService } from '../services/auth'

export default {
  name: 'Register',
  emits: ['register-success', 'switch-to-login'],
  setup(props, { emit }) {
    // 响应式数据
    const username = ref('')
    const email = ref('')
    const password = ref('')
    const confirmPassword = ref('')
    const showPassword = ref(false)
    const showConfirmPassword = ref(false)
    const loading = ref(false)
    const errorMessage = ref('')

    // 计算属性
    const usernameInvalid = computed(() => {
      return !(username.value && username.value.trim().length >= 3)
    })
    
    const emailInvalid = computed(() => {
      if (!email.value) return false
      const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
      return !re.test(email.value)
    })
    
    const passwordLevel = computed(() => {
      const n = password.value || ''
      let score = 0
      if (n.length >= 6) score++
      if (/[A-Z]/.test(n)) score++
      if (/[a-z]/.test(n)) score++
      if (/\d/.test(n)) score++
      if (/[^\w]/.test(n)) score++
      if (score <= 2) return 'weak'
      if (score === 3 || score === 4) return 'medium'
      return 'strong'
    })
    
    const passwordLabel = computed(() => {
      return passwordLevel.value === 'strong' ? '强' : (passwordLevel.value === 'medium' ? '中' : '弱')
    })
    
    const passwordMismatch = computed(() => {
      return !!confirmPassword.value && password.value !== confirmPassword.value
    })
    
    const canRegister = computed(() => {
      return !usernameInvalid.value && !emailInvalid.value && !!password.value && 
             password.value.length >= 6 && !passwordMismatch.value
    })

    // 方法
    const clearError = () => {
      errorMessage.value = ''
    }
    
    const getStrengthWidth = () => {
      const n = password.value || ''
      let score = 0
      if (n.length >= 6) score++
      if (/[A-Z]/.test(n)) score++
      if (/[a-z]/.test(n)) score++
      if (/\d/.test(n)) score++
      if (/[^\w]/.test(n)) score++
      
      // 计算宽度百分比
      return `${(score / 5) * 100}%`
    }
    
    const handleRegister = async () => {
      if (!canRegister.value) return
      errorMessage.value = ''
      
      try {
        loading.value = true
        const payload = {
          username: username.value.trim(),
          email: email.value.trim(),
          password: password.value,
          confirm_password: confirmPassword.value
        }

        const response = await authService.register(payload)

        if (response && response.user) {
          emit('register-success', response.user)
        } else {
          errorMessage.value = response?.message || '注册失败，请稍后重试'
        }
      } catch (error) {
        const data = error?.response?.data || {}
        errorMessage.value = data?.message || '注册失败，请检查输入或网络'
      } finally {
        loading.value = false
      }
    }
    
    const switchToLogin = () => {
      emit('switch-to-login')
    }

    return {
      username,
      email,
      password,
      confirmPassword,
      showPassword,
      showConfirmPassword,
      loading,
      errorMessage,
      usernameInvalid,
      emailInvalid,
      passwordLevel,
      passwordLabel,
      passwordMismatch,
      canRegister,
      clearError,
      getStrengthWidth,
      handleRegister,
      switchToLogin
    }
  }
}
</script>

<style scoped>
.register-container {
  display: flex;
  flex-direction: column;
  gap: 0;
  max-width: 400px;
  width: 100%;
  margin: 0 auto;
  padding: 24px;
  background: white;
  border-radius: var(--radius-xl);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.04), 0 2px 8px rgba(0, 0, 0, 0.02);
}

.form-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 28px 0;
  text-align: center;
  letter-spacing: -0.02em;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* 验证码输入框 */
.verification-code-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.input-hint {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-top: 6px;
  padding-left: 2px;
}

.input-hint-success {
  color: var(--color-success);
  font-size: var(--font-size-xs);
  margin-top: 6px;
  padding-left: 2px;
}

/* 彻底修复Element Plus输入框双重边框问题 */
:deep(.el-input) {
  /* 确保输入框容器没有额外边框 */
  border: none !important;
  box-shadow: none !important;
}

:deep(.el-input__wrapper) {
  /* 重置输入框包装器的所有边框和阴影 */
  transition: all 0.3s ease;
  border-radius: var(--radius-md) !important;
  box-shadow: none !important;
  border: 1px solid var(--border-color) !important;
  outline: none !important;
  background-color: #fff !important;
}

:deep(.el-input__wrapper:focus-within) {
  /* 焦点状态只保留一层阴影和边框 */
  box-shadow: 0 0 0 2px rgba(144, 202, 249, 0.2), 0 2px 8px rgba(144, 202, 249, 0.3) !important;
  border-color: #69b1ff !important;
}

:deep(.el-input__inner) {
  /* 确保内部输入元素没有额外边框 */
  border: none !important;
  box-shadow: none !important;
  background-color: transparent !important;
}

/* 禁用状态优化 */
:deep(.el-input.is-disabled .el-input__wrapper) {
  opacity: 0.7;
  background-color: var(--bg-secondary) !important;
}

/* 输入提示 */
.input-hint {
  font-size: var(--font-size-xs);
  margin-top: 2px;
  transition: all var(--transition-fast);
}

.input-hint-error {
  color: var(--error);
  font-weight: var(--font-weight-medium);
}

.input-hint-success {
  color: var(--success);
  font-weight: var(--font-weight-medium);
}

/* 密码强度指示器 */
.pw-strength {
  margin-top: 8px;
}

.strength-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  margin-bottom: 4px;
  font-weight: var(--font-weight-medium);
}

.strength-bar-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.strength-bar {
  flex: 1;
  height: 8px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-full);
  overflow: hidden;
  position: relative;
}

.strength-progress {
  height: 100%;
  transition: all var(--transition-normal);
  border-radius: var(--radius-full);
  position: relative;
}

.strength-progress::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(90deg, transparent 0%, rgba(255, 255, 255, 0.2) 50%, transparent 100%);
  animation: shimmer 2s infinite;
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}

.strength-progress.weak {
  background: var(--error);
}

.strength-progress.medium {
  background: var(--warning);
}

.strength-progress.strong {
  background: var(--success);
}

.strength-text {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  min-width: 24px;
}

.strength-text.weak {
  color: var(--error);
}

.strength-text.medium {
  color: var(--warning);
}

.strength-text.strong {
  color: var(--success);
}

/* 密码强度提示 */
.strength-hints {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 6px;
  font-size: var(--font-size-xs);
}

@media (max-width: 480px) {
  .strength-hints {
    grid-template-columns: 1fr;
  }
}

.hint-item {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-tertiary);
}

.hint-passed {
  color: var(--success);
  font-weight: var(--font-weight-medium);
}

.hint-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  font-size: 10px;
}

.hint-passed .hint-icon {
  background: var(--success-100);
  color: var(--success);
}

/* 切换提示 */
.switch-tip {
  text-align: center;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  margin-top: 24px;
}

/* 响应式调整 */
@media (max-width: 480px) {
  .form-title {
    font-size: var(--font-size-lg);
    margin-bottom: 20px;
  }
  
  .auth-form {
    gap: 14px;
  }
  
  .switch-tip {
    margin-top: 20px;
  }
}
</style>