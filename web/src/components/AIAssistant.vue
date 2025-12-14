<script setup>
import { ref, nextTick } from 'vue';
import { llmService } from '../services/llm.js';

// 组件状态
const isOpen = ref(false);
const messages = ref([]);
const newMessage = ref('');
const isLoading = ref(false);
const connectionStatus = ref('unknown'); // unknown, connected, disconnected

// 初始化时加载对话历史
const initializeChat = async () => {
  try {
    connectionStatus.value = 'unknown';
    // 先检查连接状态
    const isConnected = await llmService.checkConnection();
    connectionStatus.value = isConnected ? 'connected' : 'disconnected';
    
    // 然后加载聊天历史
    const history = await llmService.getChatHistory();
    if (history && history.messages && history.messages.length > 0) {
      messages.value = history.messages;
    } else {
      // 如果没有历史消息，添加欢迎消息
      messages.value = [{
        role: 'assistant',
        content: isConnected ? '你好！我是智能助手-PaiChat，很高兴为你服务。请问有什么我可以帮助你的吗？' : '你好！我注意到与服务器的连接暂时不可用。请检查网络连接或稍后再试。'
      }];
    }
  } catch (error) {
    console.error('初始化聊天失败:', error);
    connectionStatus.value = 'disconnected';
    messages.value = [{
      role: 'assistant',
      content: '你好！我是AI智能助手-PaiChat。初始化时发生错误，请稍后再试。'
    }];
  }
};

// 组件挂载时初始化
initializeChat();

// 切换聊天窗口 - 确保方法简单直接
const toggleChat = () => {
  console.log('AI Assistant toggle clicked');
  isOpen.value = !isOpen.value;
  if (isOpen.value) {
    nextTick(() => {
      scrollToBottom();
    });
  }
};

// 滚动到底部
const scrollToBottom = () => {
  const chatContainer = document.querySelector('.ai-chat-messages');
  if (chatContainer) {
    chatContainer.scrollTop = chatContainer.scrollHeight;
  }
};

// 发送消息功能
const sendMessage = async () => {
  const message = newMessage.value.trim();
  if (!message || isLoading.value) return;

  // 添加用户消息
  messages.value.push({ role: 'user', content: message });
  newMessage.value = '';
  
  // 滚动到底部
  await nextTick();
  scrollToBottom();
  
  try {
    isLoading.value = true;
    
    // 检查连接状态
    const isConnected = await llmService.checkConnection();
    if (!isConnected) {
      connectionStatus.value = 'disconnected';
      messages.value.push({
        role: 'assistant',
        content: '⚠️ 我当前无法连接到服务器。请检查网络连接或稍后再试。'
      });
      return;
    }
    
    connectionStatus.value = 'connected';
    
    // 添加临时加载消息
    // 创建带动态ID的临时加载消息
      const loadingId = `loading-${Date.now()}`;
      messages.value.push({
        role: 'assistant',
        content: '正在思考中...',
        isLoadingMessage: true,
        loadingId: loadingId
      });
    await nextTick();
    scrollToBottom();
    
    // 调用后端API获取回复
    const response = await llmService.sendChatMessage(message);
    
    // 替换临时加载消息为实际回复
    const loadingIndex = messages.value.findIndex(msg => msg.isLoadingMessage);
    if (loadingIndex !== -1) {
      messages.value[loadingIndex] = {
        role: 'assistant',
        content: response.response || '抱歉，我暂时无法生成回复。请稍后再试。'
      };
    } else {
      // 如果找不到临时消息（不应该发生），则添加新消息
      messages.value.push({
        role: 'assistant',
        content: response.response || '抱歉，我暂时无法生成回复。请稍后再试。'
      });
    }
  } catch (error) {
    console.error('发送消息失败:', error);
    connectionStatus.value = 'disconnected';
    
    let errorMessage = '抱歉，服务暂时不可用。请稍后再试。';
    
    if (error.message.includes('超时')) {
      errorMessage = '⏳ 请求处理时间较长，服务器可能繁忙。\n\n建议：\n• 请稍等片刻后重试\n• 简化你的问题\n• 稍后再尝试连接';
    }
    
    // 找到临时加载消息并替换
    const loadingIndex = messages.value.findIndex(msg => msg.content === '正在思考中...');
    if (loadingIndex !== -1) {
      messages.value[loadingIndex] = {
        role: 'assistant',
        content: errorMessage
      };
    } else {
      messages.value.push({
        role: 'assistant',
        content: errorMessage
      });
    }
  } finally {
    isLoading.value = false;
    await nextTick();
    scrollToBottom();
  }
};

// 清空聊天
const clearChat = async () => {
  try {
    isLoading.value = true;
    
    // 调用后端API清空历史记录
    await llmService.clearChatHistory();
    
    messages.value = [{
      role: 'assistant',
      content: '对话已清空。请问有什么我可以帮助你的吗？'
    }];
  } catch (error) {
    console.error('清空聊天失败:', error);
    
    // 即使后端失败，也清空本地消息
    messages.value = [{
      role: 'assistant',
      content: '对话已在本地清空。服务器连接不可用，无法同步。'
    }];
  } finally {
    isLoading.value = false;
  }
};

// 重试连接
const retryConnection = async () => {
  if (isLoading.value) return;
  
  try {
    isLoading.value = true;
    connectionStatus.value = 'unknown';
    
    const isConnected = await llmService.checkConnection();
    connectionStatus.value = isConnected ? 'connected' : 'disconnected';
    
    if (isConnected) {
      messages.value.push({
        role: 'assistant',
        content: '✅ 连接已恢复！你可以继续对话了。'
      });
    } else {
      messages.value.push({
        role: 'assistant',
        content: '❌ 连接失败，请稍后再试或检查网络连接。'
      });
    }
  } catch (error) {
    console.error('重试连接失败:', error);
    connectionStatus.value = 'disconnected';
    messages.value.push({
      role: 'assistant',
      content: '❌ 连接失败，请稍后再试或检查网络连接。'
    });
  } finally {
    isLoading.value = false;
    await nextTick();
    scrollToBottom();
  }
};

// 处理键盘事件
const handleKeyDown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    sendMessage();
  }
};
</script>

<template>
  <!-- AI助手容器 -->
  <div class="ai-assistant-container">
    <!-- 悬浮按钮 - 简化版，确保可点击性 -->
    <button 
      class="ai-chat-icon"
      @click="toggleChat"
      title="AI智能助手-PaiChat"
      aria-label="打开AI智能助手-PaiChat"
      type="button"
      style="position: fixed; bottom: 24px; right: 24px; z-index: 9999; pointer-events: all;"
    >
      <img src="/chat.png" alt="AI助手" class="chat-icon-image" />
    </button>
    
    <!-- 聊天窗口 -->
    <div 
      class="ai-chat-window"
      v-show="isOpen"
      style="position: fixed; bottom: 100px; right: 24px; z-index: 9998;"
    >
      <!-- 聊天头部 -->
      <div class="ai-chat-header">
        <div class="ai-chat-title">
            <svg class="ai-icon" viewBox="0 0 24 24" width="20" height="20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" stroke="currentColor" stroke-width="1.5"/>
              <circle cx="12" cy="10" r="3" stroke="currentColor" stroke-width="1.5"/>
              <path d="M9 21a1 1 0 1 0 2 0 1 1 0 1 0-2 0z" fill="currentColor"/>
            </svg>
            <span>AI智能助手-PaiChat</span>
            <span 
              class="connection-status"
              :class="connectionStatus"
              :title="connectionStatus === 'connected' ? '已连接' : connectionStatus === 'disconnected' ? '连接断开' : '连接中'"
            >
              <svg v-if="connectionStatus === 'connected'" viewBox="0 0 24 24" width="14" height="14" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M20 6L9 17l-5-5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <svg v-else-if="connectionStatus === 'disconnected'" viewBox="0 0 24 24" width="14" height="14" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <div v-else class="status-loading"></div>
            </span>
          </div>
        <div class="ai-chat-actions">
            <!-- 重试按钮 - 仅在连接断开时显示 -->
            <button 
              v-if="connectionStatus === 'disconnected'"
              class="ai-action-btn retry-btn"
              @click="retryConnection"
              :disabled="isLoading"
              title="重新连接"
            >
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M1 4v6h6M23 20v-6h-6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
            
            <button 
              class="ai-action-btn" 
              @click="clearChat"
              :disabled="isLoading"
              title="清空对话"
            >
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
          <button 
            class="ai-action-btn" 
            @click="toggleChat"
            title="关闭"
          >
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        </div>
      </div>
      
      <!-- 聊天消息区域 -->
      <div class="ai-chat-messages">
        <div 
          v-for="(msg, index) in messages" 
          :key="index"
          :class="['ai-message', msg.role, { 'loading-message': msg.isLoadingMessage }]"
        >
          <div class="ai-message-avatar">
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" fill="currentColor"/>
            </svg>
          </div>
          <div class="ai-message-content">
            <template v-if="msg.isLoadingMessage">
                <span class="loading-text">正在思考中</span>
                <span class="loading-dots">
                  <span></span>
                  <span></span>
                  <span></span>
                </span>
              </template>
            <template v-else>
              {{ msg.content }}
            </template>
          </div>
        </div>
        
        <!-- 加载指示器 -->
        <div v-if="isLoading" class="ai-message assistant">
          <div class="ai-message-avatar">
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" fill="currentColor"/>
            </svg>
          </div>
          <div class="ai-message-content">
            <div class="typing-indicator">
              <div class="dot"></div>
              <div class="dot"></div>
              <div class="dot"></div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 输入区域 -->
      <div class="ai-chat-input">
        <textarea 
          v-model="newMessage"
          @keydown="handleKeyDown"
          placeholder="输入你的问题..."
          rows="1"
          :disabled="isLoading"
        ></textarea>
        <button 
          class="ai-send-btn" 
          @click="sendMessage"
          :disabled="!newMessage.trim() || isLoading"
        >
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M22 2L11 13M22 2L9 19M22 2H2" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 简化的样式，确保按钮可点击性 */
.ai-chat-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: var(--el-color-primary);
  color: white;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--el-box-shadow-light);
  transition: all 0.3s ease;
  outline: none;
  font-size: 16px;
}

.ai-chat-icon:hover {
  background: var(--el-color-primary-light-3);
  transform: translateY(-2px);
  box-shadow: var(--el-box-shadow);
}

.ai-chat-icon:active {
  transform: translateY(0);
  box-shadow: var(--el-box-shadow-light);
}

.chat-icon-image {
  width: 60px;
  height: 60px;
  object-fit: contain;
  border-radius: 50%;
  background: white;
  padding: 4px;
}

/* 聊天窗口样式 */
.ai-chat-window {
  width: 360px;
  height: 480px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  border: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
}

.ai-chat-header {
  padding: 16px;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #f8fafc;
  border-top-left-radius: 12px;
  border-top-right-radius: 12px;
}

.ai-chat-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #1e293b;
}

.ai-icon {
  color: #6366f1;
}

.ai-chat-actions {
  display: flex;
  gap: 8px;
}

.ai-action-btn {
  background: none;
  border: none;
  padding: 6px;
  border-radius: 8px;
  cursor: pointer;
  color: #94a3b8;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ai-action-btn:hover {
  background: white;
  color: #334155;
}

.ai-chat-messages {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ai-message {
  display: flex;
  gap: 12px;
  max-width: 85%;
}

.ai-message.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.ai-message.assistant {
  align-self: flex-start;
}

.ai-message-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #f1f5f9;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.ai-message.user .ai-message-avatar {
  background: #e0e7ff;
}

.ai-message.user .ai-message-avatar svg {
  color: #6366f1;
}

.ai-message.assistant .ai-message-avatar {
  background: #dbeafe;
}

.ai-message.assistant .ai-message-avatar svg {
  color: #3b82f6;
}

.ai-message-content {
  padding: 12px 16px;
  border-radius: 8px;
  line-height: 1.5;
  word-wrap: break-word;
}

.ai-message.user .ai-message-content {
  background: #6366f1;
  color: white;
  border-bottom-right-radius: 4px;
}

.ai-message.assistant .ai-message-content {
  background: #f8fafc;
  color: #1e293b;
  border-bottom-left-radius: 4px;
}

.ai-chat-input {
  padding: 16px;
  border-top: 1px solid #e2e8f0;
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.ai-chat-input textarea {
  flex: 1;
  padding: 12px;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  resize: none;
  font-size: 16px;
  line-height: 1.5;
  transition: border-color 0.2s ease;
  min-height: 44px;
  max-height: 120px;
}

.ai-chat-input textarea:focus {
  outline: none;
  border-color: #6366f1;
}

.ai-send-btn {
  background: #6366f1;
  color: white;
  border: none;
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.ai-send-btn:hover:not(:disabled) {
  background: #4f46e5;
  transform: translateY(-1px);
}

.ai-send-btn:disabled {
  background: #f1f5f9;
  color: #94a3b8;
  cursor: not-allowed;
}

/* 加载动画样式 */
.loading-message {
  font-style: italic;
  opacity: 0.8;
  display: flex;
  align-items: center;
}

.loading-text {
  margin-right: 8px;
}

.loading-dots {
  display: inline-flex;
  align-items: center;
}

.loading-dots span {
  display: inline-block;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background-color: currentColor;
  margin: 0 1px;
  animation: dot-bounce 1.4s infinite ease-in-out both;
}

.loading-dots span:nth-child(1) {
  animation-delay: 0s;
}

.loading-dots span:nth-child(2) {
  animation-delay: 0.2s;
}

.loading-dots span:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes dot-bounce {
  0%, 80%, 100% {
    transform: scale(0);
    opacity: 0.3;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

/* 连接状态指示器 */
.connection-status {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-left: 8px;
  padding: 2px 6px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.connection-status.connected {
  background-color: #dcfce7;
  color: #166534;
}

.connection-status.disconnected {
  background-color: #fee2e2;
  color: #991b1b;
}

.connection-status.unknown {
  background-color: #fef9c3;
  color: #854d0e;
}

/* 加载状态动画 */
.status-loading {
  width: 12px;
  height: 12px;
  border: 2px solid #cbd5e1;
  border-top: 2px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 重试按钮特殊样式 */
.retry-btn {
  color: #2563eb;
}

.retry-btn:hover {
  background-color: #eff6ff;
  color: #1d4ed8;
}

/* 按钮禁用样式 */
.ai-action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.ai-action-btn:disabled:hover {
  background: none;
  color: #94a3b8;
}

  /* 加载指示器 */
  .typing-indicator {
    display: flex;
    gap: 4px;
  }

  .typing-indicator .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #94a3b8;
    animation: typing 1.4s infinite ease-in-out both;
  }

  .typing-indicator .dot:nth-child(1) {
    animation-delay: -0.32s;
  }

  .typing-indicator .dot:nth-child(2) {
    animation-delay: -0.16s;
  }

  @keyframes typing {
    0%, 80%, 100% {
      transform: scale(0);
      opacity: 0.5;
    }
    40% {
      transform: scale(1);
      opacity: 1;
    }
  }

/* 滚动条样式 */
.ai-chat-messages::-webkit-scrollbar {
  width: 6px;
}

.ai-chat-messages::-webkit-scrollbar-track {
  background: #f1f5f9;
  border-radius: 3px;
}

.ai-chat-messages::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 3px;
}

.ai-chat-messages::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}

/* 响应式调整 */
@media (max-width: 480px) {
  .ai-chat-window {
    width: calc(100vw - 32px);
    height: 70vh;
    max-width: 100%;
  }
  
  .ai-chat-icon {
    width: 56px;
    height: 56px;
  }
}
</style>