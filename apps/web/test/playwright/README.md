# Playwright Test

Uji UI sungguhan di browser, berjalan terhadap backend dan frontend yang sudah berjalan. Penjelasan tiap spec ada di
README masing-masing folder:

- [e2e/match/](e2e/match/README.md): alur partai (juri, wasit, sanksi, bagan).
- [e2e/panitia/](e2e/panitia/README.md): ruang operasi panitia dan dewan (bagan, jadwal, kategori, akun, PIN).
- [e2e/publik/](e2e/publik/README.md): layar TV scoreboard untuk penonton.
- [e2e/pwa/](e2e/pwa/README.md): uji PWA pada build produksi.

Helper bersama ada di [support/](support/), dan config ada di `apps/web/playwright.config.ts` (uji utama) dan
`apps/web/playwright.pwa.config.ts` (uji PWA).

## Perintah

Dari `apps/web`:

```bash
npm run playwright:pw              # headless, semua spec kecuali PWA
npm run playwright:pw:ui           # UI Mode: pilih dan jalankan tes satu per satu
npm run playwright:pw:headed       # browser terlihat, grid 2x2 (skala PW_ZOOM, bawaan 0.5)
npm run playwright:pw:pwa          # uji PWA (membangun aplikasi lebih dulu)
npx playwright test -g "BATAL"     # pilih tes lewat nama
npx playwright show-report test/playwright/report  # laporan HTML (trace tersimpan bila tes gagal)
```

Untuk satu folder atau file, tambahkan path setelah `--`, misalnya `npm run playwright:pw -- e2e/match`.

## Prasyarat

- Backend dan Vite dev server sudah berjalan. Target diatur oleh `E2E_BASE_URL`, bawaan `http://127.0.0.1:5173`.
  Untuk build Docker (nginx di port 80), set `E2E_BASE_URL=http://localhost`.
- Browser Playwright sudah terpasang: `npx playwright install chromium`.

Variabel opsional: `E2E_BASE_URL`, `PW_CHROMIUM` (path Chromium bila bukan bawaan Playwright), `PW_ZOOM`, `PW_SLOWMO`.

> Login dibatasi 10 kali per menit per IP. Menjalankan beberapa suite berturut-turut dengan cepat bisa memunculkan
> "too many requests" di langkah login; tunggu semenit lalu ulangi.
