import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createHead } from '@unhead/vue/client'
import './style.css'
import App from './App.vue'
import router, { warmupCommonRoutes } from './router'
import i18n from './i18n'
import { useTelegramMiniAppStore } from './stores/telegramMiniApp'

console.log(
  '%c Dujiao-Next %c Digital Commerce Platform %c',
  'background:#0071e3;color:#fff;padding:4px 8px;border-radius:4px 0 0 4px;font-weight:bold;',
  'background:#1d1d1f;color:#f5f5f7;padding:4px 8px;border-radius:0 4px 4px 0;',
  'background:transparent;',
)
console.log(
  '%cGitHub ? https://github.com/dujiao-next',
  'color:#6e6e73;',
)

const app = createApp(App)
const head = createHead()
const pinia = createPinia()

app.use(pinia)
app.use(head)
app.use(router)
app.use(i18n)

useTelegramMiniAppStore(pinia).init().then(() => {
  app.mount('#app')
})

void router.isReady().then(() => {
    warmupCommonRoutes()
})
