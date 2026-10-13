import './test/cypress/scripts/load-env.mjs'
import { defineConfig } from '@playwright/test'

/**
 * Playwright Test untuk skenario multi-perangkat (3 layar juri + 1 konsol wasit, masing-masing browser context
 * sendiri). Backend dan Vite dev server harus sudah berjalan.
 *
 *   npm run playwright:pw # headless
 *   npm run playwright:pw:ui # UI Mode: pilih tes, time-travel, watch
 *   npm run playwright:pw:headed # 4 jendela browser tersusun grid 2x2 (layar laptop)
 *
 * Variabel opsional: E2E_BASE_URL (bawaan http://127.0.0.1:5173), PW_CHROMIUM (path Chromium bila bukan
 * bawaan Playwright), PW_ZOOM (skala tampilan saat --headed, bawaan 0.5 agar layar juri muat di 1/4 layar).
 */
const headed = process.argv.includes('--headed')
const zoom = Number(process.env.PW_ZOOM ?? 0.5)

export default defineConfig({
  testDir: './test/playwright/e2e',
  testMatch: '**/*.spec.ts',
  // uji PWA butuh build produksi + vite preview: dijalankan terpisah (npm run playwright:pw:pwa)
  testIgnore: '**/pwa.spec.ts',
  // Semua tes berbagi satu backend/gelanggang: jalan berurutan, tiap tes membuat partainya sendiri
  fullyParallel: false,
  workers: 1,
  outputDir: 'test/playwright/test-results',
  timeout: 90_000,
  expect: { timeout: 8_000 },
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'test/playwright/report' }]],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://127.0.0.1:5173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    launchOptions: {
      executablePath: process.env.PW_CHROMIUM || undefined,
      slowMo: headed ? Number(process.env.PW_SLOWMO ?? 150) : 0,
      args: ['--no-sandbox', ...(headed ? [`--force-device-scale-factor=${zoom}`] : [])],
    },
  },
})
