import './cypress/scripts/load-env.mjs'
import { execSync } from 'node:child_process'
import { defineConfig } from 'cypress'

export default defineConfig({
  e2e: {
    baseUrl: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    viewportWidth: 1920,
    viewportHeight: 1080,
    video: false,
    screenshotOnRunFailure: true,
    defaultCommandTimeout: 8000,
    retries: { runMode: 1, openMode: 0 },
    supportFile: 'cypress/support/e2e.ts',
    specPattern: 'cypress/e2e/**/*.cy.ts',
    // spec pemulihan restart hanya berjalan bila lingkungan menyediakan perintah restart backend
    expose: { E2E_RESTART: Boolean(process.env.E2E_RESTART_CMD) },
    setupNodeEvents(on) {
      // Cache token API di proses Node supaya seluruh spec dalam satu run hanya login sekali per
      // akun (endpoint login dibatasi 10 request/menit). Tidak ada state yang disimpan ke disk.
      const tokens = new Map<string, string>()
      on('task', {
        'server:restart': () => {
          if (!process.env.E2E_RESTART_CMD) throw new Error('E2E_RESTART_CMD tidak diset')
          execSync(process.env.E2E_RESTART_CMD, { stdio: 'inherit', timeout: 120_000 })
          return null
        },
        'tokenCache:get': (key: string) => tokens.get(key) ?? null,
        'tokenCache:set': ({ key, token }: { key: string; token: string }) => {
          tokens.set(key, token)
          return null
        },
      })
    },
  },
})
