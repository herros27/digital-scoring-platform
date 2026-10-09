import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { queryClient } from './lib/queryClient'
import App from './App.vue'
import router from './router'
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
