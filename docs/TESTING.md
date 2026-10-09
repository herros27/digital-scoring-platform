# Panduan Menjalankan Semua Tes

Ringkasan satu pintu. Detail tiap suite ada di `apps/web/cypress/e2e/match/README.md` (alur partai, Playwright,
temuan) dan `README.md` akar (daftar tes Go).

## Prasyarat umum

| Kebutuhan           | Cara                                                          |
| ------------------- | ------------------------------------------------------------- |
| PostgreSQL hidup    | `docker compose up -d postgres` (atau Postgres lokal)         |
| `.env` di akar repo | salin dari `.env.example`; `DB_URL` menunjuk ke Postgres tadi |
| Dependensi frontend | `cd apps/web && npm ci`                                       |
| Browser Playwright  | `cd apps/web && npx playwright install chromium` (sekali)     |

## Ringkasan suite

| Suite                                                                  | Butuh                              | Perintah                               | Dari folder |
| ---------------------------------------------------------------------- | ---------------------------------- | -------------------------------------- | ----------- |
| Go (unit + integrasi)                                                  | Postgres                           | `go test -count=1 ./...`               | akar repo   |
| Frontend unit (vitest)                                                 | -                                  | `npm test`                             | `apps/web`  |
| Tipe + lint                                                            | -                                  | `npm run typecheck` dan `npm run lint` | `apps/web`  |
| Alur partai headless (spec 01-12)                                      | backend + Vite                     | `npm run test:match-flow`              | `apps/web`  |
| Playwright (4 layar, kompetisi penuh, TV, panitia, akun, administrasi) | backend + Vite                     | `npm run test:pw`                      | `apps/web`  |
| Playwright PWA                                                         | backend (build + preview otomatis) | `npm run test:pw:pwa`                  | `apps/web`  |
| Cypress (spec `ui-*` dan admin)                                        | backend + Vite + binary Cypress    | `npm run cypress:run`                  | `apps/web`  |

Tes Go yang memakai database akan dilewati bila Postgres tidak ada (`TEST_DB_URL` untuk database lain).

## Urutan lengkap (satu kali jalan)

Terminal 1, akar repo:

```bash
go run ./cmd/seed/main.go      # sekali, mengisi data awal
go run ./cmd/server            # backend :8080 (atau: docker compose up -d)
```

Terminal 2, `apps/web`:

```bash
npm run dev                    # Vite :5173
```

Terminal 3, `apps/web`:

```bash
npm run typecheck && npm run lint && npm test
npm run test:match-flow        # +-3 menit
npm run test:pw                # headless
npm run test:pw:pwa
```

Lalu di akar repo: `go test -count=1 ./...`.

## Variabel uji lewat `.env`

Variabel berawalan `E2E_`, `PW_`, dan `CYPRESS_` otomatis dibaca dari `.env` **di akar repo** oleh `test:match-flow`,
Playwright, dan Cypress (yang sudah diset di shell menang). Contoh:

```
E2E_BASE_URL=http://localhost
E2E_RESTART_CMD=powershell -NoProfile -ExecutionPolicy Bypass -File ..\..\scripts\restart-backend.ps1 -Container scoring-backend
PW_CHROMIUM=C:\path\ke\chrome.exe
```

Jangan taruh variabel ini di `.env.prod`: berkas itu menjadi environment kontainer backend, bukan untuk tes.
Perintah `E2E_RESTART_CMD` memakai path relatif, jadi jalankan tes dari `apps/web`.

## Pilihan berguna

- Satu spec headless: `npm run test:match-flow -- 03`
- Playwright dengan UI untuk memilih tes: `npm run test:pw:ui`; lihat jendela browser: `npm run test:pw:headed`
- Satu tes Playwright berdasarkan nama: `npx playwright test -g "BATAL"`
- Laporan HTML: `npx playwright show-report`
- Menguji build Docker (nginx, port 80) alih-alih server dev: set `E2E_BASE_URL=http://localhost`

## Spec yang butuh pengaturan khusus

- **Spec 06 (pemulihan restart server)** dilewati kecuali `E2E_RESTART_CMD` diset. Windows (PowerShell):
  ```powershell
  $env:E2E_RESTART_CMD = 'powershell -NoProfile -ExecutionPolicy Bypass -File ..\..\scripts\restart-backend.ps1'
  npm run test:match-flow -- 06
  ```
  Linux/macOS: `E2E_RESTART_CMD="$PWD/../../scripts/restart-backend.sh" npm run test:match-flow -- 06`.
  Backend di Docker: tambahkan `-Container scoring-backend` (PowerShell) atau `CONTAINER=scoring-backend` (bash).
- **Playwright PWA** membangun frontend sendiri dan menyajikannya di :4173.

## Masalah umum

- `ECONNREFUSED 127.0.0.1:5173` / "Backend/Vite tidak merespons": jalankan backend dan `npm run dev` dulu.
- "too many requests" saat login: login dibatasi per menit; tunggu semenit lalu ulangi.
- `failed to ping db ... connection refused`: Postgres belum hidup atau `DB_URL` di `.env` salah.
- Cypress tidak bisa dibuka: binary belum terpasang (`npx cypress install`). Spec 01-12 tetap bisa dijalankan
  lewat `test:match-flow` tanpa binary itu.
