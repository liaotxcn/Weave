<template>
  <div class="teams-center">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-bg-decoration"></div>
      <div class="header-content">
        <div class="header-icon-wrapper">
          <div class="header-icon">
            <el-icon :size="28"><UserFilled /></el-icon>
          </div>
        </div>
        <div class="header-text">
          <h1>团队中心</h1>
          <p>管理您的团队、成员和协作项目</p>
        </div>
      </div>
    </div>

    <!-- 主内容区 -->
    <div class="main-content">
      <!-- 搜索与统计工具栏 -->
      <div class="toolbar">
        <div class="toolbar-left">
          <div class="search-box el-input-reset">
            <el-input
              v-model="keyword"
              placeholder="搜索团队名称..."
              clearable
              :prefix-icon="Search"
            />
          </div>
          <div class="team-stats">
            <span class="stats-label">共</span>
            <span class="stats-value">{{ filteredTeams.length }}</span>
            <span class="stats-label">个团队</span>
          </div>
        </div>
        <el-button type="primary" @click="showCreateModal = true" :icon="Plus" class="create-btn">
          创建团队
        </el-button>
      </div>
      
      <!-- 状态区域 -->
      <div v-if="loading" class="status-container loading-state">
        <el-skeleton :rows="3" animated />
      </div>
      
      <div v-else-if="errorMessage" class="status-container error-state">
        <el-result
          icon="error"
          title="加载失败"
          sub-title="{{ errorMessage }}"
        >
          <template #extra>
            <el-button type="primary" @click="loadTeams" :icon="Loading">重试</el-button>
          </template>
        </el-result>
      </div>
      
      <div v-else>
        <!-- 空状态 -->
        <el-empty v-if="filteredTeams.length === 0" :image-size="200">
          <template #description>
            <span>{{ keyword ? '没有找到匹配的团队' : '您还没有加入任何团队' }}</span>
          </template>
          <template #footer>
            <el-button type="primary" @click="showCreateModal = true" :icon="Plus">创建第一个团队</el-button>
          </template>
        </el-empty>
        
        <!-- 团队列表 -->
        <div v-else class="team-grid">
          <div
            v-for="team in filteredTeams"
            :key="team.id"
            class="team-card card-base"
            @click="viewTeam(team)"
          >
            <div class="team-header">
              <div class="team-avatar" :style="{'--avatar-bg': getAvatarColor(team.name)}">
                {{ getAvatarLetter(team.name) }}
              </div>
              <div class="team-actions">
                <el-button
                  type="text"
                  @click.stop="editTeam(team)"
                  title="编辑团队"
                  size="small"
                >
                  <el-icon><EditPen /></el-icon>
                </el-button>
                <el-button
                  type="text"
                  @click.stop="viewTeamMembers(team)"
                  title="查看成员"
                  size="small"
                >
                  <el-icon><UserFilled /></el-icon>
                </el-button>
              </div>
            </div>

            <div class="team-content">
              <h3 class="team-name">{{ team.name }}</h3>
              <p class="team-description" v-if="team.description">{{ team.description }}</p>
              <p class="team-description placeholder" v-else>暂无团队描述</p>
            </div>

            <div class="team-meta">
              <span class="meta-tag">
                <el-icon><UserFilled /></el-icon>
                {{ team.members ? parseMembers(team.members).length : 0 }} 成员
              </span>
              <span class="meta-tag" :class="team.owner_id === this.$root.currentUser?.id ? 'tag-owner' : 'tag-member'">
                {{ team.owner_id === this.$root.currentUser?.id ? '管理员' : '成员' }}
              </span>
            </div>

            <div class="team-footer">
              <span class="create-time">创建于 {{ formatDate(team.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 创建团队模态框 -->
    <el-dialog
      v-model="showCreateModal"
      title="创建新团队"
      width="520px"
      destroy-on-close
      class="team-dialog"
    >
      <div class="dialog-body el-input-reset">
        <div class="form-group">
          <label class="form-label">团队名称 <span class="required">*</span></label>
          <el-input
            v-model="createForm.name"
            placeholder="请输入团队名称，如：产品开发团队"
            maxlength="50"
            show-word-limit
          />
        </div>

        <div class="form-group">
          <label class="form-label">团队描述</label>
          <el-input
            v-model="createForm.description"
            type="textarea"
            placeholder="请输入团队描述，帮助团队成员了解团队的目标和职责"
            :rows="4"
            maxlength="200"
            show-word-limit
          />
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="closeCreateModal" class="cancel-btn">取消</el-button>
          <el-button type="primary" @click="submitCreateTeam" :loading="creatingTeam" :icon="Plus" class="submit-btn">
            {{ creatingTeam ? '创建中...' : '创建团队' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑团队模态框 -->
    <el-dialog
      v-model="showEditModal"
      title="编辑团队信息"
      width="520px"
      destroy-on-close
      class="team-dialog"
    >
      <div class="dialog-body el-input-reset">
        <div class="form-group">
          <label class="form-label">团队名称 <span class="required">*</span></label>
          <el-input
            v-model="editForm.name"
            placeholder="请输入团队名称"
            maxlength="50"
            show-word-limit
          />
        </div>

        <div class="form-group">
          <label class="form-label">团队描述</label>
          <el-input
            v-model="editForm.description"
            type="textarea"
            placeholder="请输入团队描述"
            :rows="4"
            maxlength="200"
            show-word-limit
          />
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="closeEditModal" class="cancel-btn">取消</el-button>
          <el-button type="primary" @click="submitUpdateTeam" :loading="updatingTeam" :icon="EditPen" class="submit-btn">
            {{ updatingTeam ? '更新中...' : '更新团队' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 团队成员管理模态框 -->
    <el-dialog
      v-model="showMembersModal"
      :title="currentTeam?.name + ' - 团队成员'"
      width="800px"
      destroy-on-close
    >
      <div class="modal-body">
        <!-- 搜索与添加成员 -->
        <div class="members-toolbar">
          <div class="el-input-reset" style="width: 300px;">
          <el-input
            v-model="memberSearchKeyword"
            placeholder="搜索团队成员..."
            clearable
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          </div>
          <el-button type="primary" @click="showAddMemberForm = true" :icon="Plus">
            添加成员
          </el-button>
        </div>

        <!-- 成员列表 -->
        <div v-if="loading" class="status-container loading-state">
          <el-skeleton :rows="5" animated />
        </div>
        <div v-else class="members-list">
          <el-empty v-if="teamMembers.length === 0" description="暂无团队成员" />
          <el-table 
            v-else 
            :data="filteredTeamMembers" 
            stripe 
            style="width: 100%;"
            border
            size="small"
          >
            <el-table-column prop="username" label="用户名" width="200">
              <template #default="scope">
                <el-avatar 
                  :size="32" 
                  :style="{ backgroundColor: getAvatarColor(scope.row.username || 'User'), marginRight: '10px' }"
                >
                  {{ getAvatarLetter(scope.row.username || 'U') }}
                </el-avatar>
                <span>{{ scope.row.username || '未知用户' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="role" label="角色" width="140">
              <template #default="scope">
                <el-tag :type="scope.row.role === 'owner' ? 'success' : scope.row.role === 'admin' ? 'warning' : 'info'">
                  {{ scope.row.role === 'owner' ? '所有者' : scope.row.role === 'admin' ? '管理员' : '成员' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="scope">
                <el-button
                  type="danger"
                  text
                  size="small"
                  @click="removeTeamMember(scope.row)"
                  :disabled="scope.row.role === 'owner'"
                  :title="scope.row.role === 'owner' ? '不能移除团队所有者' : '移除成员'"
                >
                  移除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 添加成员表单 -->
        <el-divider v-if="showAddMemberForm" />
        <div v-if="showAddMemberForm" class="add-member-form-container">
          <h3>添加新成员</h3>
          <div class="dialog-body el-input-reset">
            <div class="form-group">
              <label class="form-label">用户ID <span class="required">*</span></label>
              <el-input
                type="number"
                v-model="addMemberForm.user_id"
                placeholder="请输入用户ID"
              />
            </div>

            <div class="form-group">
              <label class="form-label">角色 <span class="required">*</span></label>
              <el-select
                v-model="addMemberForm.role"
                style="width: 100%;"
                placeholder="请选择角色"
              >
                <el-option value="member">成员</el-option>
                <el-option value="admin">管理员</el-option>
              </el-select>
            </div>
          </div>
          <div class="form-actions">
            <el-button @click="showAddMemberForm = false" class="cancel-btn">取消</el-button>
            <el-button type="primary" @click="submitAddMember" :loading="addingMember" :icon="Plus" class="submit-btn">
              {{ addingMember ? '添加中...' : '添加成员' }}
            </el-button>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { Search, Plus, EditPen, UserFilled, CircleCloseFilled, Loading } from '@element-plus/icons-vue'
import api, { teamService } from '../services/auth'

export default {
  name: 'TeamsCenter',
  components: {
    Search,
    Plus,
    EditPen,
    UserFilled,
    CircleCloseFilled,
    Loading
  },
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
    
    // 提交创建团队
    async submitCreateTeam() {
      // 手动验证表单
      if (!this.createForm.name || !this.createForm.name.trim()) {
        this.$message.warning('请输入团队名称')
        return
      }

      if (this.createForm.name.trim().length < 2 || this.createForm.name.trim().length > 50) {
        this.$message.warning('团队名称长度在 2 到 50 个字符')
        return
      }

      if (this.createForm.description && this.createForm.description.length > 200) {
        this.$message.warning('团队描述不能超过 200 个字符')
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
        this.$message.success('团队创建成功')

        // 关闭模态框
        this.closeCreateModal()
      } catch (e) {
        const errorMsg = e?.response?.data?.message || '创建团队失败，请稍后重试'
        this.$message.error(errorMsg)
      } finally {
        this.creatingTeam = false
      }
    },
      
    // 更新团队信息
    async submitUpdateTeam() {
      // 手动验证表单
      if (!this.editForm.name || !this.editForm.name.trim()) {
        this.$message.warning('请输入团队名称')
        return
      }

      if (this.editForm.name.trim().length < 2 || this.editForm.name.trim().length > 50) {
        this.$message.warning('团队名称长度在 2 到 50 个字符')
        return
      }

      if (this.editForm.description && this.editForm.description.length > 200) {
        this.$message.warning('团队描述不能超过 200 个字符')
        return
      }

      this.updatingTeam = true
      try {
        await teamService.updateTeam(this.currentTeam.id, this.editForm)
        this.closeEditModal()
        this.loadTeams() // 更新团队列表
        this.$message.success('团队信息更新成功')
      } catch (error) {
        this.$message.error('更新团队信息失败')
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
        await teamService.removeTeamMember(this.currentTeam.id, member.user_id)
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
.teams-center {
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0;
  max-width: 1200px;
  margin: 0 auto;
  min-height: 100%;
  font-family: var(--font-sans);
}

/* ===== 页面头部 ===== */
.page-header {
  position: relative;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #a78bfa 100%);
  border-radius: var(--radius-xl);
  padding: 32px 36px;
  color: white;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.25), 0 0 40px rgba(139, 92, 246, 0.1);
}

.header-bg-decoration {
  position: absolute;
  top: -50%;
  right: -10%;
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.15) 0%, transparent 70%);
  border-radius: 50%;
  animation: float 8s ease-in-out infinite;
}

.header-bg-decoration::before {
  content: '';
  position: absolute;
  bottom: -30%;
  left: -5%;
  width: 300px;
  height: 300px;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, transparent 70%);
  border-radius: 50%;
  animation: float 6s ease-in-out infinite reverse;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0) scale(1);
  }
  50% {
    transform: translateY(-20px) scale(1.05);
  }
}

.header-content {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 20px;
}

.header-icon-wrapper {
  flex-shrink: 0;
}

.header-icon {
  width: 64px;
  height: 64px;
  background: rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(10px);
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15), inset 0 1px 0 rgba(255, 255, 255, 0.25);
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}

.page-header:hover .header-icon {
  transform: rotate(-5deg) scale(1.05);
  background: rgba(255, 255, 255, 0.25);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2), inset 0 1px 0 rgba(255, 255, 255, 0.3);
}

.header-text h1 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -0.02em;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.header-text p {
  margin: 0;
  opacity: 0.95;
  font-size: 15px;
  font-weight: 400;
  letter-spacing: 0.01em;
}

/* ===== 主内容区 ===== */
.main-content {
  background: var(--color-surface);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  padding: 28px 32px;
}

/* ===== 工具栏 ===== */
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 28px;
  flex-wrap: wrap;
  gap: 20px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

.search-box {
  width: 320px;
}

.team-stats {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 8px 16px;
  background: var(--bg-secondary);
  border-radius: 10px;
  border: 1px solid var(--border-light);
}

.stats-label {
  font-size: 14px;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.stats-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--primary-600);
}

.create-btn {
  height: 40px;
  font-size: 14px;
  font-weight: 600;
  border-radius: 10px;
  padding: 0 24px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.create-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(99, 102, 241, 0.3);
}

/* ===== 状态区域 ===== */
.status-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  text-align: center;
}

.loading-state {
  color: var(--color-text-secondary);
}

.error-state {
  color: var(--error);
}

/* ===== 团队网格 ===== */
.team-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 24px;
}

/* ===== 团队卡片基础样式 ===== */
.card-base {
  background: white;
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  padding: 24px;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.card-base::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: linear-gradient(90deg, var(--primary-500), var(--primary-400));
  transform: scaleX(0);
  transition: transform 0.3s ease;
}

.card-base:hover {
  transform: translateY(-6px);
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.12);
  border-color: var(--border-medium);
}

.card-base:hover::before {
  transform: scaleX(1);
}

/* ===== 团队头部 ===== */
.team-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.team-actions {
  display: flex;
  gap: 6px;
  opacity: 0;
  transition: opacity 0.25s ease;
}

.card-base:hover .team-actions {
  opacity: 1;
}

/* ===== 团队头像 ===== */
.team-avatar {
  width: 64px;
  height: 64px;
  background-color: var(--avatar-bg, var(--color-primary));
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
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.18), transparent);
  z-index: 1;
}

.card-base:hover .team-avatar {
  transform: scale(1.08) rotate(3deg);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.22);
}

/* ===== 团队内容 ===== */
.team-content {
  margin-bottom: 20px;
}

.team-name {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0 0 10px 0;
  letter-spacing: -0.02em;
  transition: color 0.3s ease;
  line-height: 1.3;
}

.card-base:hover .team-name {
  color: var(--primary-600);
}

.team-description {
  font-size: 14px;
  color: var(--color-text-secondary);
  line-height: 1.7;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.team-description.placeholder {
  color: var(--color-text-tertiary);
  font-style: italic;
}

/* ===== 团队元信息 ===== */
.team-meta {
  display: flex;
  gap: 10px;
  margin-bottom: 18px;
  flex-wrap: wrap;
}

.meta-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 8px;
  background: var(--bg-secondary);
  color: var(--color-text-secondary);
  border: 1px solid var(--border-light);
  transition: all 0.25s ease;
}

.meta-tag.tag-owner {
  background: rgba(16, 185, 129, 0.1);
  color: #059669;
  border-color: rgba(16, 185, 129, 0.25);
}

.meta-tag.tag-member {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
  border-color: rgba(245, 158, 11, 0.25);
}

/* ===== 团队底部 ===== */
.team-footer {
  text-align: right;
  padding-top: 16px;
  border-top: 1px solid var(--bg-tertiary);
}

.create-time {
  font-size: 12px;
  color: var(--color-text-tertiary);
  font-weight: 500;
}

/* ===== 对话框统一样式 ===== */
.team-dialog :deep(.el-dialog) {
  border-radius: 16px;
  overflow: hidden;
}

.team-dialog :deep(.el-dialog__header) {
  padding: 24px 28px 0 !important;
  margin-right: 0 !important;
}

.team-dialog :deep(.el-dialog__title) {
  font-size: 20px !important;
  font-weight: 700 !important;
  color: var(--color-text-primary) !important;
}

.team-dialog :deep(.el-dialog__body) {
  padding: 24px 28px !important;
}

.team-dialog :deep(.el-dialog__footer) {
  padding: 0 28px 24px !important;
}

.dialog-body {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.required {
  color: var(--error);
  margin-left: 2px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-btn,
.submit-btn {
  height: 38px;
  font-size: 14px;
  font-weight: 600;
  border-radius: 9px;
  padding: 0 20px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.25);
}

/* ===== 成员管理对话框 ===== */
.members-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  flex-wrap: wrap;
  gap: 16px;
}

.members-list {
  margin-bottom: 24px;
}

.add-member-form-container {
  margin-top: 24px;
}

.add-member-form-container h3 {
  margin-top: 0;
  margin-bottom: 20px;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}

/* ===== 响应式设计 ===== */
@media (max-width: 968px) {
  .teams-center {
    gap: 20px;
  }

  .page-header {
    padding: 28px 28px;
  }

  .header-icon {
    width: 56px;
    height: 56px;
  }

  .header-icon :deep(.el-icon) {
    font-size: 24px !important;
  }

  .header-text h1 {
    font-size: 25px;
  }

  .main-content {
    padding: 24px 26px;
  }

  .team-grid {
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 20px;
  }
}

@media (max-width: 768px) {
  .teams-center {
    padding: 0 12px;
    gap: 16px;
  }

  .page-header {
    padding: 24px 22px;
  }

  .header-content {
    flex-direction: column;
    text-align: center;
    gap: 16px;
  }

  .header-icon {
    width: 60px;
    height: 60px;
  }

  .header-text h1 {
    font-size: 23px;
  }

  .header-text p {
    font-size: 14px;
  }

  .main-content {
    padding: 20px;
  }

  .toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .toolbar-left {
    flex-direction: column;
    align-items: stretch;
  }

  .search-box {
    width: 100%;
  }

  .create-btn {
    width: 100%;
  }

  .members-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .team-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .card-base {
    padding: 20px;
  }
}

@media (max-width: 480px) {
  .page-header {
    padding: 20px 18px;
  }

  .header-icon {
    width: 52px;
    height: 52px;
  }

  .header-icon :deep(.el-icon) {
    font-size: 22px !important;
  }

  .header-text h1 {
    font-size: 21px;
  }

  .header-text p {
    font-size: 13px;
  }

  .main-content {
    padding: 18px;
  }

  .card-base {
    padding: 18px;
  }

  .team-avatar {
    width: 56px;
    height: 56px;
    font-size: 24px;
  }

  .team-name {
    font-size: 18px;
  }

  .status-container {
    padding: 60px 16px;
  }
}
</style>