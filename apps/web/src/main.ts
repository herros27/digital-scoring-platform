import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { queryClient } from './lib/queryClient'
import App from './App.vue'
import router from './router'
// Font di-host sendiri: tampilan sama dengan atau tanpa internet (acara bisa berlangsung di jaringan lokal)
import '@fontsource/space-grotesk/latin-300.css'
import '@fontsource/space-grotesk/latin-400.css'
import '@fontsource/space-grotesk/latin-500.css'
import '@fontsource/space-grotesk/latin-600.css'
import '@fontsource/space-grotesk/latin-700.css'
import '@fontsource/jetbrains-mono/latin-300.css'
import '@fontsource/jetbrains-mono/latin-400.css'
import '@fontsource/jetbrains-mono/latin-500.css'
import '@fontsource/jetbrains-mono/latin-600.css'
import '@fontsource/jetbrains-mono/latin-700.css'
import '@fontsource/jetbrains-mono/latin-800.css'
import '@fontsource/jetbrains-mono/latin-400-italic.css'
import './style.css'

const app = createApp(App)

app.use(createPinia())
app.use(VueQueryPlugin, { queryClient })
app.use(router)

app.mount('#app')

// PWA: cangkang aplikasi tersedia offline dan dapat dipasang ke layar utama tablet. Hanya pada build produksi dan
// butuh konteks aman (HTTPS atau localhost). Versi baru menunggu sampai aplikasi ditutup, tidak memotong pertandingan.
if (import.meta.env.PROD) {
  void import('virtual:pwa-register').then(({ registerSW }) => registerSW({ immediate: true }))
}
