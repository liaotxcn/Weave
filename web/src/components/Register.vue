<template>
  <div class="register-container">
    <h2 class="form-title">注册</h2>
    <el-form class="auth-form el-input-reset" @submit.prevent="handleRegister" label-position="top" size="large">
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
        <el-button type="primary" native-type="submit" :disabled="loading || !canRegister" :loading="loading" size="default" class="submit-btn">
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
  max-width: 420px;
  width: 100%;
  margin: 0 auto;
  padding: 12px 16px;
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-card);
}

.submit-btn {
  width: 100%;
  height: 38px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.02em;
  border-radius: 10px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.submit-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(99, 102, 241, 0.35);
}

.switch-btn {
  font-weight: 500;
  color: var(--color-primary);
  padding: 4px 8px;
}

.switch-btn:hover {
  color: var(--primary-700);
}
</style>