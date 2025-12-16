// FormatConverter 插件

import { formatConverterService } from '../services/formatConverter.js'

class FormatConverterPlugin {
  constructor() {
    this.name = 'FormatConverterPlugin'
    this.version = '1.0.0'
    this.description = '格式转换工具：支持JSON、YAML和Protobuf格式之间的相互转换'
    this.conversionHistory = []
    this.maxHistoryItems = 10
  }

  // 初始化插件
  async initialize() {

    try {
      // 获取插件信息
      const info = await formatConverterService.getPluginInfo()

    } catch (error) {
      console.error('格式转换器插件初始化失败:', error)
    }
  }

  // 渲染插件界面
  render() {
    // 返回符合Vue组件要求的对象，包含template和mounted钩子函数
    return {
      template: `
<div class="format-converter-plugin">
  <!-- 插件标题和描述 -->
  <div class="plugin-header">
    <h2 class="plugin-title">格式转换器</h2>
    <p class="plugin-desc">支持JSON、YAML和Protobuf格式之间的相互转换</p>
  </div>

  <!-- 转换控制区域 -->
  <div class="conversion-controls">
    <div class="conversion-selector">
      <label for="conversion-type">转换类型:</label>
      <select id="conversion-type" class="conversion-dropdown">
        <option value="json-to-yaml">JSON → YAML</option>
        <option value="yaml-to-json">YAML → JSON</option>
        <option value="json-to-protobuf">JSON → Protobuf (Base64)</option>
        <option value="protobuf-to-json">Protobuf (Base64) → JSON</option>
      </select>
      <button id="example-data-btn" class="btn-secondary">示例数据</button>
    </div>
    
    <button id="convert-btn" class="btn-primary">执行转换</button>
  </div>

  <!-- 编辑器和结果区域 -->
  <div class="editor-container">
    <div class="editor-section">
      <div class="editor-header">
        <h3>输入</h3>
        <button id="clear-input-btn" class="btn-icon" title="清空输入">🗑️</button>
      </div>
      <div class="validation-result" id="input-validation"></div>
      <textarea id="input-textarea" placeholder="请输入要转换的数据..." rows="15"></textarea>
    </div>
    
    <div class="swap-arrow">⇄</div>
    
    <div class="editor-section">
      <div class="editor-header">
        <h3>输出</h3>
        <button id="copy-output-btn" class="btn-icon" title="复制结果">📋</button>
      </div>
      <div class="output-status" id="output-status"></div>
      <textarea id="output-textarea" placeholder="转换结果将显示在这里..." rows="15" readonly></textarea>
    </div>
  </div>

  <!-- 历史记录区域 -->
  <div class="history-section">
    <div class="history-header">
      <h3>转换历史</h3>
      <button id="clear-history-btn" class="btn-icon" title="清空历史">🗑️</button>
    </div>
    <div id="history-list" class="history-list">
      <div class="no-history">暂无转换历史</div>
    </div>
  </div>

  <!-- 帮助提示 -->
  <div class="help-section">
    <h4>使用说明</h4>
    <ul>
      <li>选择要执行的转换类型</li>
      <li>在输入框中输入相应格式的数据</li>
      <li>点击「执行转换」按钮开始转换</li>
      <li>点击「示例数据」按钮可以加载示例内容</li>
      <li>Protobuf数据将以Base64编码的形式显示和输入</li>
    </ul>
  </div>

  <style scoped>
  .format-converter-plugin {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
    background-color: #fafafa;
    border-radius: 8px;
  }

  .plugin-header {
    text-align: center;
    margin-bottom: 24px;
    padding-bottom: 16px;
    border-bottom: 1px solid #e8e8e8;
  }

  .plugin-title {
    margin: 0 0 8px 0;
    color: #262626;
    font-size: 24px;
    font-weight: 600;
  }

  .plugin-desc {
    margin: 0;
    color: #595959;
    font-size: 14px;
  }

  .conversion-controls {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    padding: 16px;
    background-color: #ffffff;
    border-radius: 6px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  }

  .conversion-selector {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .conversion-selector label {
    font-weight: 500;
    color: #262626;
    white-space: nowrap;
  }

  .conversion-dropdown {
    padding: 8px 12px;
    border: 1px solid #d9d9d9;
    border-radius: 4px;
    background-color: #ffffff;
    font-size: 14px;
    min-width: 200px;
  }

  .btn-primary, .btn-secondary, .btn-icon {
    padding: 8px 16px;
    border: none;
    border-radius: 4px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-primary {
    background-color: #1890ff;
    color: #ffffff;
  }

  .btn-primary:hover {
    background-color: #40a9ff;
  }

  .btn-secondary {
    background-color: #ffffff;
    color: #262626;
    border: 1px solid #d9d9d9;
  }

  .btn-secondary:hover {
    color: #1890ff;
    border-color: #1890ff;
  }

  .btn-icon {
    padding: 4px;
    background: transparent;
    color: #595959;
    font-size: 16px;
  }

  .btn-icon:hover {
    color: #1890ff;
  }

  .editor-container {
    display: flex;
    gap: 20px;
    margin-bottom: 24px;
  }

  .editor-section {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .editor-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .editor-header h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 500;
    color: #262626;
  }

  .validation-result, .output-status {
    font-size: 12px;
    margin-bottom: 8px;
    min-height: 16px;
  }

  .validation-result.error {
    color: #ff4d4f;
  }

  .validation-result.success {
    color: #52c41a;
  }

  .output-status {
    color: #1890ff;
  }

  textarea {
    flex: 1;
    padding: 12px;
    border: 1px solid #d9d9d9;
    border-radius: 4px;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.5;
    resize: vertical;
    background-color: #ffffff;
  }

  textarea:focus {
    outline: none;
    border-color: #1890ff;
    box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
  }

  textarea[readonly] {
    background-color: #f5f5f5;
  }

  .swap-arrow {
    align-self: center;
    font-size: 24px;
    color: #d9d9d9;
  }

  .history-section {
    margin-bottom: 24px;
    padding: 16px;
    background-color: #ffffff;
    border-radius: 6px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  }

  .history-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .history-header h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 500;
    color: #262626;
  }

  .history-list {
    max-height: 200px;
    overflow-y: auto;
  }

  .history-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px;
    margin-bottom: 8px;
    background-color: #fafafa;
    border-radius: 4px;
    border-left: 3px solid #1890ff;
  }

  .history-item-header {
    flex: 1;
  }

  .history-type {
    display: block;
    font-weight: 500;
    color: #262626;
    margin-bottom: 4px;
  }

  .history-time {
    font-size: 12px;
    color: #8c8c8c;
  }

  .history-item-actions {
    margin-left: 16px;
  }

  .btn-small {
    padding: 4px 12px;
    border: none;
    border-radius: 4px;
    background-color: #f0f0f0;
    color: #262626;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-small:hover {
    background-color: #1890ff;
    color: #ffffff;
  }

  .no-history {
    text-align: center;
    color: #8c8c8c;
    padding: 40px;
    font-size: 14px;
  }

  .help-section {
    padding: 16px;
    background-color: #f0f8ff;
    border-radius: 6px;
    border: 1px solid #bae7ff;
  }

  .help-section h4 {
    margin: 0 0 12px 0;
    color: #1890ff;
    font-size: 14px;
    font-weight: 500;
  }

  .help-section ul {
    margin: 0;
    padding-left: 20px;
    color: #595959;
    font-size: 13px;
    line-height: 1.6;
  }

  .help-section li {
    margin-bottom: 4px;
  }

  @media (max-width: 768px) {
    .editor-container {
      flex-direction: column;
    }

    .swap-arrow {
      transform: rotate(90deg);
    }

    .conversion-controls {
      flex-direction: column;
      gap: 16px;
    }

    .conversion-selector {
      flex-direction: column;
      align-items: stretch;
      gap: 8px;
    }
  }
  </style>
`,
      mounted() {
        const container = this.$el;
        const plugin = this.$parent.plugin; // 获取插件实例引用
        const inputTextarea = container.querySelector('#input-textarea');
        const outputTextarea = container.querySelector('#output-textarea');
        const conversionTypeSelect = container.querySelector('#conversion-type');
        const convertBtn = container.querySelector('#convert-btn');
        const clearInputBtn = container.querySelector('#clear-input-btn');
        const copyOutputBtn = container.querySelector('#copy-output-btn');
        const clearHistoryBtn = container.querySelector('#clear-history-btn');
        const exampleDataBtn = container.querySelector('#example-data-btn');
        const historyList = container.querySelector('#history-list');
        const inputValidation = container.querySelector('#input-validation');
        const outputStatus = container.querySelector('#output-status');
        
        // 执行转换
        convertBtn.addEventListener('click', async () => {
          const input = inputTextarea.value.trim();
          if (!input) {
            inputValidation.textContent = '请输入要转换的数据';
            inputValidation.style.color = '#ff4d4f';
            return;
          }
          
          try {
            inputValidation.textContent = '';
            outputStatus.textContent = '正在转换...';
            outputStatus.style.color = '#1890ff';
            
            const conversionType = conversionTypeSelect.value;
            let result;
            
            switch (conversionType) {
              case 'json-to-yaml':
                result = await plugin.convertJsonToYaml(input);
                break;
              case 'yaml-to-json':
                result = await plugin.convertYamlToJson(input);
                break;
              case 'json-to-protobuf':
                result = await plugin.convertJsonToProtobuf(input);
                break;
              case 'protobuf-to-json':
                result = await plugin.convertProtobufToJson(input);
                break;
            }
            
            outputTextarea.value = result;
            outputStatus.textContent = '转换成功';
            outputStatus.style.color = '#52c41a';
            
            // 更新历史记录显示
            updateHistoryList();
            
          } catch (error) {
            outputStatus.textContent = `转换失败: ${error.message}`;
            outputStatus.style.color = '#ff4d4f';
            console.error('转换错误:', error);
          }
        });
        
        // 清空输入
        clearInputBtn.addEventListener('click', () => {
          inputTextarea.value = '';
          inputValidation.textContent = '';
        });
        
        // 复制输出
        copyOutputBtn.addEventListener('click', () => {
          if (outputTextarea.value) {
            navigator.clipboard.writeText(outputTextarea.value)
              .then(() => {
                const originalText = outputStatus.textContent;
                outputStatus.textContent = '已复制到剪贴板';
                outputStatus.style.color = '#52c41a';
                setTimeout(() => {
                  outputStatus.textContent = originalText;
                }, 2000);
              })
              .catch(err => {
                console.error('复制失败:', err);
              });
          }
        });
        
        // 清空历史
        clearHistoryBtn.addEventListener('click', () => {
          plugin.clearHistory();
          updateHistoryList();
        });
        
        // 加载示例数据
        exampleDataBtn.addEventListener('click', () => {
          const conversionType = conversionTypeSelect.value;
          let exampleData = '';
          
          if (conversionType.startsWith('json-to-')) {
            exampleData = plugin.getExampleData('json');
          } else if (conversionType.startsWith('yaml-to-')) {
            exampleData = plugin.getExampleData('yaml');
          }
          
          if (exampleData) {
            inputTextarea.value = exampleData;
            inputValidation.textContent = '';
          }
        });
        
        // 验证JSON输入
        inputTextarea.addEventListener('input', () => {
          const conversionType = conversionTypeSelect.value;
          if (conversionType === 'json-to-yaml' || conversionType === 'json-to-protobuf') {
            const validation = plugin.validateJson(inputTextarea.value);
            if (!validation.valid && inputTextarea.value.trim()) {
              inputValidation.textContent = `JSON格式错误: ${validation.error}`;
              inputValidation.style.color = '#ff4d4f';
            } else {
              inputValidation.textContent = validation.valid ? 'JSON格式正确' : '';
              if (validation.valid) {
                inputValidation.style.color = '#52c41a';
              }
            }
          } else {
            inputValidation.textContent = '';
          }
        });
        
        // 更新历史记录列表
        function updateHistoryList() {
          // 直接使用插件实例的conversionHistory属性
          const history = plugin.conversionHistory;
          historyList.innerHTML = '';
          
          if (history.length === 0) {
            historyList.innerHTML = '<div class="no-history">暂无转换历史</div>';
            return;
          }
          
          history.forEach((item, index) => {
            const historyItem = document.createElement('div');
            historyItem.className = 'history-item';
            
            const timestamp = new Date(item.timestamp).toLocaleString();
            const conversionType = plugin.getConversionTypeName(item.type);
            
            historyItem.innerHTML = `
              <div class="history-item-header">
                <span class="history-type">${conversionType}</span>
                <span class="history-time">${timestamp}</span>
              </div>
              <div class="history-item-actions">
                <button class="btn-small" data-index="${index}">重用</button>
              </div>
            `;
            
            // 重用历史记录的点击事件
            const reuseBtn = historyItem.querySelector('button');
            reuseBtn.addEventListener('click', (e) => {
              const idx = parseInt(e.currentTarget.dataset.index);
              const selectedItem = history[idx];
              inputTextarea.value = selectedItem.input;
              outputTextarea.value = selectedItem.output;
              
              // 根据历史记录类型设置转换类型
              conversionTypeSelect.value = selectedItem.type;
              
              inputValidation.textContent = '';
              outputStatus.textContent = '已加载历史记录';
              outputStatus.style.color = '#1890ff';
            });
            
            historyList.appendChild(historyItem);
          });
        }
        
        // 初始加载
        updateHistoryList();
      }
    };
  }

  // 获取插件信息
  getInfo() {
    return {
      name: this.name,
      version: this.version,
      description: this.description,
      historyCount: this.conversionHistory.length
    }
  }

  // JSON转YAML
  async convertJsonToYaml(jsonInput) {
    try {
      // 验证JSON格式
      const parsedJson = JSON.parse(jsonInput)
      
      // 调用服务进行转换
      const yamlResult = await formatConverterService.jsonToYaml(parsedJson)
      
      // 保存到历史记录
      this.addToHistory({
        type: 'json-to-yaml',
        input: jsonInput,
        output: yamlResult,
        timestamp: new Date()
      })
      
      return yamlResult
    } catch (error) {
      console.error('JSON转YAML失败:', error)
      throw new Error(`JSON格式无效或转换失败: ${error.message}`)
    }
  }

  // YAML转JSON
  async convertYamlToJson(yamlInput) {
    try {
      // 调用服务进行转换
      const jsonResult = await formatConverterService.yamlToJson(yamlInput)
      
      // 格式化JSON输出
      const formattedJson = JSON.stringify(jsonResult, null, 2)
      
      // 保存到历史记录
      this.addToHistory({
        type: 'yaml-to-json',
        input: yamlInput,
        output: formattedJson,
        timestamp: new Date()
      })
      
      return formattedJson
    } catch (error) {
      console.error('YAML转JSON失败:', error)
      throw new Error(`YAML格式无效或转换失败: ${error.message}`)
    }
  }

  // JSON转Protobuf（Base64编码输出）
  async convertJsonToProtobuf(jsonInput) {
    try {
      // 验证JSON格式
      const parsedJson = JSON.parse(jsonInput)
      
      // 调用服务进行转换
      const protobufResult = await formatConverterService.jsonToProtobuf(parsedJson)
      
      // 将二进制数据转换为Base64编码
      const base64Result = this.arrayBufferToBase64(protobufResult)
      
      // 保存到历史记录
      this.addToHistory({
        type: 'json-to-protobuf',
        input: jsonInput,
        output: base64Result,
        timestamp: new Date()
      })
      
      return base64Result
    } catch (error) {
      console.error('JSON转Protobuf失败:', error)
      throw new Error(`JSON格式无效或转换失败: ${error.message}`)
    }
  }

  // Protobuf转JSON（接受Base64编码输入）
  async convertProtobufToJson(protobufBase64) {
    try {
      // 将Base64转换为二进制数据
      const binaryData = this.base64ToArrayBuffer(protobufBase64)
      
      // 调用服务进行转换
      const jsonResult = await formatConverterService.protobufToJson(binaryData)
      
      // 格式化JSON输出
      const formattedJson = JSON.stringify(jsonResult, null, 2)
      
      // 保存到历史记录
      this.addToHistory({
        type: 'protobuf-to-json',
        input: protobufBase64,
        output: formattedJson,
        timestamp: new Date()
      })
      
      return formattedJson
    } catch (error) {
      console.error('Protobuf转JSON失败:', error)
      throw new Error(`Protobuf数据无效或转换失败: ${error.message}`)
    }
  }

  // 获取转换历史
  getConversionHistory() {
    return [...this.conversionHistory]
  }

  // 清空转换历史
  clearHistory() {
    this.conversionHistory = []
  }

  // 添加到历史记录
  addToHistory(item) {
    // 添加到历史记录开头
    this.conversionHistory.unshift(item)
    
    // 限制历史记录数量
    if (this.conversionHistory.length > this.maxHistoryItems) {
      this.conversionHistory = this.conversionHistory.slice(0, this.maxHistoryItems)
    }
  }

  // 辅助方法：ArrayBuffer转Base64
  arrayBufferToBase64(buffer) {
    let binary = ''
    const bytes = new Uint8Array(buffer)
    const len = bytes.byteLength
    for (let i = 0; i < len; i++) {
      binary += String.fromCharCode(bytes[i])
    }
    return window.btoa(binary)
  }

  // 辅助方法：Base64转ArrayBuffer
  base64ToArrayBuffer(base64) {
    const binaryString = window.atob(base64)
    const len = binaryString.length
    const bytes = new Uint8Array(len)
    for (let i = 0; i < len; i++) {
      bytes[i] = binaryString.charCodeAt(i)
    }
    return bytes.buffer
  }

  // 验证JSON格式
  validateJson(jsonString) {
    try {
      JSON.parse(jsonString)
      return { valid: true }
    } catch (error) {
      return { valid: false, error: error.message }
    }
  }

  // 获取转换类型的友好名称
  getConversionTypeName(type) {
    const typeNames = {
      'json-to-yaml': 'JSON → YAML',
      'yaml-to-json': 'YAML → JSON',
      'json-to-protobuf': 'JSON → Protobuf',
      'protobuf-to-json': 'Protobuf → JSON'
    }
    return typeNames[type] || type
  }

  // 生成示例数据
  getExampleData(type) {
    switch (type) {
      case 'json':
        return JSON.stringify({
          name: "示例数据",
          version: "1.0.0",
          description: "这是一个示例JSON数据",
          author: {
            name: "张三",
            email: "zhangsan@example.com"
          },
          tags: ["示例", "JSON", "格式"],
          metadata: {
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z"
          },
          active: true,
          count: 42,
          settings: {
            theme: "light",
            notifications: false
          }
        }, null, 2)
      case 'yaml':
        return `name: 示例数据
version: "1.0.0"
description: 这是一个示例YAML数据
author:
  name: 张三
  email: zhangsan@example.com
tags:
  - 示例
  - YAML
  - 格式
metadata:
  createdAt: 2024-01-01T00:00:00Z
  updatedAt: 2024-01-01T00:00:00Z
active: true
count: 42
settings:
  theme: light
  notifications: false`
      default:
        return ""
    }
  }
}

export default FormatConverterPlugin