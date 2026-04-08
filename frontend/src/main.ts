import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'

// Inject the web app manifest synchronously, before the browser fetches anything,
// so iOS "Add to Home Screen" always gets the correct start_url.
;(() => {
  const checkinMatch = window.location.pathname.match(/^\/checkin\/([^/]+)$/)
  const link = document.createElement('link')
  link.rel = 'manifest'
  link.href = checkinMatch
    ? `/api/parent/${checkinMatch[1]}/manifest.json`
    : '/manifest.json'
  document.head.appendChild(link)
})()

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
