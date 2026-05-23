import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { registerDqIcons } from '@danqing/dq-shell'
import App from './App.vue'
import { installDanQingUi } from './plugins/dq-ui'
import '@danqing/dq-tokens/dq-mac.css'
import '@danqing/dq-ui/style.css'
import '@danqing/dq-shell/style.css'
import './styles/theme.css'
import './styles/theme-apple-dark.css'
import './styles/theme-apple-chrome.css'
import './styles/theme-apple-finish.css'
import './styles/theme-apple-native.css'
import './styles/teams.css'
import './styles/theme-apple-teams.css'

document.documentElement.classList.add('dark', 'dq-mac-ui')

const app = createApp(App)
registerDqIcons(app)
app.use(createPinia())
installDanQingUi(app)
app.mount('#app')
