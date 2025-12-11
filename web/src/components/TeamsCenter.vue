<template>
  <div class="teams-center">
    <div class="header-section">
      <h1 class="page-title">协作团队</h1>
      <button class="create-team-btn" @click="showCreateModal = true">
        <svg class="plus-icon" viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M12 5V19M5 12H19" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        创建团队
      </button>
    </div>
    
    <div class="content-card">
      <!-- 搜索与统计工具栏 -->
      <div class="toolbar">
        <div class="search-container">
          <svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="11" cy="11" r="8" stroke="currentColor" stroke-width="1.5"/>
            <path d="M21 21L16.65 16.65" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <input class="search-input" v-model="keyword" placeholder="搜索团队名称..." />
        </div>
        <div class="stats">
          <span class="team-count">共 <strong>{{ filteredTeams.length }}</strong> 个团队</span>
        </div>
      </div>
      
      <!-- 状态区域 -->
      <div v-if="loading" class="status-container loading-state">
        <div class="loading-spinner"></div>
        <p>正在加载团队列表...</p>
      </div>
      
      <div v-else-if="errorMessage" class="status-container error-state">
        <svg class="error-icon" viewBox="0 0 24 24" width="24" height="24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <circle cx="12" cy="12" r="10" stroke="#ef4444" stroke-width="1.5"/>
          <path d="M12 8V12M12 16H12.01" stroke="#ef4444" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        <p>{{ errorMessage }}</p>
        <button class="retry-btn" @click="loadTeams">重试</button>
      </div>
      
      <div v-else>
        <!-- 空状态 -->
        <div v-if="filteredTeams.length === 0" class="status-container empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24" width="48" height="48" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="12" cy="12" r="10" stroke="#cbd5e1" stroke-width="1.5"/>
            <path d="M8 12L16 12" stroke="#cbd5e1" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <h3>暂无团队</h3>
          <p>{{ keyword ? '没有找到匹配的团队' : '您还没有加入任何团队' }}</p>
        </div>
        
        <!-- 团队列表 -->
        <div v-else class="team-grid">
          <div v-for="team in filteredTeams" :key="team.id" class="team-card" @click="viewTeam(team)">
            <div class="team-header">
              <div class="team-avatar" :style="{'--avatar-bg': getAvatarColor(team.name)}">
                {{ getAvatarLetter(team.name) }}
              </div>
              <div class="team-actions">
                <button class="action-btn" @click.stop="editTeam(team)" title="编辑团队">
                  <svg viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" stroke="currentColor" stroke-width="1.5"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" stroke="currentColor" stroke-width="1.5"/>
                  </svg>
                </button>
                <button class="action-btn" @click.stop="viewTeamMembers(team)" title="查看成员">
                  <svg viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" stroke="currentColor" stroke-width="1.5"/>
                    <circle cx="8.5" cy="7" r="4" stroke="currentColor" stroke-width="1.5"/>
                    <path d="M20 8.5a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9z" stroke="currentColor" stroke-width="1.5"/>
                  </svg>
                </button>
              </div>
            </div>
            
            <div class="team-content">
              <h3 class="team-name">{{ team.name }}</h3>
              <p class="team-description" v-if="team.description">{{ team.description }}</p>
              <p class="team-description placeholder" v-else>暂无团队描述</p>
            </div>
            
            <div class="team-meta">
              <div class="meta-item">
                <svg class="meta-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" stroke="currentColor" stroke-width="1.5"/>
                  <circle cx="8.5" cy="7" r="4" stroke="currentColor" stroke-width="1.5"/>
                  <path d="M20 8.5a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9z" stroke="currentColor" stroke-width="1.5"/>
                </svg>
                <span>{{ team.members ? parseMembers(team.members).length : 0 }} 成员</span>
              </div>
              <div class="meta-item">
                <svg class="meta-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M12 22C17.5228 22 22 17.5228 22 12C22 6.47715 17.5228 2 12 2C6.47715 2 2 6.47715 2 12C2 17.5228 6.47715 22 12 22Z" stroke="currentColor" stroke-width="1.5"/>
                  <path d="M12 8V12M12 16H12.01" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                </svg>
                <span>{{ team.owner_id === this.$root.currentUser?.id ? '管理员' : '成员' }}</span>
              </div>
            </div>
            
            <div class="team-footer">
              <span class="create-time">创建于 {{ formatDate(team.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 创建团队模态框 -->
    <div v-if="showCreateModal" class="modal-overlay" @click="closeCreateModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>创建新团队</h2>
          <button class="modal-close" @click="closeCreateModal">
            <svg viewBox="0 0 24 24" width="20" height="20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <form @submit.prevent="submitCreateTeam" class="create-team-form">
            <div class="form-group">
              <label for="teamName" class="form-label">
                <span>团队名称</span>
                <span class="char-count" v-if="createForm.name.length">{{ createForm.name.length }}/50</span>
              </label>
              <input
                type="text"
                id="teamName"
                v-model="createForm.name"
                placeholder="请输入团队名称，如：产品开发团队"
                required
                maxlength="50"
                class="form-input"
              />
              <div v-if="errors.name" class="error-message">{{ errors.name }}</div>
              <div v-else-if="createForm.name.length" class="success-indicator"></div>
            </div>
            
            <div class="form-group">
              <label for="teamDescription" class="form-label">
                <span>团队描述</span>
                <span class="char-count">{{ createForm.description.length }}/200</span>
              </label>
              <textarea
                id="teamDescription"
                v-model="createForm.description"
                placeholder="请输入团队描述，帮助团队成员了解团队的目标和职责"
                rows="4"
                maxlength="200"
                class="form-textarea"
              ></textarea>
            </div>
            
            <div class="modal-footer">
              <button type="button" class="cancel-btn" @click="closeCreateModal">取消</button>
              <button type="submit" class="submit-btn" :disabled="creatingTeam">
                <span v-if="creatingTeam" class="loading-spinner-small"></span>
                {{ creatingTeam ? '创建中...' : '创建团队' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- 编辑团队模态框 -->
    <div v-if="showEditModal" class="modal-overlay" @click="closeEditModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>编辑团队信息</h2>
          <button class="modal-close" @click="closeEditModal">
            <svg viewBox="0 0 24 24" width="20" height="20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <form @submit.prevent="submitUpdateTeam" class="edit-team-form">
            <div class="form-group">
              <label for="editTeamName" class="form-label">
                <span>团队名称</span>
                <span class="char-count" v-if="editForm.name.length">{{ editForm.name.length }}/50</span>
              </label>
              <input
                type="text"
                id="editTeamName"
                v-model="editForm.name"
                placeholder="请输入团队名称"
                required
                maxlength="50"
                class="form-input"
              />
              <div v-if="errors.name" class="error-message">{{ errors.name }}</div>
            </div>
            
            <div class="form-group">
              <label for="editTeamDescription" class="form-label">
                <span>团队描述</span>
                <span class="char-count">{{ editForm.description.length }}/200</span>
              </label>
              <textarea
                id="editTeamDescription"
                v-model="editForm.description"
                placeholder="请输入团队描述"
                rows="4"
                maxlength="200"
                class="form-textarea"
              ></textarea>
            </div>
            
            <div class="modal-footer">
              <button type="button" class="cancel-btn" @click="closeEditModal">取消</button>
              <button type="submit" class="submit-btn" :disabled="updatingTeam">
                <span v-if="updatingTeam" class="loading-spinner-small"></span>
                {{ updatingTeam ? '更新中...' : '更新团队' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- 团队成员管理模态框 -->
    <div v-if="showMembersModal" class="modal-overlay" @click="closeMembersModal">
      <div class="modal-content members-modal" @click.stop>
        <div class="modal-header">
          <h2>{{ currentTeam?.name }} - 团队成员</h2>
          <button class="modal-close" @click="closeMembersModal">
            <svg viewBox="0 0 24 24" width="20" height="20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <!-- 搜索与添加成员 -->
          <div class="members-toolbar">
            <div class="search-container">
              <svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
                <circle cx="11" cy="11" r="8" stroke="currentColor" stroke-width="1.5"/>
                <path d="M21 21L16.65 16.65" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
              <input class="search-input" v-model="memberSearchKeyword" placeholder="搜索团队成员..." />
            </div>
            <button class="add-member-btn" @click="showAddMemberForm = true">
              <svg class="plus-icon" viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M12 5V19M5 12H19" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              添加成员
            </button>
          </div>

          <!-- 成员列表 -->
          <div v-if="loading" class="status-container loading-state">
            <div class="loading-spinner"></div>
            <p>正在加载成员列表...</p>
          </div>
          <div v-else class="members-list">
            <div v-if="teamMembers.length === 0" class="empty-members">
              <p>暂无团队成员</p>
            </div>
            <div v-else>
              <div v-for="member in filteredTeamMembers" :key="member.user_id" class="member-item">
                <div class="member-info">
                  <div class="member-avatar" :style="{'--avatar-bg': getAvatarColor(member.username || 'User')}">
                    {{ getAvatarLetter(member.username || 'U') }}
                  </div>
                  <div class="member-details">
                    <span class="member-username">{{ member.username || '未知用户' }}</span>
                    <span class="member-role">{{ member.role === 'owner' ? '所有者' : member.role === 'admin' ? '管理员' : '成员' }}</span>
                  </div>
                </div>
                <div class="member-actions">
                  <button 
                    class="remove-member-btn"
                    @click="removeTeamMember(member)"
                    :disabled="member.role === 'owner'"
                    :title="member.role === 'owner' ? '不能移除团队所有者' : '移除成员'"
                  >
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <path d="M3 6h18M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6m3 0V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2M10 11v6M14 11v6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                    移除
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 添加成员表单 -->
          <div v-if="showAddMemberForm" class="add-member-form-container">
            <div class="form-section-header">
              <h3>添加新成员</h3>
              <button class="close-form-btn" @click="showAddMemberForm = false">
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              </button>
            </div>
            <form @submit.prevent="submitAddMember" class="add-member-form">
              <div class="form-group">
                <label for="addMemberId" class="form-label">用户ID</label>
                <input
                  type="number"
                  id="addMemberId"
                  v-model="addMemberForm.user_id"
                  placeholder="请输入用户ID"
                  required
                  class="form-input"
                />
              </div>
              
              <div class="form-group">
                <label for="addMemberRole" class="form-label">角色</label>
                <select
                  id="addMemberRole"
                  v-model="addMemberForm.role"
                  class="form-select"
                >
                  <option value="member">成员</option>
                  <option value="admin">管理员</option>
                </select>
              </div>
              
              <div class="form-actions">
                <button type="button" class="cancel-btn" @click="showAddMemberForm = false">取消</button>
                <button type="submit" class="submit-btn" :disabled="addingMember">
                  <span v-if="addingMember" class="loading-spinner-small"></span>
                  {{ addingMember ? '添加中...' : '添加成员' }}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import api, { teamService } from '../services/auth'

export default {
  name: 'TeamsCenter',
  data() {
    return {
      loading: true,
      teams: [],
      errorMessage: '',
      keyword: '',
      showCreateModal: false,
      showEditModal: false,
      showMembersModal: false,
      creatingTeam: false,
      updatingTeam: false,
      createForm: {
        name: '',
        description: ''
      },
      editForm: {
        id: '',
        name: '',
        description: ''
      },
      currentTeam: null,
      teamMembers: [],
      memberSearchKeyword: '',
      showAddMemberForm: false,
      addingMember: false,
      addMemberForm: {
        user_id: '',
        role: 'member'
      },
      errors: {}
    }
  },
  mounted() {
    this.loadTeams()
  },
  computed: {
    // 搜索过滤团队
    filteredTeams() {
      const kw = (this.keyword || '').trim().toLowerCase()
      if (!kw) return this.teams
      return this.teams.filter(t => (t.name || '').toLowerCase().includes(kw))
    },
    // 搜索过滤团队成员
    filteredTeamMembers() {
      const kw = (this.memberSearchKeyword || '').trim().toLowerCase()
      if (!kw) return this.teamMembers
      return this.teamMembers.filter(member => {
        const username = (member.username || '').toLowerCase()
        return username.includes(kw)
      })
    }
  },
  methods: {
    getAvatarColor(name) {
      const colors = [
        '#4f46e5', '#06b6d4', '#10b981', '#f59e0b', '#ef4444',
        '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#84cc16'
      ];
      let hash = 0;
      for (let i = 0; i < name.length; i++) {
        hash = name.charCodeAt(i) + ((hash << 5) - hash);
      }
      return colors[Math.abs(hash) % colors.length];
    },
    getAvatarLetter(name) {
      return name ? name.charAt(0).toUpperCase() : 'T';
    },
    async loadTeams() {
      this.loading = true
      this.errorMessage = ''
      try {
        const res = await api.get('/api/v1/teams/')
        this.teams = Array.isArray(res) ? res : []
      } catch (e) {
        this.errorMessage = e?.response?.data?.message || '获取团队列表失败'
      } finally {
        this.loading = false
      }
    },
    
    parseMembers(membersStr) {
      try {
        if (!membersStr) return []
        if (membersStr.trim().startsWith('[')) {
          return JSON.parse(membersStr)
        }
        return membersStr.split(',').map(s => s.trim()).filter(Boolean)
      } catch (_) {
        return []
      }
    },
    
    formatDate(dt) {
      if (!dt) return '-'
      try {
        const date = new Date(dt)
        return date.toLocaleDateString('zh-CN', {
          year: 'numeric',
          month: 'short',
          day: 'numeric'
        })
      } catch (_) {
        return dt
      }
    },
    

    
    // 查看团队详情
    viewTeam(team) {
      // 这里可以添加查看团队详情的逻辑
      console.log('查看团队:', team)
    },
    
    // 编辑团队
    editTeam(team) {
      this.currentTeam = team
      this.editForm = {
        name: team.name,
        description: team.description || ''
      }
      this.showEditModal = true
      this.errors = {}
    },
    // 关闭编辑团队模态框
    closeEditModal() {
      this.showEditModal = false
      this.errors = {}
    },
    // 关闭成员管理模态框
    closeMembersModal() {
      this.showMembersModal = false
      this.currentTeam = null
      this.teamMembers = []
      this.memberSearchKeyword = ''
      this.showAddMemberForm = false
      this.addMemberForm = {
        user_id: '',
        role: 'member'
      }
    },
    
    // 打开创建团队模态框
    createTeam() {
      this.showCreateModal = true
    },
    
    // 关闭创建团队模态框
    closeCreateModal() {
      this.showCreateModal = false
      this.createForm = {
        name: '',
        description: ''
      }
      this.errors = {}
    },
    
    // 验证创建表单
    validateCreateForm() {
      const errors = {}
      
      if (!this.createForm.name?.trim()) {
        errors.name = '团队名称不能为空'
      } else if (this.createForm.name.trim().length < 2) {
        errors.name = '团队名称至少需要2个字符'
      } else if (this.createForm.name.trim().length > 50) {
        errors.name = '团队名称不能超过50个字符'
      }
      
      if (this.createForm.description && this.createForm.description.length > 200) {
        errors.description = '团队描述不能超过200个字符'
      }
      
      this.errors = errors
      return Object.keys(errors).length === 0
    },
    
    // 提交创建团队
    async submitCreateTeam() {
      if (!this.validateCreateForm()) {
        return
      }
      
      this.creatingTeam = true
      try {
        const teamData = {
          name: this.createForm.name.trim(),
          description: this.createForm.description.trim()
        }
        
        // 调用后端API创建团队
        await api.post('/api/v1/teams/', teamData)
        
        // 创建成功后刷新团队列表
        await this.loadTeams()
        
        // 显示成功提示
        alert('团队创建成功')
        
        // 关闭模态框
        this.closeCreateModal()
      } catch (e) {
        const errorMsg = e?.response?.data?.message || '创建团队失败，请稍后重试'
        alert(errorMsg)
      } finally {
        this.creatingTeam = false
      }
    },
    
    // 更新团队信息
    async submitUpdateTeam() {
      this.updatingTeam = true
      this.errors = {}
      try {
        await teamService.updateTeam(this.currentTeam.id, this.editForm)
        this.closeEditModal()
        this.loadTeams() // 更新团队列表
        alert('团队信息更新成功')
      } catch (error) {
        alert('更新团队信息失败')
        if (error.response?.data?.errors) {
          this.errors = error.response.data.errors
        }
      } finally {
        this.updatingTeam = false
      }
    },
    // 查看团队成员列表
    async viewTeamMembers(team) {
      this.currentTeam = team
      this.loading = true
      try {
        await this.loadTeamMembers(team.id)
        this.showMembersModal = true
      } catch (error) {
        alert('加载团队成员失败')
      } finally {
        this.loading = false
      }
    },
    // 加载团队成员列表
    async loadTeamMembers(teamId) {
      try {
        const response = await teamService.getTeamMembers(teamId)
        this.teamMembers = response.members || []
      } catch (error) {
        console.error('加载团队成员失败:', error)
        throw error
      }
    },
    // 搜索团队成员
    async searchTeamMembers() {
      if (!this.memberSearchKeyword.trim()) {
        await this.loadTeamMembers(this.currentTeam.id)
        return
      }
      try {
        const response = await teamService.searchTeamMembers(this.currentTeam.id, this.memberSearchKeyword)
        this.teamMembers = response.members || []
      } catch (error) {
        console.error('搜索团队成员失败:', error)
        alert('搜索团队成员失败')
      }
    },
    // 添加团队成员
    async submitAddMember() {
      this.addingMember = true
      try {
        await teamService.addTeamMember(this.currentTeam.id, this.addMemberForm)
        this.showAddMemberForm = false
        this.addMemberForm = { user_id: '', role: 'member' } // 重置表单
        await this.loadTeamMembers(this.currentTeam.id) // 刷新成员列表
        alert('添加成员成功')
      } catch (error) {
        console.error('添加成员失败:', error)
        alert('添加成员失败')
      } finally {
        this.addingMember = false
      }
    },
    // 移除团队成员
    async removeTeamMember(member) {
      if (!confirm(`确定要移除成员 ${member.username} 吗？`)) {
        return
      }
      try {
        await teamService.removeTeamMember(this.currentTeam.id, member.id)
        await this.loadTeamMembers(this.currentTeam.id) // 刷新成员列表
        alert('移除成员成功')
      } catch (error) {
        console.error('移除成员失败:', error)
        alert('移除成员失败')
      }
    }
  }
}
</script>

<style scoped>
/* 基本容器样式 */
.teams-center {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

/* 头部区域 */
.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 2px solid rgba(255, 255, 255, 0.2);
}

.page-title {
  font-size: 32px;
  font-weight: 800;
  color: white;
  margin: 0;
  letter-spacing: -0.025em;
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.create-team-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 28px;
  background: white;
  color: #667eea;
  border: none;
  border-radius: 16px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
}

.create-team-btn:hover {
  transform: translateY(-3px) scale(1.02);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  background: #f8fafc;
}

/* 内容卡片 */
.content-card {
  background: white;
  border-radius: 24px;
  padding: 28px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  transition: box-shadow 0.3s ease;
  backdrop-filter: blur(10px);
}

/* 团队卡片 */
.team-card {
  background: white;
  border-radius: 20px;
  padding: 24px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06);
  border: 1px solid #f0f4f8;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.team-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.12);
  border-color: #e8f1f8;
}

/* 团队卡片头部 */
.team-avatar {
  width: 64px;
  height: 64px;
  border-radius: 20px;
  background-color: var(--avatar-bg, #667eea);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 700;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.35s ease;
}

.team-card:hover .team-avatar {
  transform: scale(1.1) rotate(5deg);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.2);
}

/* 模态框样式 */
.modal-content {
  background: white;
  border-radius: 24px;
  width: 100%;
  max-width: 520px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  transform: scale(0.95) translateY(20px);
  opacity: 0;
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.35s ease;
  animation: modalSlideIn 0.35s cubic-bezier(0.4, 0, 0.2, 1) forwards;
  padding: 32px;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

/* 按钮样式 */
.submit-btn {
  padding: 14px 28px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 14px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(102, 126, 234, 0.3);
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-4px) scale(1.03);
  box-shadow: 0 8px 24px rgba(102, 126, 234, 0.45);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .teams-center {
    padding: 16px;
  }
  
  .content-card {
    padding: 20px;
  }
  
  .team-grid {
    grid-template-columns: 1fr;
    gap: 20px;
  }
  
  .modal-content {
    padding: 24px;
    margin: 16px;
  }
}

/* 其他保持不变的样式 */
.primary-btn {
  padding: 12px 24px;
  background: #4f46e5;
  color: white;
  border: none;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.35s ease;
}

.primary-btn:hover {
  background: #4338ca;
  transform: translateY(-3px);
  box-shadow: 0 10px 30px rgba(79, 70, 229, 0.35);
}

.team-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 24px;
  animation: fadeInUp 0.6s ease-out;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.team-card {
  background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  padding: 24px;
  cursor: pointer;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  animation: fadeIn 0.5s ease-out;
  opacity: 0;
  animation-fill-mode: forwards;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.team-card:hover {
  transform: translateY(-6px) scale(1.02);
  border-color: #cbd5e1;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.12);
}

.team-card:active {
  transform: translateY(-2px) scale(0.98);
  transition: transform 0.15s ease;
}

.team-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18px;
}

.team-avatar {
  width: 64px;
  height: 64px;
  background-color: var(--avatar-bg, #4f46e5);
  color: white;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 700;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.team-avatar::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.15) 0%, rgba(255, 255, 255, 0) 100%);
  z-index: 1;
}

.team-avatar span {
  position: relative;
  z-index: 2;
}

.team-card:hover .team-avatar {
  transform: scale(1.05) translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
}

.team-card:hover .team-avatar {
  transform: scale(1.05) rotate(2deg);
}

.team-actions {
  display: flex;
  gap: 12px;
  opacity: 0;
  transition: opacity 0.3s ease, transform 0.3s ease;
  transform: translateY(-4px);
  margin-top: 8px;
}

.team-card:hover .team-actions {
  opacity: 1;
  transform: translateY(0);
}

.action-btn {
  width: 36px;
  height: 36px;
  border: none;
  background: #f1f5f9;
  border-radius: 10px;
  color: #64748b;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  transform: scale(0.95);
}

.action-btn:hover {
  background: #4f46e5;
  color: white;
  transform: translateY(-3px) scale(1.05);
  box-shadow: 0 6px 16px rgba(79, 70, 229, 0.35);
}

.action-btn:active {
  transform: translateY(-1px) scale(0.98);
  transition: transform 0.15s ease;
}

.team-content {
  margin-bottom: 20px;
}

.team-name {
  font-size: 20px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 10px 0;
  transition: color 0.3s ease;
  letter-spacing: -0.025em;
}

.team-card:hover .team-name {
  color: #667eea;
}

.team-description {
  font-size: 14px;
  color: #64748b;
  line-height: 1.6;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.team-description.placeholder {
  color: #94a3b8;
  font-style: italic;
}

.team-meta {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #f1f5f9;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
}

.meta-icon {
  stroke: currentColor;
  width: 16px;
  height: 16px;
}

.team-footer {
  font-size: 12px;
  color: #94a3b8;
}

.create-time {
  display: flex;
  align-items: center;
  gap: 6px;
}

/* 创建团队模态框 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.modal-content {
  background: white;
  border-radius: 18px;
  width: 100%;
  max-width: 520px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.25);
  transform: scale(0.95) translateY(20px);
  opacity: 0;
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.35s ease;
  animation: modalSlideIn 0.35s cubic-bezier(0.4, 0, 0.2, 1) forwards;
  padding: 32px;
}

@keyframes modalSlideIn {
  to {
    transform: scale(1) translateY(0);
    opacity: 1;
  }
}

.modal-content:hover {
  transform: scale(1);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 28px;
  padding: 0 8px;
  border-bottom: none;
}

.modal-header h2 {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  color: #1e293b;
  letter-spacing: -0.02em;
}

.modal-close {
  background: #f1f5f9;
  border: none;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #64748b;
  border-radius: 10px;
  transition: all 0.3s ease;
}

.modal-close:hover {
  background: #e2e8f0;
  color: #334155;
  transform: rotate(90deg);
}

.modal-body {
  padding: 0 8px;
}

.create-team-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.form-group {
  margin-bottom: 28px;
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0;
  font-weight: 600;
  color: #334155;
  font-size: 15px;
}

.char-count {
  font-size: 12px;
  font-weight: 400;
  color: #94a3b8;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 14px 18px;
  border: 2px solid #e2e8f0;
  border-radius: 14px;
  font-size: 15px;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  background: #f8fafc;
  box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.05);
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.15), inset 0 1px 3px rgba(0, 0, 0, 0.05);
  background: white;
  transform: translateY(-2px);
}

.form-textarea {
  resize: vertical;
  min-height: 120px;
  line-height: 1.6;
}

.success-indicator {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background-color: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4);
  }
  70% {
    box-shadow: 0 0 0 10px rgba(16, 185, 129, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0);
  }
}

.error-message {
  color: #ef4444;
  font-size: 13px;
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  animation: shake 0.5s ease-in-out;
}

.error-message::before {
  content: '⚠';
  font-size: 14px;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  10%, 30%, 50%, 70%, 90% { transform: translateX(-2px); }
  20%, 40%, 60%, 80% { transform: translateX(2px); }
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 16px;
  padding-top: 28px;
  padding-bottom: 8px;
  border-top: 1px solid #f1f5f9;
  flex-wrap: wrap;
}

.cancel-btn {
  padding: 12px 24px;
  background: white;
  color: #64748b;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.35s ease;
}

.cancel-btn:hover {
  border-color: #cbd5e1;
  color: #334155;
  transform: translateY(-2px);
  background: #f8fafc;
}

.submit-btn {
  padding: 12px 24px;
  background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
  color: white;
  border: none;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.submit-btn::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.2), transparent);
  transition: left 0.5s ease;
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-3px) scale(1.02);
  box-shadow: 0 8px 24px rgba(79, 70, 229, 0.35);
}

.submit-btn:hover:not(:disabled)::before {
  left: 100%;
}

.submit-btn:active:not(:disabled) {
  transform: translateY(-1px) scale(0.98);
  transition: transform 0.15s ease;
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.loading-spinner-small {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top: 2px solid white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .teams-center {
    padding: 16px;
  }
  
  .header-section {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
  
  .content-card {
    padding: 16px;
  }
  
  .toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  
  .search-container {
    max-width: 100%;
  }
  
  .team-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .team-card {
    padding: 16px;
  }
}
</style>