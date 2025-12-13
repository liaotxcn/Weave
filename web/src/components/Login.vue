<template>
  <div class="login-container">
    <h2 class="form-title">登录</h2>
    
    <el-form class="auth-form" @submit.prevent="handleLogin" label-position="top" size="large">
      <!-- 用户名输入 -->
      <el-form-item label="用户名" required>
        <el-input 
          v-model="username" 
          placeholder="请输入用户名" 
          autofocus 
          clearable 
          @input="clearError"
          prefix-icon="User"
        />
      </el-form-item>
      
      <!-- 密码输入 -->
      <el-form-item label="密码" required>
        <el-input 
          v-model="password" 
          type="password" 
          placeholder="请输入密码" 
          show-password 
          clearable 
          @input="clearError"
          prefix-icon="Lock"
        />
      </el-form-item>
      
      <!-- 验证码输入 -->
      <el-form-item label="邮箱验证码" required>
        <div class="verification-code-wrap">
          <el-input 
            v-model="verificationCode" 
            placeholder="请输入验证码" 
            maxlength="6" 
            clearable 
            @input="clearError" 
            style="flex: 1;"
            prefix-icon="Message"
          />
          <el-button 
            type="primary" 
            :disabled="loading || !canSendCode || countdown > 0"
            @click="sendVerificationCode"
            title="验证码将发送至您注册时使用的邮箱"
            :plain="true"
            size="large"
            style="min-width: 120px;"
          >
            {{ countdown > 0 ? `${countdown}秒后重新获取` : '获取验证码' }}
          </el-button>
        </div>
        <div class="input-hint">
          验证码将发送至您注册时使用的邮箱，请确保该邮箱可访问
        </div>
      </el-form-item>
      
      <div class="assist">
        <el-checkbox v-model="rememberMe">记住我</el-checkbox>
      </div>
      
      <!-- 验证码发送成功提示将使用ElMessage组件显示 -->
      
      <el-button type="primary" native-type="submit" :disabled="loading || !canLogin" :loading="loading" style="width: 100%;" size="large">
        登录
      </el-button>
    </el-form>
    
    <p class="switch-tip">
      还没有账号？
      <el-button type="text" @click="switchToRegister">立即注册</el-button>
    </p>
  </div>
</template>

<script>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { authService } from '../services/auth'

export default {
  name: 'Login',
  emits: ['login-success', 'switch-to-register'],
  setup(props, { emit }) {
    // 响应式数据
    const username = ref('')
    const password = ref('')

    const verificationCode = ref('')
    const rememberMe = ref(true)
    const showPassword = ref(false)
    const loading = ref(false)
    const errorMessage = ref('')
    const successMessage = ref('')
    const countdown = ref(0)
    let countdownTimer = null

    // 计算属性
    // 计算属性 - 表单验证
    const usernameInvalid = computed(() => {
      return !(username.value && username.value.trim().length > 0)
    })
    
    const passwordInvalid = computed(() => {
      return !(password.value && password.value.length > 0)
    })
    
    const verificationCodeInvalid = computed(() => {
      return !(verificationCode.value && verificationCode.value.trim().length > 0)
    })
    
    const canSendCode = computed(() => {
      return username.value && username.value.trim().length > 0
    })
    
    const canLogin = computed(() => {
      return !usernameInvalid.value && !passwordInvalid.value && !verificationCodeInvalid.value
    })

    // 方法
    const clearError = () => {
      errorMessage.value = ''
      successMessage.value = ''
      // 不清除verificationCodeSent，保留验证码发送成功提示
    }
    
    const startCountdown = () => {
      countdown.value = 60
      countdownTimer = setInterval(() => {
        countdown.value--
        if (countdown.value <= 0) {
          clearInterval(countdownTimer)
          countdown.value = 0
        }
      }, 1000)
    }
    
    // 新增一个专门用于标记验证码发送成功的变量
    const verificationCodeSent = ref(false)
    
    const sendVerificationCode = async () => {
      if (!canSendCode.value || countdown.value > 0) return
      clearError()
      
      try {
        const response = await authService.sendVerificationCode({ username: username.value.trim() })
        // 使用ElMessage组件显示验证码发送成功提示
        ElMessage.success('验证码已发送至您的邮箱，请注意查收')
        startCountdown()
      } catch (error) {
        const data = error?.response?.data || {}
        // 使用ElMessage组件显示验证码发送失败提示
        ElMessage.error(data?.message || '发送验证码失败，请检查网络')
      }
    }
    
    const handleLogin = async () => {
      if (!canLogin.value) return
      errorMessage.value = ''
      verificationCodeSent.value = false
      
      try {
        loading.value = true
        
        // 发送包含用户名、密码和邮箱验证码的登录请求
        const payload = {
          username: username.value.trim(),
          password: password.value,
          code: verificationCode.value,
          remember_me: rememberMe.value
        }
        
        const response = await authService.login(payload)

        if (response && response.user) {
          emit('login-success', response.user)
        } else {
          // 处理非2xx但有响应体的情况
          ElMessage.error(response?.message || '登录失败，请稍后重试')
        }
      } catch (error) {
        console.log('登录错误详情:', error)
        // 全面的错误信息处理，确保能获取到具体错误内容
        let errorMsg = '登录失败，请检查账号或网络'
        
        if (error?.response) {
          // 服务器返回了错误响应
          const data = error.response.data
          if (data?.message) {
            errorMsg = data.message
          } else if (typeof data === 'string') {
            errorMsg = data
          } else if (error.response.statusText) {
            errorMsg = error.response.statusText
          }
        } else if (error?.message) {
          // 网络错误或其他错误
          errorMsg = error.message
        }
        
        ElMessage.error(errorMsg)
      } finally {
        loading.value = false
      }
    }
    
    const switchToRegister = () => {
      emit('switch-to-register')
    }

    return {
      username,
      password,
      verificationCode,
      rememberMe,
      showPassword,
      loading,
      errorMessage,
      successMessage,
      countdown,
      verificationCodeSent,
      usernameInvalid,
      passwordInvalid,
      verificationCodeInvalid,
      canSendCode,
      canLogin,
      clearError,
      handleLogin,
      sendVerificationCode,
      switchToRegister
    }
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  flex-direction: column;
  gap: 0;
  position: relative;
  max-width: 400px;
  width: 100%;
  margin: 0 auto;
  padding: 24px;
  background: white;
  border-radius: var(--radius-xl);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.04), 0 2px 8px rgba(0, 0, 0, 0.02);
}

/* 验证码输入框 */
.verification-code-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
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

.input-hint {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-top: 4px;
  padding-left: 4px;
}

/* 辅助选项 */
.assist {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
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
  
  .assist {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .switch-tip {
    margin-top: 20px;
  }
}
</style>