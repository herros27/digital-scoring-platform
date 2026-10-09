import { defineConfig } from '@playwright/test'
import base from './playwright.config.ts'

/**
 * Uji PWA pada BUILD PRODUKSI (service worker hanya terdaftar di build, bukan `vite dev`): membangun lalu menyajikan
 * dengan `vite preview` di port 4173 (proxy /api dan /ws ke backend). Backend harus berjalan.
 *
 *   npm run test:pw:pwa
 */
export default defineConfig({
  ...base,
  testIgnore: undefined,
  testMatch: '**/pwa.spec.ts',
  reporter: [['list']],
  use: { ...base.use, baseURL: 'http://127.0.0.1:4173' },
  webServer: {
    command: 'npm run build && npx vite preview --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: true,
    timeout: 180_000,
  },
})
