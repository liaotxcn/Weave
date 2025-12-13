import { createApp } from 'vue'
import App from './App.vue'
import './styles/style.css'
import './styles/shared.css'
import './styles/patterns.css'
import './styles/animations.css'
import axios from 'axios'

// 引入Element Plus
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

// 引入Element Plus图标
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

// 导入插件
import FormatConverterPlugin from './plugins/FormatConverterPlugin.js'
import pluginManager from './pluginManager.js'

const app = createApp(App)
app.config.globalProperties.$axios = axios

// 注册插件
pluginManager.registerPlugin('FormatConverterPlugin', new FormatConverterPlugin())

// 使用Element Plus
app.use(ElementPlus)

// 注册所有Element Plus图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.mount('#app')
