// Hello插件

class HelloPlugin {
  constructor() {
    this.name = 'HelloPlugin'
    this.version = '1.0.0'
    this.description = '一个简单的Hello插件示例'
  }

  // 初始化插件
  initialize() {
    // 这里可以添加插件的初始化逻辑
  }

  // 获取插件信息
  getInfo() {
    return {
      name: this.name,
      version: this.version,
      description: this.description
    }
  }

  // 插件方法示例
  sayHello() {
    return 'Hello, Weave!'
  }

  // 渲染插件内容
  render() {
    return {
      template: `<div class="plugin-hello">
                  <h3>👋 Hello Plugin</h3>
                  <p>这是一个简单的插件示例</p>
                  <p>当前版本: ${this.version}</p>
                </div>`,
      css: `.plugin-hello {
              padding: 1rem;
              border-radius: 8px;
              background-color: #f0f4f8;
              border: 1px solid #ddd;
            }
            .plugin-hello h3 {
              margin-top: 0;
              color: #333;
            }`
    }
  }

  // 销毁插件
  destroy() {
    // 这里可以添加插件的清理逻辑
  }
}

export default HelloPlugin