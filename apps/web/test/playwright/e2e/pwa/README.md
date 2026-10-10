# E2E Playwright: PWA

Uji PWA pada **build produksi**. Service worker hanya terdaftar di build, bukan di `vite dev`, jadi folder ini
memakai config terpisah, `playwright.pwa.config.ts`, yang membangun aplikasi lalu menyajikannya dengan `vite preview`
di port 4173. Backend harus tetap berjalan.

| Spec           | Cakupan                                                                                                                                       |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `pwa.spec.ts`  | manifest dan semua ikon terunduh dengan ukuran yang dijanjikan; tidak bergantung internet; service worker aktif dan bisa dipasang; offline hanya membuka cangkang aplikasi (data API tidak pernah dari cache); tablet juri dengan PWA tetap bisa login PIN dan realtime |

## Menjalankan

Dari `apps/web`:

```bash
npm run playwright:pw:pwa
```

Config ini tidak memakai `E2E_BASE_URL` (baseURL-nya `http://127.0.0.1:4173`). Jangan jalankan `playwright:pw` untuk folder
ini; config utama sengaja mengabaikan `pwa.spec.ts`.
