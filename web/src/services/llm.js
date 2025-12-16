// LLM服务调用模块
export const llmService = {
  // API基础配置
  config: {
    baseURL: window.location.origin, // 使用当前域名作为基础URL
    timeout: 180000, // 增加到3分钟超时，适应LLM处理时间较长的情况
  },

  // 创建带超时控制的fetch请求
  async fetchWithTimeout(url, options, timeout = this.config.timeout) {
    const controller = new AbortController();
    const { signal } = controller;
    
    // 设置超时
    const timeoutId = setTimeout(() => controller.abort(), timeout);
    
    try {
      // 显示超时警告（但不中止请求）
      const warningTimeoutId = setTimeout(() => {

      }, Math.min(timeout / 2, 30000)); // 超时一半或30秒时显示警告
      
      const response = await fetch(url, {
        ...options,
        signal
      });
      
      clearTimeout(warningTimeoutId);
      return response;
    } finally {
      clearTimeout(timeoutId);
    }
  },

  // 发送聊天消息
  async sendChatMessage(message) {
    try {
      const startTime = Date.now();

      
      // 使用配置的baseURL而不是硬编码URL
      const endpoint = `${this.config.baseURL}/plugins/LLMChat/api/chat`;

      
      const response = await this.fetchWithTimeout(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // 包含cookie以确保会话一致
        body: JSON.stringify({ message })
      });
      
      clearTimeout(timeoutId);
      
      const endTime = Date.now();

      
      if (!response.ok) {
        // 尝试获取错误详情
        let errorDetails = '';
        try {
          const errorData = await response.json();
          errorDetails = JSON.stringify(errorData);
        } catch (e) {
          errorDetails = await response.text();
        }
        throw new Error(`API错误: ${response.status} ${response.statusText}, 详情: ${errorDetails}`);
      }
      
      // 解析响应
      const data = await response.json();

      
      // 确保响应格式正确
      if (!data || typeof data.response === 'undefined') {
        throw new Error('无效的响应格式: 缺少response字段');
      }
      
      return data;
    } catch (error) {
      if (error.name === 'AbortError') {
        console.error('[LLM] 请求超时:', error);
        throw new Error('请求超时，请稍后再试');
      }
      console.error('[LLM] 发送聊天消息失败:', error);
      throw error;
    }
  },

  // 获取对话历史
  async getChatHistory() {
    try {
      const endpoint = `${this.config.baseURL}/plugins/LLMChat/api/history`;

      
      const response = await this.fetchWithTimeout(endpoint, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include' // 包含cookie以确保会话一致
      }, 60000); // 历史记录请求使用1分钟超时
      
      if (!response.ok) {
        let errorDetails = '';
        try {
          const errorData = await response.json();
          errorDetails = JSON.stringify(errorData);
        } catch (e) {
          errorDetails = await response.text();
        }
        throw new Error(`获取历史记录失败: ${response.status} ${response.statusText}, 详情: ${errorDetails}`);
      }
      
      const data = await response.json();
      
      // 标准化响应格式
      if (!data.messages && data.length > 0) {
        // 如果返回的是直接的消息数组，进行转换
        return { messages: data };
      }
      
      return data;
    } catch (error) {
      console.error('[LLM] 获取对话历史失败:', error);
      // 返回空历史而不是抛出错误，以便UI可以正常初始化
      return { messages: [] };
    }
  },

  // 清空对话历史
  async clearChatHistory() {
    try {
      const endpoint = `${this.config.baseURL}/plugins/LLMChat/api/clear-history`;

      
      const response = await this.fetchWithTimeout(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include'
      }, 60000); // 清空历史请求使用1分钟超时
      
      if (!response.ok) {
        let errorDetails = '';
        try {
          const errorData = await response.json();
          errorDetails = JSON.stringify(errorData);
        } catch (e) {
          errorDetails = await response.text();
        }
        throw new Error(`清空历史记录失败: ${response.status} ${response.statusText}, 详情: ${errorDetails}`);
      }
      
      return await response.json();
    } catch (error) {
      console.error('[LLM] 清空对话历史失败:', error);
      // 即使失败也返回成功，让UI可以继续操作
      return { success: true };
    }
  },

  // 检查API连接状态
  async checkConnection() {
    try {
      const endpoint = `${this.config.baseURL}/plugins/LLMChat/api/health`;

      
      // 使用健康检查端点而不是history端点，这样更轻量
      const response = await this.fetchWithTimeout(endpoint, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include'
      }, 15000); // 健康检查使用15秒超时
      
      // 如果health端点不可用，尝试使用history端点作为备选
      if (!response.ok) {

        const historyEndpoint = `${this.config.baseURL}/plugins/LLMChat/api/history`;
        return this.checkConnectionWithFallback(historyEndpoint);
      }
      
      return response.ok;
    } catch (error) {
      console.error('[LLM] 健康检查失败，尝试使用history端点:', error);
      const historyEndpoint = `${this.config.baseURL}/plugins/LLMChat/api/history`;
      return this.checkConnectionWithFallback(historyEndpoint);
    }
  },
  
  // 备用连接检查方法
  async checkConnectionWithFallback(endpoint) {
    try {

      const response = await this.fetchWithTimeout(endpoint, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include'
      }, 15000); // 备用连接检查也使用15秒超时
      return response.ok;
    } catch (error) {
      console.error('[LLM] 备用连接检查也失败:', error);
      return false;
    }
  }
};