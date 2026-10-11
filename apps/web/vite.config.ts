import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import { VitePWA } from 'vite-plugin-pwa'
import { defineConfig } from 'vitest/config'

// API dan WebSocket diteruskan ke backend Go, baik saat dev maupun `vite preview` (dipakai uji PWA)
const proxy = {
  '/api': { target: 'http://localhost:8080', changeOrigin: true },
  '/ws': { target: 'ws://localhost:8080', ws: true },
}

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      // "prompt": pembaruan menunggu sampai semua tab/aplikasi ditutup. Service worker baru TIDAK mengambil alih
      // saat pertandingan berlangsung (tidak ada reload tiba-tiba di tablet juri).
      registerType: 'prompt',
      injectRegister: false, // didaftarkan manual di src/main.ts (hanya build produksi)
      includeAssets: ['favicon.svg', 'icons/apple-touch-icon.png'],
      manifest: {
        id: '/',
        name: 'Digital Scoring Silat',
        short_name: 'DS Silat',
        description:
          'Penilaian pertandingan Pencak Silat secara digital: layar juri, konsol wasit, dan scoreboard.',
        lang: 'id',
        start_url: '/judge/login',
        scope: '/',
        display: 'standalone',
        orientation: 'any',
        background_color: '#020617',
        theme_color: '#0b1220',
        categories: ['sports', 'utilities'],
        icons: [
          { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
          { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
          {
            src: '/icons/maskable-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
        shortcuts: [
          { name: 'Masuk Juri / Operator Meja (PIN)', short_name: 'Juri', url: '/judge/login' },
          { name: 'Konsol Meja', short_name: 'Meja', url: '/referee' },
          { name: 'Scoreboard', short_name: 'Skor', url: '/scoreboard' },
        ],
      },
      workbox: {
        // cangkang aplikasi (HTML/JS/CSS/ikon) tersedia offline; DATA pertandingan tidak pernah dari cache
        globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//, /^\/ws\//],
        cleanupOutdatedCaches: true,
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
  server: { host: '0.0.0.0', port: 5173, proxy },
  preview: { host: '0.0.0.0', port: 4173, proxy },
})
