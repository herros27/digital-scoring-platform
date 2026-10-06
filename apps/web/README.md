# DigitalScoring Web Application (`apps/web`)

Modern, industry-standard Vue 3 frontend for the DigitalScoring Pencak Silat platform.

## Architecture & Specifications

- [Frontend Technical Specification](file:///d:/Freelance/DigitalScoring/apps/web/FRONTEND_SPECIFICATION.md): Dokumen spesifikasi teknis lengkap, standar arsitektur industri, tech stack, state management, testing, dan deployment.
- [Frontend Integration Plan](file:///d:/Freelance/DigitalScoring/apps/web/INTEGRATION_PLAN.md): Cetak biru integrasi frontend Vue 3 ke Go Backend & WebSocket Hub.
- [Frontend Design Master Prompt](file:///d:/Freelance/DigitalScoring/apps/web/FRONTEND_DESIGN_PROMPT.md): Panduan UI/UX, alur pertandingan, dan layout arena.

## Scripts

Jalankan dari folder `apps/web`:

| Perintah | Fungsi |
|----------|--------|
| `npm run dev` | Dev server Vite (port 5173, proxy `/api` dan `/ws` ke backend di `localhost:8080`) |
| `npm run build` | Typecheck (`vue-tsc -b`) lalu build produksi |
| `npm run typecheck` | Hanya typecheck |
| `npm run lint` / `npm run lint:fix` | ESLint (`@typescript-eslint/no-explicit-any` = error; tidak boleh ada `any`) |
| `npm run format` / `npm run format:check` | Prettier (CI belum memaksa `format:check` karena kode lama belum diformat) |
| `npm test` | Vitest (guard router, helper PIN, helper error, pemetaan status pertandingan) |

CI (`.github/workflows/ci.yml`, job `frontend`) menjalankan lint, typecheck, tes, dan build.

### Simulasi end-to-end terhadap backend

Dengan backend dan data seed berjalan (`go run ./cmd/seed`, `go run ./cmd/server`):

```bash
node scripts/e2e-simulation.mjs
```

Skrip memverifikasi login PIN-saja, akses WebSocket (penonton publik, token PIN, token arena lain), hak akses perintah, konsensus juri, pause/resume, hasil pertandingan, dan `GET /live/scoreboard`. Keluar dengan kode 1 bila ada pemeriksaan gagal.

### Login perangkat gelanggang

`/judge/login` hanya meminta PIN (juri 6 digit, wasit 8 digit). PIN dibuat dari halaman **Perangkat & PIN** (`/ops?tab=pin`) dan ditampilkan di sana. Rute `/ops` wajib login (role `admin`, `dewan`, `operator`, `contingent`); guard router sekarang memeriksa `allowedRoles` pada semua rute.
