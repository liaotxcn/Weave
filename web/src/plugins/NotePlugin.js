// Note插件

import { noteService } from '../services/note.js'
import { authService } from '../services/auth.js'

class NotePlugin {
  constructor() {
    this.name = 'NotePlugin'
    this.version = '1.0.0'
    this.description = '一个简单的笔记插件'
    this.notes = []
  }

  // 初始化插件
  async initialize() {
    // 未登录时不加载笔记，避免触发后端鉴权错误
    if (!authService.isAuthenticated()) {

      return
    }
    // 从后端API加载笔记
    try {
      await this.loadNotesFromAPI()

    } catch (error) {
      console.error('笔记插件初始化失败:', error)
    }
  }

  // 获取插件信息
  getInfo() {
    return {
      name: this.name,
      version: this.version,
      description: this.description,
      noteCount: this.notes.length
    }
  }

  // 添加笔记
  async addNote(title, content) {
    try {
      // noteService.createNote会返回经过auth.js响应拦截器处理后的response.data
      const result = await noteService.createNote({ title, content })
      // 更新本地笔记列表
      await this.loadNotesFromAPI()
      return result
    } catch (error) {
      console.error('添加笔记失败:', error)
      throw error
    }
  }

  // 获取所有笔记
  getAllNotes() {
    return this.notes
  }

  // 从后端API加载笔记
  async loadNotesFromAPI() {
    try {
      // 未登录直接返回空列表
      if (!authService.isAuthenticated()) {
        this.notes = []
        return []
      }
      const data = await noteService.getAllNotes()
      // 检查响应格式，根据auth.js中响应拦截器的行为调整
      if (data && Array.isArray(data)) {
        // 处理直接返回的笔记数组
        this.notes = data
      } else if (data && data.notes && Array.isArray(data.notes)) {
        // 处理嵌套在notes字段中的笔记数组
        this.notes = data.notes
      } else {
        console.warn('Unexpected response format:', data)
        this.notes = []
      }

      return this.notes
    } catch (error) {
      console.error('从API加载笔记失败:', error)
      this.notes = []
      return []
    }
  }

  // 删除笔记
  async deleteNote(id) {
    try {
      await noteService.deleteNote(id)
      // 更新本地笔记列表
      this.loadNotesFromAPI()
    } catch (error) {
      console.error('删除笔记失败:', error)
      throw error
    }
  }

  // 更新笔记
  async updateNote(id, payload) {
    try {
      const result = await noteService.updateNote(id, payload)
      await this.loadNotesFromAPI()
      return result
    } catch (error) {
      console.error('更新笔记失败:', error)
      throw error
    }
  }



  // 渲染插件内容
  render() {
    return {
      template: `<div class="plugin-note">
                  <div class="note-header">
                    <h3 class="note-title">📝 笔记插件</h3>
                    <span class="note-meta" v-if="notes && notes.length">{{ notes.length }} 条</span>
                  </div>

                  <div class="note-form">
                    <input v-model="newNoteTitle" placeholder="笔记标题" type="text" class="input">
                    <textarea v-model="newNoteContent" placeholder="笔记内容" class="textarea" @input="autoGrow($event)"></textarea>
                    <button class="btn btn-primary" @click="addNewNote" :disabled="adding || !canAdd">{{ adding ? '添加中…' : '添加笔记' }}</button>
                    <span class="feedback" v-if="feedback">{{ feedback }}</span>
                  </div>

                  <div class="notes-list">
                    <div v-for="note in notes" :key="note.id" class="note-item">
                      <div v-if="editingId === note.id" class="edit-area">
                        <input v-model="editTitle" placeholder="编辑标题" type="text" class="input">
                        <textarea v-model="editContent" placeholder="编辑内容" class="textarea" @input="autoGrow($event)"></textarea>
                        <div class="edit-actions">
                          <button class="btn btn-primary" @click="saveEdit(note.id)" :disabled="saving">{{ saving ? '保存中…' : '保存' }}</button>
                          <button class="btn btn-secondary" @click="cancelEdit">取消</button>
                        </div>
                      </div>
                      <div v-else class="view-area">
                        <h4 class="item-title">{{ note.title }}</h4>
                        <p class="item-content">{{ note.content }}</p>
                        <small class="item-time">{{ formatDate(note.created_time) }}</small>
                        <div class="actions">
                          <button class="btn btn-secondary" @click="startEdit(note)">修改</button>
                          <button class="btn btn-danger" @click="deleteNoteItem(note.id)">删除</button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>`,
      data: function() {
        const pluginInstance = this.plugin || window.notePluginInstance || this;
        return {
          newNoteTitle: '',
          newNoteContent: '',
          notes: pluginInstance.getAllNotes ? pluginInstance.getAllNotes() : [],
          editingId: null,
          editTitle: '',
          editContent: '',
          adding: false,
          saving: false,
          feedback: ''
        }
      },
      computed: {
        canAdd: function() {
          return this.newNoteTitle && this.newNoteTitle.trim().length > 0
        }
      },
      methods: {
        autoGrow: function(e) {
          const el = e && e.target
          if (!el) return
          el.style.height = 'auto'
          el.style.height = Math.min(el.scrollHeight, 240) + 'px'
        },
        addNewNote: async function() {
          if (!this.canAdd) return
          this.adding = true
          try {
            await this.addNote(this.newNoteTitle, this.newNoteContent)
            this.newNoteTitle = ''
            this.newNoteContent = ''
            this.notes = this.getAllNotes()
            this.feedback = '已添加'
            setTimeout(() => { this.feedback = '' }, 1500)
          } catch (error) {
            console.error('添加笔记失败:', error)
            alert('添加笔记失败，请稍后重试')
          } finally {
            this.adding = false
          }
        },
        deleteNoteItem: async function(id) {
          try {
            await this.deleteNote(id)
            this.notes = this.getAllNotes()
            this.feedback = '已删除'
            setTimeout(() => { this.feedback = '' }, 1200)
          } catch (error) {
            console.error('删除笔记失败:', error)
            alert('删除笔记失败，请稍后重试')
          }
        },
        startEdit: function(note) {
          this.editingId = note.id
          this.editTitle = note.title
          this.editContent = note.content
          this.$nextTick(() => {
            const inputs = document.querySelectorAll('.edit-area .input')
            if (inputs && inputs[0]) inputs[0].focus()
          })
        },
        cancelEdit: function() {
          this.editingId = null
          this.editTitle = ''
          this.editContent = ''
        },
        saveEdit: async function(id) {
          this.saving = true
          try {
            await this.updateNote(id, { title: this.editTitle, content: this.editContent })
            this.notes = this.getAllNotes()
            this.cancelEdit()
            this.feedback = '已保存'
            setTimeout(() => { this.feedback = '' }, 1500)
          } catch (error) {
            console.error('更新笔记失败:', error)
            alert('更新笔记失败，请稍后重试')
          } finally {
            this.saving = false
          }
        },
        formatDate: function(dateString) {
          return new Date(dateString).toLocaleString()
        }
      },
      watch: {
        notes: function(newNotes) {
          this.notes = newNotes
        }
      },
      css: `.plugin-note { padding: 16px; border-radius: 10px; background: #ffffff; border: 1px solid #e5e7eb; }
            .note-header { display:flex; align-items:center; justify-content:space-between; margin-bottom: 12px; }
            .note-title { margin:0; color:#1f2937; font-weight:600; font-size:18px; }
            .note-meta { color:#6b7280; font-size:12px; }

            .note-form { display:flex; flex-direction:column; gap:8px; margin-bottom: 12px; }
            .input { width:100%; padding:10px; border:1px solid #d1d5db; border-radius:8px; background:#fff; font-size:14px; }
            .textarea { width:100%; min-height:96px; padding:10px; border:1px solid #d1d5db; border-radius:8px; background:#fff; font-size:14px; resize:vertical; }
            .feedback { margin-left:8px; color:#64748b; font-size:12px; }

            .notes-list { display:flex; flex-direction:column; gap:10px; }
            .note-item { padding:12px; border:1px solid #e5e7eb; border-radius:8px; background:#fafafa; transition: background-color .15s ease, border-color .15s ease; }
            .note-item:hover { background:#f5f5f5; border-color:#e2e8f0; }

            .item-title { margin:0 0 6px; color:#111827; font-size:16px; font-weight:600; }
            .item-content { margin:0 0 6px; white-space:pre-wrap; color:#374151; }
            .item-time { color:#9ca3af; display:block; margin-bottom:8px; }

            .actions, .edit-actions { display:flex; gap:8px; }
            .btn { border:none; padding:6px 12px; border-radius:8px; font-size:13px; cursor:pointer; }
            .btn-primary { background:#667eea; color:#fff; }
            .btn-secondary { background:#eef2ff; color:#3949ab; }
            .btn-danger { background:#ef4444; color:#fff; }
            .btn:disabled { opacity:0.6; cursor:not-allowed; }
            .edit-area .input, .edit-area .textarea { margin-bottom:8px; }`
    }
  }

  // 销毁插件
  destroy() {
    // 这里可以添加插件的清理逻辑
  }
}

export default NotePlugin