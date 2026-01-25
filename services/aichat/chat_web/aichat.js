// 配置
const config = {
    apiBaseURL: 'http://localhost:8080',
    userId: 'default_user',
    streaming: true,
    maxImagesPerRequest: 5
};

// DOM元素
const chatMessages = document.getElementById('chatMessages');
const messageInput = document.getElementById('messageInput');
const sendButton = document.getElementById('sendButton');
const clearButton = document.getElementById('clearButton');
const connectionStatus = document.getElementById('connectionStatus');
const pauseButton = document.getElementById('pauseButton');
const continueButton = document.getElementById('continueButton');
const stopButton = document.getElementById('stopButton');
const imageInput = document.getElementById('imageInput');
const selectedImagesContainer = document.getElementById('selectedImages');
const historyModal = document.getElementById('historyModal');
const historyMessages = document.getElementById('historyMessages');

// 状态
let isLoading = false;
let isConnected = false;
let isStreaming = false;
let selectedImages = [];

// 初始化
document.addEventListener('DOMContentLoaded', async () => {
    await checkConnection();
    await loadHistory();
});

// 检查连接状态
async function checkConnection() {
    try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 5000);
        
        const response = await fetch(`${config.apiBaseURL}/health`, {
            method: 'GET',
            signal: controller.signal
        });
        
        clearTimeout(timeoutId);
        
        isConnected = response.ok;
        updateConnectionStatus();
        return isConnected;
    } catch (error) {
        console.error('连接检查失败:', error.message);
        isConnected = false;
        updateConnectionStatus();
        return false;
    }
}

// 更新连接状态显示
function updateConnectionStatus() {
    connectionStatus.className = `connection-status ${isConnected ? 'connected' : 'disconnected'}`;
    connectionStatus.textContent = isConnected ? '已连接' : '连接失败';
}

// 手动重试连接
async function retryConnection() {
    connectionStatus.textContent = '正在重试...';
    await checkConnection();
}

// 加载聊天历史到主窗口
async function loadHistory() {
    if (!isConnected && !await checkConnection()) {
        return;
    }

    try {
        const response = await fetch(`${config.apiBaseURL}/api/chat/history?user_id=${config.userId}`);
        if (!response.ok) throw new Error('加载历史记录失败');
        
        const data = await response.json();
        if (data.messages && data.messages.length > 0) {
            chatMessages.innerHTML = '';
            data.messages.forEach(msg => {
                addMessage(msg.Role.toLowerCase(), msg.Content);
            });
        }
    } catch (error) {
        console.error('加载历史记录失败:', error);
        addMessage('assistant', '加载聊天历史失败，请稍后再试');
    }
}

// 显示历史记录
function showHistory() {
    document.getElementById('historyModal').style.display = 'block';
    loadHistoryToModal();
}

// 隐藏历史记录
function hideHistory() {
    document.getElementById('historyModal').style.display = 'none';
}

// 加载历史记录到弹窗
async function loadHistoryToModal() {
    const historyMessages = document.getElementById('historyMessages');
    historyMessages.innerHTML = '<div style="text-align: center; padding: 20px;">加载中...</div>';

    try {
        const response = await fetch(`${config.apiBaseURL}/api/chat/history?user_id=${config.userId}`);
        if (!response.ok) {
            throw new Error('获取历史记录失败');
        }

        const data = await response.json();
        
        if (data.messages && data.messages.length > 0) {
            historyMessages.innerHTML = '';
            
            // 按对话分组
            const conversations = groupMessagesByConversation(data.messages);
            
            conversations.forEach((messages, index) => {
                const conversationElement = document.createElement('div');
                conversationElement.className = 'history-message';
                
                const header = document.createElement('div');
                header.className = 'history-message-header';
                header.innerHTML = `<span>对话 ${conversations.length - index}</span><span>${new Date().toLocaleDateString()}</span>`;
                
                const content = document.createElement('div');
                content.className = 'history-message-content';
                
                messages.forEach((msg, msgIndex) => {
                    const msgElement = document.createElement('div');
                    msgElement.style.marginBottom = '8px';
                    msgElement.innerHTML = `<strong>${msg.role === 'user' ? '你' : 'AI'}:</strong> ${msg.content || '(图片消息)'}`;
                    content.appendChild(msgElement);
                });
                
                conversationElement.appendChild(header);
                conversationElement.appendChild(content);
                historyMessages.appendChild(conversationElement);
            });
        } else {
            historyMessages.innerHTML = '<div style="text-align: center; padding: 20px; color: var(--text-secondary);">暂无历史记录</div>';
        }
    } catch (error) {
        console.error('加载历史记录失败:', error);
        historyMessages.innerHTML = '<div style="text-align: center; padding: 20px; color: var(--danger-color);">加载失败，请重试</div>';
    }
}

// 按对话分组消息
function groupMessagesByConversation(messages) {
    const conversations = [];
    let currentConversation = [];
    
    messages.forEach(msg => {
        if (msg.role === 'user' && currentConversation.length > 0) {
            conversations.unshift([...currentConversation]);
            currentConversation = [];
        }
        currentConversation.push(msg);
    });
    
    if (currentConversation.length > 0) {
        conversations.unshift(currentConversation);
    }
    
    return conversations;
}

// 点击弹窗外部关闭
window.onclick = function(event) {
    const modal = document.getElementById('historyModal');
    if (event.target === modal) {
        hideHistory();
    }
}

// 发送消息
async function sendMessage() {
    const message = messageInput.value.trim();
    
    if (!message && selectedImages.length === 0) return;
    if (isLoading) return;

    if (selectedImages.length > config.maxImagesPerRequest) {
        alert(`图片数量超过限制，最多允许 ${config.maxImagesPerRequest} 张图片`);
        return;
    }

    if (!isConnected && !await checkConnection()) {
        alert('无法连接到服务器，请稍后再试');
        return;
    }

    addMessage('user', message, selectedImages);
    messageInput.value = '';
    messageInput.style.height = 'auto';
    scrollToBottom();

    isLoading = true;
    sendButton.disabled = true;

    try {
        if (config.streaming) {
            await sendStreamMessage(message);
        } else {
            await sendNonStreamMessage(message);
        }
    } catch (error) {
        console.error('发送消息失败:', error);
        addMessage('assistant', '抱歉，处理请求时出现错误，请稍后再试');
    } finally {
        isLoading = false;
        isStreaming = false;
        sendButton.disabled = false;
        updateControlButtons();
        clearSelectedImages();
    }
}

// 处理图片选择
function handleImageSelection(event) {
    const files = event.target.files;
    if (files.length === 0) return;

    if (selectedImages.length + files.length > config.maxImagesPerRequest) {
        alert(`图片数量超过限制，最多允许 ${config.maxImagesPerRequest} 张图片`);
        event.target.value = '';
        return;
    }

    for (let i = 0; i < files.length; i++) {
        const file = files[i];
        if (!file.type.startsWith('image/')) continue;

        const reader = new FileReader();
        reader.onload = function(e) {
            const imageData = {
                name: file.name,
                data: e.target.result,
                type: file.type
            };
            selectedImages.push(imageData);
            addImagePreview(imageData);
        };
        reader.readAsDataURL(file);
    }

    event.target.value = '';
}

// 添加图片预览
function addImagePreview(imageData) {
    const imageContainer = document.createElement('div');
    imageContainer.className = 'selected-image';
    imageContainer.dataset.name = imageData.name;

    const img = document.createElement('img');
    img.src = imageData.data;
    img.alt = imageData.name;

    const removeButton = document.createElement('button');
    removeButton.className = 'remove-image';
    removeButton.textContent = '×';
    removeButton.onclick = function() {
        removeImage(imageData.name);
        imageContainer.remove();
    };

    imageContainer.appendChild(img);
    imageContainer.appendChild(removeButton);
    selectedImagesContainer.appendChild(imageContainer);
}

// 移除图片
function removeImage(name) {
    selectedImages = selectedImages.filter(img => img.name !== name);
}

// 清空选中的图片
function clearSelectedImages() {
    selectedImages = [];
    selectedImagesContainer.innerHTML = '';
}

// 显示历史记录
function showHistory() {
    document.getElementById('historyModal').style.display = 'block';
    loadHistoryToModal();
}

// 隐藏历史记录
function hideHistory() {
    document.getElementById('historyModal').style.display = 'none';
}

// 加载历史记录
async function loadHistory() {
    const historyMessages = document.getElementById('historyMessages');
    historyMessages.innerHTML = '<div style="text-align: center; padding: 20px;">加载中...</div>';

    try {
        const response = await fetch(`${config.apiBaseURL}/api/chat/history?user_id=${config.userId}`);
        if (!response.ok) {
            throw new Error('获取历史记录失败');
        }

        const data = await response.json();
        
        if (data.messages && data.messages.length > 0) {
            historyMessages.innerHTML = '';
            
            // 按对话分组
            const conversations = groupMessagesByConversation(data.messages);
            
            conversations.forEach((messages, index) => {
                const conversationElement = document.createElement('div');
                conversationElement.className = 'history-message';
                
                const header = document.createElement('div');
                header.className = 'history-message-header';
                header.innerHTML = `<span>对话 ${conversations.length - index}</span><span>${new Date().toLocaleDateString()}</span>`;
                
                const content = document.createElement('div');
                content.className = 'history-message-content';
                
                messages.forEach((msg, msgIndex) => {
                    const msgElement = document.createElement('div');
                    msgElement.style.marginBottom = '8px';
                    msgElement.innerHTML = `<strong>${msg.role === 'user' ? '你' : 'AI'}:</strong> ${msg.content || '(图片消息)'}`;
                    content.appendChild(msgElement);
                });
                
                conversationElement.appendChild(header);
                conversationElement.appendChild(content);
                historyMessages.appendChild(conversationElement);
            });
        } else {
            historyMessages.innerHTML = '<div style="text-align: center; padding: 20px; color: var(--text-secondary);">暂无历史记录</div>';
        }
    } catch (error) {
        console.error('加载历史记录失败:', error);
        historyMessages.innerHTML = '<div style="text-align: center; padding: 20px; color: var(--danger-color);">加载失败，请重试</div>';
    }
}

// 按对话分组消息
function groupMessagesByConversation(messages) {
    const conversations = [];
    let currentConversation = [];
    
    messages.forEach(msg => {
        if (msg.role === 'user' && currentConversation.length > 0) {
            conversations.unshift([...currentConversation]);
            currentConversation = [];
        }
        currentConversation.push(msg);
    });
    
    if (currentConversation.length > 0) {
        conversations.unshift(currentConversation);
    }
    
    return conversations;
}

// 点击弹窗外部关闭
window.onclick = function(event) {
    const modal = document.getElementById('historyModal');
    if (event.target === modal) {
        hideHistory();
    }
}

// 将图片转换为Base64
function convertToBase64(imageData) {
    const base64Prefix = ';base64,';
    const base64Index = imageData.indexOf(base64Prefix);
    if (base64Index === -1) return '';
    let base64Str = imageData.substring(base64Index + base64Prefix.length);
    base64Str = base64Str.replace(/\s/g, '');
    while (base64Str.length % 4 !== 0) {
        base64Str += '=';
    }
    return base64Str;
}

// 发送非流式消息
async function sendNonStreamMessage(message) {
    const base64Images = selectedImages.map(img => convertToBase64(img.data));

    const response = await fetch(`${config.apiBaseURL}/api/chat`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            user_input: message,
            user_id: config.userId,
            base64_images: base64Images,
            image_urls: []
        })
    });

    if (!response.ok) {
        try {
            const errorData = await response.json();
            throw new Error(errorData.error || 'API请求失败');
        } catch (e) {
            throw new Error('API请求失败');
        }
    }
    
    const data = await response.json();
    addMessage('assistant', data.content);
}

// 发送流式消息
async function sendStreamMessage(message) {
    const loadingMessage = addMessage('assistant', '', true);
    let fullContent = '';

    isStreaming = true;
    updateControlButtons();

    const base64Images = selectedImages.map(img => convertToBase64(img.data));

    const response = await fetch(`${config.apiBaseURL}/api/chat/stream`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            user_input: message,
            user_id: config.userId,
            base64_images: base64Images,
            image_urls: []
        })
    });

    if (!response.ok) {
        try {
            const errorData = await response.json();
            throw new Error(errorData.error || 'API请求失败');
        } catch (e) {
            throw new Error('API请求失败');
        }
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    try {
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            const chunk = decoder.decode(value, { stream: true });
            const lines = chunk.split('\n');

            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const data = line.slice(6).trim();
                    if (data) {
                        try {
                            const parsed = JSON.parse(data);
                            if (parsed.error) {
                                updateMessage(loadingMessage, `错误: ${parsed.error}`);
                                scrollToBottom();
                                return;
                            }
                            if (parsed.content && parsed.status !== 'completed') {
                                fullContent += parsed.content;
                                updateMessage(loadingMessage, fullContent);
                                scrollToBottom();
                            }

                            if (parsed.status === 'completed') {
                                return;
                            }
                        } catch (e) {
                            console.error('解析流式响应失败:', e);
                        }
                    }
                }
            }
        }
    } finally {
        isStreaming = false;
        updateControlButtons();
        
        if (loadingMessage) {
            loadingMessage.classList.remove('loading');
        }
    }
}

// 控制聊天流
async function controlChat(action) {
    if (!isConnected || !isStreaming) return;

    try {
        const response = await fetch(`${config.apiBaseURL}/api/chat/control`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: config.userId,
                action: action
            })
        });

        if (!response.ok) {
            try {
                const errorData = await response.json();
                throw new Error(errorData.error || '控制请求失败');
            } catch (e) {
                throw new Error('控制请求失败');
            }
        }
    } catch (error) {
        console.error('控制操作失败:', error);
        alert(`控制操作失败: ${error.message}`);
    }
}

// 更新控制按钮状态
function updateControlButtons() {
    pauseButton.disabled = !isStreaming;
    continueButton.disabled = !isStreaming;
    stopButton.disabled = !isStreaming;
}

// 添加消息
function addMessage(role, content, images = []) {
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${role}`;

    const avatarDiv = document.createElement('div');
    avatarDiv.className = 'message-avatar';
    if (role === 'user') {
        avatarDiv.textContent = '👤';
    } else {
        avatarDiv.innerHTML = '<svg t="1768989011593" class="icon" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" p-id="33830" width="20" height="20"><path d="M170.666667 0h682.666666a170.666667 170.666667 0 0 1 170.666667 170.666667v682.666666a170.666667 170.666667 0 0 1-170.666667 170.666667H170.666667a170.666667 170.666667 0 0 1-170.666667-170.666667V170.666667A170.666667 170.666667 0 0 1 162.133333 0.213333L170.666667 0z" fill="#0075C2" p-id="33831"></path><path d="M409.429333 573.312a499.498667 499.498667 0 0 1 297.386667-302.592l-36.565333-21.034667-126.293334-72.064a52.565333 52.565333 0 0 0-52.309333 0L365.482667 249.685333 239.530667 321.92A52.48 52.48 0 0 0 213.333333 367.786667v201.088a409.6 409.6 0 0 1 196.266667 4.437333h-0.170667z" fill="#FFFFFF" p-id="33832"></path><path d="M239.530667 701.696l125.952 72.277333 18.261333 10.752a497.365333 497.365333 0 0 1 21.76-199.253333A395.861333 395.861333 0 0 0 213.333333 581.632v74.496a52.437333 52.437333 0 0 0 26.197334 45.568zM433.877333 580.778667c38.784 13.141333 75.392 32 108.586667 55.978666a47.36 47.36 0 0 1 68.608 62.506667c20.778667 23.893333 38.698667 50.090667 53.418667 78.122667l5.632-3.2 125.952-72.277334a52.48 52.48 0 0 0 26.410666-45.568V367.786667a52.522667 52.522667 0 0 0-26.453333-45.482667l-59.818667-34.986667a473.472 473.472 0 0 0-302.336 293.461334z" fill="#FFFFFF" p-id="33833"></path><path d="M570.197333 722.346667a47.402667 47.402667 0 0 1-37.12-76.629334 396.8 396.8 0 0 0-103.253333-53.162666 473.045333 473.045333 0 0 0-18.730667 207.573333l80.469334 46.250667a52.565333 52.565333 0 0 0 52.309333 0l109.696-62.976a394.88 394.88 0 0 0-50.773333-74.197334c-8.789333 8.362667-20.48 13.056-32.597334 13.098667z" fill="#FFFFFF" p-id="33834"></path></svg>';
    }

    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';
    
    if (content) {
        const textDiv = document.createElement('div');
        textDiv.textContent = content;
        contentDiv.appendChild(textDiv);
    }

    if (images && images.length > 0) {
        const imagesContainer = document.createElement('div');
        imagesContainer.style.cssText = 'display:flex;flex-direction:column;gap:8px;margin-top:8px;';

        images.forEach(imageData => {
            const img = document.createElement('img');
            img.src = imageData.data;
            img.alt = imageData.name;
            img.style.cssText = 'max-width:200px;border-radius:8px;object-fit:cover;';
            imagesContainer.appendChild(img);
        });

        contentDiv.appendChild(imagesContainer);
    }

    messageDiv.appendChild(avatarDiv);
    messageDiv.appendChild(contentDiv);
    chatMessages.appendChild(messageDiv);

    scrollToBottom();
    return messageDiv;
}

// 更新消息内容
function updateMessage(messageElement, content) {
    const contentElement = messageElement.querySelector('.message-content');
    if (contentElement) {
        contentElement.textContent = content;
    }
}

// 滚动到底部
function scrollToBottom() {
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// 清除聊天历史
async function clearHistory() {
    if (!confirm('确定要清除所有聊天历史吗？')) return;

    try {
        if (isStreaming) {
            await controlChat('stop');
        }

        const response = await fetch(`${config.apiBaseURL}/api/chat/history?user_id=${config.userId}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('清除历史记录失败');

        // 清空主聊天界面
        chatMessages.innerHTML = '';
        addMessage('assistant', '聊天历史已清除，有什么可以帮助你的吗？');
        clearSelectedImages();

        // 更新历史记录弹窗内容
        const historyMessages = document.getElementById('historyMessages');
        historyMessages.innerHTML = '<div style="text-align: center; padding: 20px; color: var(--text-secondary);">暂无历史记录</div>';
    } catch (error) {
        console.error('清除历史记录失败:', error);
        alert('清除历史记录失败，请稍后再试');
    } finally {
        isStreaming = false;
        updateControlButtons();
    }
}

// 显示历史记录
function showHistory() {
    document.getElementById('historyModal').style.display = 'block';
    loadHistory();
}

// 隐藏历史记录
function hideHistory() {
    document.getElementById('historyModal').style.display = 'none';
}

// 加载历史记录
async function loadHistory() {
    const historyMessages = document.getElementById('historyMessages');
    historyMessages.innerHTML = '<div style="text-align: center; padding: 20px;">加载中...</div>';

    try {
        const response = await fetch(`${config.apiBaseURL}/api/chat/history?user_id=${config.userId}`);
        if (!response.ok) {
            throw new Error('获取历史记录失败');
        }

        const data = await response.json();
        
        if (data.messages && data.messages.length > 0) {
            historyMessages.innerHTML = '';
            
            // 按对话分组
            const conversations = groupMessagesByConversation(data.messages);
            
            conversations.forEach((messages, index) => {
                const conversationElement = document.createElement('div');
                conversationElement.className = 'history-message';
                
                const header = document.createElement('div');
                header.className = 'history-message-header';
                header.innerHTML = `<span>对话 ${conversations.length - index}</span><span>${new Date().toLocaleDateString()}</span>`;
                
                const content = document.createElement('div');
                content.className = 'history-message-content';
                
                messages.forEach((msg, msgIndex) => {
                    const msgElement = document.createElement('div');
                    msgElement.style.marginBottom = '8px';
                    msgElement.innerHTML = `<strong>${msg.role === 'user' ? '你' : 'AI'}:</strong> ${msg.content || '(图片消息)'}`;
                    content.appendChild(msgElement);
                });
                
                conversationElement.appendChild(header);
                conversationElement.appendChild(content);
                historyMessages.appendChild(conversationElement);
            });
        } else {
            historyMessages.innerHTML = '<div style="text-align: center; padding: 20px; color: var(--text-secondary);">暂无历史记录</div>';
        }
    } catch (error) {
        console.error('加载历史记录失败:', error);
        historyMessages.innerHTML = '<div style="text-align: center; padding: 20px; color: var(--danger-color);">加载失败，请重试</div>';
    }
}

// 按对话分组消息
function groupMessagesByConversation(messages) {
    const conversations = [];
    let currentConversation = [];
    
    messages.forEach(msg => {
        if (msg.role === 'user' && currentConversation.length > 0) {
            conversations.unshift([...currentConversation]);
            currentConversation = [];
        }
        currentConversation.push(msg);
    });
    
    if (currentConversation.length > 0) {
        conversations.unshift(currentConversation);
    }
    
    return conversations;
}

// 点击弹窗外部关闭
window.onclick = function(event) {
    const modal = document.getElementById('historyModal');
    if (event.target === modal) {
        hideHistory();
    }
}

// 自动调整输入框高度
messageInput.addEventListener('input', () => {
    messageInput.style.height = 'auto';
    messageInput.style.height = Math.min(messageInput.scrollHeight, 120) + 'px';
});