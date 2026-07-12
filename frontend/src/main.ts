import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { registerDqIcons } from '@danqing/dq-shell'
import App from './App.vue'
import { router } from './router'
import { installDanQingUi } from './plugins/dq-ui'
import { i18n } from './i18n'
import '@danqing/dq-tokens/dq-mac.css'
import '@danqing/dq-tokens/dq-glass.css'
import '@danqing/dq-ui/style.css'
import '@danqing/dq-shell/style.css'
import '@/styles/teams.css'

document.documentElement.classList.add('dark', 'dq-mac-ui')

function showFatalError(err: unknown) {
  const container = document.getElementById('app')
  const msg = err instanceof Error ? err.message : String(err)
  const stack = err instanceof Error ? err.stack : ''
  if (container) {
    container.innerHTML = `<div style="padding:20px;font-family:ui-monospace,monospace;color:#c00;background:#fff;white-space:pre-wrap;word-break:break-word;"><strong>Fatal Error:</strong> ${msg}\n${stack}</div>`
  }
  // eslint-disable-next-line no-console
  console.error('[FATAL]', err)
}

window.addEventListener('error', (e) => showFatalError(e.error))
window.addEventListener('unhandledrejection', (e) => showFatalError(e.reason))

const app = createApp(App)
registerDqIcons(app)
app.use(createPinia())
app.use(router)
app.use(i18n)
installDanQingUi(app)
app.mount('#app')

app.config.errorHandler = (err, vm, info) => {
  // eslint-disable-next-line no-console
  console.error('[Vue Error]', err, info, vm)
  showFatalError(err)
}
