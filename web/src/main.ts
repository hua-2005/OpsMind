/**
 * Vue 应用入口
 *
 * 创建 Vue app，注册 Pinia、Router、全局样式。
 */

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/global.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')
