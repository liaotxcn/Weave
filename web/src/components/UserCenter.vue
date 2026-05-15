<template>
  <div class="user-center">
    <div class="welcome-header">
      <div class="welcome-content">
        <el-avatar :size="48" :icon="UserFilled" class="user-avatar-large"></el-avatar>
        <div class="welcome-info">
          <h2>个人中心</h2>
          <p>管理您的账户信息、安全设置和活动记录</p>
        </div>
      </div>
    </div>

    <div class="cards">
      <div class="profile-card card-base">
        <div class="card-header">
          <div class="card-icon">
            <el-icon><User /></el-icon>
          </div>
          <h3>基本资料</h3>
        </div>
        
        <div class="form-content el-input-reset">
          <div class="form-group">
            <label class="form-label">用户名</label>
            <el-input 
              :value="user?.username" 
              readonly 
              placeholder="用户名不可修改"
              :prefix-icon="User"
            />
          </div>

          <div class="form-group">
            <label class="form-label">邮箱地址</label>
            <el-input 
              v-model="email" 
              type="email" 
              placeholder="name@example.com" 
              :prefix-icon="Message"
              @input="clearStatus" 
            />
            <div v-if="emailInvalid" class="field-error">请输入有效的邮箱地址</div>
          </div>

          <div class="form-group">
            <label class="form-label">创建时间</label>
            <el-input 
              :value="formatDate(user?.created_at)" 
              readonly 
              :prefix-icon="Calendar"
            />
          </div>
        </div>

        <div class="card-actions">
          <el-button 
            type="primary" 
            @click="updateProfile" 
            :disabled="updating || !canSave"
            :loading="updating"
            :icon="Check"
          >
            保存资料
          </el-button>
        </div>
      </div>

      <div class="security-card card-base">
        <div class="card-header">
          <div class="card-icon">
            <el-icon><Lock /></el-icon>
          </div>
          <h3>安全设置</h3>
        </div>

        <div class="form-content el-input-reset">
          <div class="form-group">
            <label class="form-label">当前密码</label>
            <el-input 
              v-model="currentPassword" 
              type="password" 
              placeholder="请输入当前密码" 
              :prefix-icon="Lock" 
              @input="clearStatus"
            />
            <div v-if="currentPasswordError" class="field-error">{{ currentPasswordError }}</div>
          </div>
          
          <div class="form-group">
            <label class="form-label">新密码</label>
            <el-input 
              v-model="newPassword" 
              type="password" 
              placeholder="至少6个字符" 
              :prefix-icon="Key" 
              @input="clearStatus"
            />
            
            <div v-if="newPassword" class="password-strength">
              <div class="strength-bar">
                <div class="strength-fill" :class="passwordLevel"></div>
              </div>
              <div class="strength-info">
                <span class="strength-label" :class="passwordLevel">{{ passwordLabel }}</span>
                <span class="strength-tips">
                  <span v-if="newPassword.length < 6">• 至少6个字符</span>
                  <span v-if="!/[A-Z]/.test(newPassword)">• 包含大写字母</span>
                  <span v-if="!/[a-z]/.test(newPassword)">• 包含小写字母</span>
                  <span v-if="!/\d/.test(newPassword)">• 包含数字</span>
                  <span v-if="!/[^\w]/.test(newPassword)">• 包含特殊字符</span>
                </span>
              </div>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">确认密码</label>
            <el-input 
              v-model="confirmNewPassword" 
              type="password" 
              placeholder="再次输入新密码" 
              :prefix-icon="Key" 
              @input="clearStatus"
            />
            <div v-if="passwordMismatch" class="field-error">两次输入的密码不一致</div>
          </div>
        </div>

        <div class="card-actions">
          <el-button 
            type="primary" 
            @click="updatePassword" 
            :disabled="updating || !canUpdatePassword"
            :loading="updating"
            :icon="DocumentCopy"
          >
            更新密码
          </el-button>
        </div>
      </div>
    </div>

    <el-card class="activity-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <div class="card-icon">
            <el-icon><List /></el-icon>
          </div>
          <h3>近期活动</h3>
          <span class="card-subtitle">审计日志记录</span>
        </div>
      </template>
      
      <div class="activity-content">
        <el-empty v-if="auditLogs.length === 0" description="暂无活动记录">
          <template #icon>
            <el-icon :size="48"><List /></el-icon>
          </template>
        </el-empty>
        
        <div v-else class="activity-list">
          <el-timeline>
            <el-timeline-item 
              v-for="log in currentPageLogs" 
              :key="log.id" 
              :timestamp="formatDate(log.created_at)"
              placement="top"
            >
              <div class="activity-item">
                <div class="activity-action">
                  <el-tag :type="getActionType(log.action)">
                    {{ log.action }}
                  </el-tag>
                  <span class="resource-info">{{ log.resource_type }} #{{ log.resource_id }}</span>
                </div>
                <div v-if="log.new_value || log.old_value" class="activity-details">
                  <div v-if="log.old_value" class="change-item old">
                    <el-icon><CircleCloseFilled /></el-icon>
                    <span>变更前: {{ short(log.old_value) }}</span>
                  </div>
                  <div v-if="log.new_value" class="change-item new">
                    <el-icon><CircleCheckFilled /></el-icon>
                    <span>变更后: {{ short(log.new_value) }}</span>
                  </div>
                </div>
              </div>
            </el-timeline-item>
          </el-timeline>
        </div>
        
        <!-- 分页控制 -->
        <div v-if="auditLogs.length > 0" class="pagination">
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[8, 16, 24]"
            :total="auditLogs.length"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
          />
        </div>
      </div>
    </el-card>

    <!-- 消息提示 -->
    <el-message :message="errorMessage" type="error" :show-close="true" v-if="errorMessage" duration="3000" />
    <el-message :message="successMessage" type="success" :show-close="true" v-if="successMessage" duration="3000" />
  </div>
</template>

<script>
import api, { authService } from '../services/auth'
import { 
  UserFilled, 
  User, 
  Message, 
  Calendar, 
  Lock, 
  Key, 
  Check, 
  List, 
  DocumentCopy,
  CircleCloseFilled,
  CircleCheckFilled
} from '@element-plus/icons-vue'

export default {
  name: 'UserCenter',
  components: {
    UserFilled,
    User,
    Message,
    Calendar,
    Lock,
    Key,
    Check,
    List,
    DocumentCopy,
    CircleCloseFilled,
    CircleCheckFilled
  },
  props: {
    currentUser: { type: Object, default: null }
  },
  data() {
    return {
      user: null,
      email: '',
      currentPassword: '',
      newPassword: '',
      confirmNewPassword: '',
      auditLogs: [],
      updating: false,
      errorMessage: '',
      successMessage: '',
      currentPasswordError: '',
      // 分页相关状态
      currentPage: 1,
      pageSize: 8
    }
  },
  mounted() {
    this.loadUser()
  },
  computed: {
    // 是否允许保存资料
    canSave() {
      return !!this.email && !this.emailInvalid
    },
    // 当前页显示的数据
    currentPageLogs() {
      const startIndex = (this.currentPage - 1) * this.pageSize
      const endIndex = startIndex + this.pageSize
      return this.auditLogs.slice(startIndex, endIndex)
    },
    // 新增：邮箱是否非法
    emailInvalid() {
      if (!this.email) return false
      const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
      return !re.test(this.email)
    },
    // 新增：密码强度等级样式
    passwordLevel() {
      const n = this.newPassword || ''
      let score = 0
      if (n.length >= 6) score++
      if (/[A-Z]/.test(n)) score++
      if (/[a-z]/.test(n)) score++
      if (/\d/.test(n)) score++
      if (/[^\w]/.test(n)) score++
      if (score <= 2) return 'weak'
      if (score === 3 || score === 4) return 'medium'
      return 'strong'
    },
    // 新增：密码强度文案
    passwordLabel() {
      return this.passwordLevel === 'strong' ? '强' : (this.passwordLevel === 'medium' ? '中' : '弱')
    },
    // 新增：两次密码是否不一致
    passwordMismatch() {
      return !!this.confirmNewPassword && this.newPassword !== this.confirmNewPassword
    },
    // 新增：是否允许更新密码
    canUpdatePassword() {
      return !!this.currentPassword && !!this.newPassword && this.newPassword.length >= 6 && !this.passwordMismatch && !this.currentPasswordError
    }
  },
  methods: {
    // 分页相关方法
    handleSizeChange(val) {
      this.pageSize = val
      this.currentPage = 1
    },
    handleCurrentChange(val) {
      this.currentPage = val
    },
    handleSizeChange(newSize) {
      this.pageSize = newSize
      this.currentPage = 1 // Reset to first page when changing page size
    },
    handleCurrentChange(newPage) {
      this.currentPage = newPage
    },
    // 新增：输入时清除顶部成功/错误提示，避免阻碍表单操作
    clearStatus() {
      this.errorMessage = ''
      this.successMessage = ''
      this.currentPasswordError = ''
    },
    // 新增：获取操作类型对应的Element Plus标签类型
    getActionType(action) {
      const actionMap = {
        'UPDATE': 'warning',
        'CREATE': 'success', 
        'DELETE': 'danger'
      }
      return actionMap[action] || 'warning'
    },
    async loadUser() {
      try {
        const cur = authService.getCurrentUser()
        if (!cur || !cur.id) {
          this.errorMessage = '未获取到当前用户信息，请重新登录'
          return
        }
        
        // 当前使用的API调用方式
        const res = await api.get(`/api/v1/users/${cur.id}`)
        this.user = res
        this.email = res.email || ''
        
        // 保存所有的审计日志，用于本地分页
        // 确保auditLogs是数组，并且初始化currentPage为1
        this.auditLogs = Array.isArray(res.audit_logs) ? res.audit_logs : []
        this.currentPage = 1 // 重置为第一页
        this.errorMessage = ''
      } catch (e) {
        this.errorMessage = e?.response?.data?.message || '加载用户信息失败'
      }
    },
    async updateProfile() {
      try {
        this.updating = true
        this.successMessage = ''
        this.errorMessage = ''
        const cur = authService.getCurrentUser()
        if (!cur || !cur.id) throw new Error('未登录')
        const payload = { email: this.email }
        const updated = await api.put(`/api/v1/users/${cur.id}`, payload)
        this.user = updated
        this.$emit('updated-user', updated)
        this.successMessage = '资料已更新'
      } catch (e) {
        this.errorMessage = e?.response?.data?.message || '更新失败'
      } finally {
        this.updating = false
      }
    },
    async updatePassword() {
      try {
        this.updating = true
        this.successMessage = ''
        this.errorMessage = ''
        
        // 表单验证
        if (!this.newPassword || this.newPassword.length < 6) {
          this.errorMessage = '新密码至少6个字符'
          return
        }
        if (this.newPassword !== this.confirmNewPassword) {
          this.errorMessage = '两次输入的密码不一致'
          return
        }
        
        // 确保用户已登录
        const cur = authService.getCurrentUser()
        if (!cur || !cur.id) throw new Error('未登录')
        
        // 验证当前密码是否输入
        if (!this.currentPassword) {
          this.currentPasswordError = '请输入当前密码'
          return
        }
        
        // 使用正确的API端点和参数格式
        const payload = {
          current_password: this.currentPassword,
          new_password: this.newPassword
        }
        
        // 发送修改密码请求（使用POST方法）
        await api.post('/api/v1/users/change-password', payload)
        
        // 成功后重置表单并显示成功消息
        this.currentPassword = ''
        this.newPassword = ''
        this.confirmNewPassword = ''
        this.successMessage = '密码已成功更新'
        
        // 可以考虑让用户重新登录以确保安全性
        // 此处可以添加自动登出逻辑或提示
      } catch (e) {
          // 处理错误响应
          if (e?.response?.status === 403 && e?.response?.data?.error === 'CSRF token validation failed') {
            this.errorMessage = '安全验证失败，请刷新页面后重试'
          } else if (e?.response?.status === 400 && e?.response?.data?.message?.includes('当前密码')) {
            // 当前密码错误的情况
            this.currentPasswordError = e?.response?.data?.message || '当前密码错误'
          } else {
            this.errorMessage = e?.response?.data?.message || '密码更新失败'
          }
        } finally {
          this.updating = false
        }
    },
    formatDate(dt) {
      if (!dt) return '-'
      try { return new Date(dt).toLocaleString() } catch (_) { return dt }
    },
    short(text) {
      if (!text) return ''
      const s = String(text)
      return s.length > 180 ? s.slice(0, 180) + '…' : s
    }
  }
}
</script>

<style scoped>
.user-center {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
  max-width: 1100px;
  margin: 0 auto;
}

/* ===== 欢迎头部 ===== */
.welcome-header {
  background: linear-gradient(135deg, var(--primary-500), var(--primary-700));
  border-radius: var(--radius-xl);
  padding: 24px 28px;
  color: white;
}

.welcome-content {
  display: flex;
  align-items: center;
  gap: 18px;
}

.user-avatar-large {
  transition: transform 0.2s ease;
}

.user-avatar-large:hover {
  transform: scale(1.04);
}

.welcome-info h2 {
  margin: 0 0 6px 0;
  font-size: 22px;
  font-weight: 700;
}

.welcome-info p {
  margin: 0;
  opacity: 0.88;
  font-size: 14px;
}

/* ===== 卡片网格 ===== */
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(380px, 1fr));
  gap: 20px;
}

/* ===== 卡片基础样式 ===== */
.card-base {
  background: var(--color-surface);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  padding: 24px;
  transition: all 0.25s ease;
}

.card-base:hover {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
  border-color: var(--border-medium);
}

/* ===== 卡片头部 ===== */
.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 18px;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--bg-tertiary);
}

.card-icon {
  width: 36px;
  height: 36px;
  background: var(--primary-50);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-600);
  flex-shrink: 0;
  font-size: 17px;
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

/* ===== 表单区域 ===== */
.form-content {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  padding-bottom: 2px;
}

.field-error {
  font-size: 12px;
  color: var(--error);
  margin-top: 4px;
  line-height: 1.4;
}

/* 密码强度 */
.password-strength {
  margin-top: 8px;
  padding: 12px 14px;
  background: var(--bg-secondary);
  border-radius: 10px;
  border: 1px solid var(--border-light);
}

.strength-bar {
  height: 5px;
  background: var(--bg-tertiary);
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 10px;
}

.strength-fill {
  height: 100%;
  border-radius: 3px;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  width: 0%;
}

.strength-fill.weak { width: 33%; background: var(--error); }
.strength-fill.medium { width: 66%; background: var(--warning); }
.strength-fill.strong { width: 100%; background: var(--success); }

.strength-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.strength-label {
  font-size: 13px;
  font-weight: 700;
}
.strength-label.weak { color: var(--error); }
.strength-label.medium { color: var(--warning); }
.strength-label.strong { color: var(--success); }

.strength-tips {
  font-size: 11px;
  color: var(--color-text-tertiary);
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

/* 操作按钮区 */
.card-actions {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--bg-tertiary);
  display: flex;
  justify-content: flex-start;
}

.card-actions .el-button {
  height: 40px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.01em;
  border-radius: 10px;
  padding: 0 24px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.card-actions .el-button--primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(99, 102, 241, 0.3);
}

/* 活动卡片 */
.activity-card {
  grid-column: 1 / -1;
  background: var(--color-surface);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.activity-card :deep(.el-card__header) {
  padding: 20px 24px 0 !important;
  border-bottom: none !important;
}

.activity-card :deep(.el-card__body) {
  padding: 20px 24px 24px !important;
}

.activity-content {
  min-height: 160px;
}

/* 活动列表 */
.activity-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.activity-item {
  padding: 14px 16px;
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 1px solid var(--border-light);
  transition: all 0.25s ease;
}

.activity-item:hover {
  background: var(--color-surface);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
  transform: translateY(-2px);
  border-color: var(--border-medium);
}

.activity-action {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.resource-info {
  color: var(--color-text-secondary);
  font-size: 13px;
}

.activity-details {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-light);
}

.change-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 6px;
}

.change-item.old {
  background: var(--error-100);
  color: var(--error-700);
}

.change-item.new {
  background: var(--success-100);
  color: var(--success-700);
}

/* 分页 */
.pagination {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--border-light);
}

/* 响应式 */
@media (max-width: 768px) {
  .user-center {
    padding: 0 12px;
    gap: 16px;
  }
  
  .welcome-header {
    padding: 20px 20px;
  }
  
  .welcome-content {
    flex-direction: column;
    text-align: center;
    gap: 14px;
  }
  
  .welcome-info h2 {
    font-size: 19px;
  }
  
  .cards {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .card-base {
    padding: 20px;
  }
  
  .activity-action {
    width: 100%;
  }
}

@media (max-width: 480px) {
  .welcome-header {
    padding: 18px 18px;
  }
  
  .user-avatar-large :deep(img) {
    max-width: 48px !important;
    max-height: 48px !important;
  }
  
  .welcome-info h2 {
    font-size: 18px;
  }
  
  .welcome-info p {
    font-size: 13px;
  }
  
  .card-actions .el-button {
    width: 100%;
  }
  
  .card-base {
    padding: 18px;
  }
}
</style>

