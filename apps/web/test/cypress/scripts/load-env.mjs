// Muat variabel uji (E2E_*, PW_*, CYPRESS_*) dari .env di akar repo, sehingga tidak perlu `$env:` setiap kali.
// Yang sudah diset di shell menang atas isi berkas. Variabel lain di .env (JWT_SECRET, DB_URL, ...) sengaja diabaikan.
import process from 'node:process'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { parseEnv } from 'node:util'

const file = fileURLToPath(new URL('../../../../../.env', import.meta.url))
if (existsSync(file)) {
  for (const [key, value] of Object.entries(parseEnv(readFileSync(file, 'utf8')))) {
    if (/^(E2E_|PW_|CYPRESS_)/.test(key) && process.env[key] === undefined)
      process.env[key] = key === 'E2E_BASE_URL' ? value.replace(/\/+$/, '') : value
  }
}
