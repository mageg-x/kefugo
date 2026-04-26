import './style.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import TDesign from 'tdesign-vue-next'
import 'tdesign-vue-next/es/style/index.css'
import Chat from '@tdesign-vue-next/chat'
import '@tdesign-vue-next/chat/es/style/index.css'
import App from './App.vue'
import router from './router/router'
import { initRuntimeI18n } from './script/i18n'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(ElementPlus)
app.use(TDesign)
app.use(Chat)
app.use(router)
initRuntimeI18n()
app.mount('#app')
