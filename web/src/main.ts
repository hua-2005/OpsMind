// Vue 应用入口：创建实例、注册 Pinia/Router/ElementPlus、挂载
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'

// Linear Design 暗色主题全局样式
import './styles/global.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
