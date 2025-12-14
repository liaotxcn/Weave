<template>
  <div class="teams-center">
    <el-card class="content-card" shadow="hover">
      <!-- 搜索与统计工具栏 -->
      <div class="toolbar">
        <div class="toolbar-left">
          <el-input
            v-model="keyword"
            placeholder="搜索团队名称..."
            style="width: 300px;"
            clearable
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-statistic title="共" :value="filteredTeams.length" suffix="个团队" />
        </div>
        <el-button type="primary" @click="showCreateModal = true" :icon="Plus">
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
          <el-card
            v-for="team in filteredTeams"
            :key="team.id"
            class="team-card"
            @click="viewTeam(team)"
            shadow="hover"
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
              <el-tag size="small" type="info" effect="plain">
                <el-icon><UserFilled /></el-icon>
                {{ team.members ? parseMembers(team.members).length : 0 }} 成员
              </el-tag>
              <el-tag size="small" :type="team.owner_id === this.$root.currentUser?.id ? 'success' : 'warning'" effect="plain">
                {{ team.owner_id === this.$root.currentUser?.id ? '管理员' : '成员' }}
              </el-tag>
            </div>
            
            <div class="team-footer">
              <span class="create-time">创建于 {{ formatDate(team.created_at) }}</span>
            </div>
          </el-card>
        </div>
      </div>
    </el-card>
    
    <!-- 创建团队模态框 -->
    <el-dialog
      v-model="showCreateModal"
      title="创建新团队"
      width="500px"
      destroy-on-close
    >
      <el-form :model="createForm" ref="createFormRef" @submit.prevent="submitCreateTeam" label-position="top">
        <el-form-item 
          label="团队名称" 
          prop="name"
          :rules="[{ required: true, message: '请输入团队名称', trigger: 'blur' },
                   { min: 2, max: 50, message: '团队名称长度在 2 到 50 个字符', trigger: 'blur' }]"
        >
          <el-input
            v-model="createForm.name"
            placeholder="请输入团队名称，如：产品开发团队"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>
        
        <el-form-item 
          label="团队描述"
          prop="description"
          :rules="[{ max: 200, message: '团队描述不能超过 200 个字符', trigger: 'blur' }]"
        >
          <el-input
            v-model="createForm.description"
            type="textarea"
            placeholder="请输入团队描述，帮助团队成员了解团队的目标和职责"
            rows="4"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeCreateModal">取消</el-button>
        <el-button type="primary" @click="submitCreateTeam" :loading="creatingTeam" :icon="Plus">
          {{ creatingTeam ? '创建中...' : '创建团队' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑团队模态框 -->
    <el-dialog
      v-model="showEditModal"
      title="编辑团队信息"
      width="500px"
      destroy-on-close
    >
      <el-form :model="editForm" ref="editFormRef" @submit.prevent="submitUpdateTeam" label-position="top">
        <el-form-item 
          label="团队名称" 
          prop="name"
          :rules="[{ required: true, message: '请输入团队名称', trigger: 'blur' },
                   { min: 2, max: 50, message: '团队名称长度在 2 到 50 个字符', trigger: 'blur' }]"
        >
          <el-input
            v-model="editForm.name"
            placeholder="请输入团队名称"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>
        
        <el-form-item 
          label="团队描述"
          prop="description"
          :rules="[{ max: 200, message: '团队描述不能超过 200 个字符', trigger: 'blur' }]"
        >
          <el-input
            v-model="editForm.description"
            type="textarea"
            placeholder="请输入团队描述"
            rows="4"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeEditModal">取消</el-button>
        <el-button type="primary" @click="submitUpdateTeam" :loading="updatingTeam" :icon="EditPen">
          {{ updatingTeam ? '更新中...' : '更新团队' }}
        </el-button>
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
          <el-input
            v-model="memberSearchKeyword"
            placeholder="搜索团队成员..."
            style="width: 300px;"
            clearable
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
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
          <el-form :model="addMemberForm" ref="addMemberFormRef" @submit.prevent="submitAddMember" label-position="top">
            <el-form-item 
              label="用户ID" 
              prop="user_id"
              :rules="[{ required: true, message: '请输入用户ID', trigger: 'blur' }]"
            >
              <el-input
                type="number"
                v-model="addMemberForm.user_id"
                placeholder="请输入用户ID"
              />
            </el-form-item>
            
            <el-form-item 
              label="角色" 
              prop="role"
              :rules="[{ required: true, message: '请选择角色', trigger: 'change' }]"
            >
              <el-select
                v-model="addMemberForm.role"
                style="width: 100%;"
              >
                <el-option value="member">成员</el-option>
                <el-option value="admin">管理员</el-option>
              </el-select>
            </el-form-item>
            
            <div class="form-actions">
              <el-button @click="showAddMemberForm = false">取消</el-button>
              <el-button type="primary" @click="submitAddMember" :loading="addingMember" :icon="Plus">
                {{ addingMember ? '添加中...' : '添加成员' }}
              </el-button>
            </div>
          </el-form>
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
      errors: {},
      createFormRef: null,
      editFormRef: null,
      addMemberFormRef: null
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
    
    // 提交创建团队
    async submitCreateTeam() {
      if (!this.$refs.createFormRef) return;
      
      await this.$refs.createFormRef.validate(async (valid) => {
        if (!valid) return;
        
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
      });
    },
      
    // 更新团队信息
    async submitUpdateTeam() {
      if (!this.$refs.editFormRef) return;
      
      await this.$refs.editFormRef.validate(async (valid) => {
        if (!valid) return;
        
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
      });
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

/* 内容卡片 */
.content-card {
  border-radius: 24px;
  padding: 28px;
  backdrop-filter: blur(10px);
}

/* 团队卡片 */
.team-card {
  border-radius: 20px;
  margin-bottom: 20px;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}

.team-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.12);
}

/* 团队卡片内部样式 */
.team-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.team-actions {
  display: flex;
  gap: 8px;
}

.team-content {
  margin-bottom: 16px;
}

.team-name {
  font-size: 20px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 10px 0;
  letter-spacing: -0.025em;
}

.team-card:hover .team-name {
  color: #667eea;
}

.team-meta {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.team-footer {
  text-align: right;
}

/* 工具栏样式 */
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  flex-wrap: wrap;
  gap: 16px;
}

/* 搜索与统计容器 */
.toolbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

/* 团队数量统计样式 */
:deep(.el-statistic) {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin-left: 16px;
  align-self: center;
}

/* 成员管理模态框样式 */
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
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}

/* 状态容器 */
.status-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.loading-state {
  color: #64748b;
}

.error-state {
  color: #ef4444;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .teams-center {
    padding: 16px;
  }
  
  .content-card {
    padding: 20px;
  }
  
  .header-section {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
  
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  
  .members-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  
  .team-grid {
    grid-template-columns: 1fr;
    gap: 16px;
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