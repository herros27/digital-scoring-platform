# E2E Playwright: Pertandingan

Spec UI sungguhan untuk alur partai: juri, wasit, sanksi, dan bagan. Berbeda dengan Cypress (`test/cypress/e2e/match`),
spec ini membuka **banyak browser context sekaligus** (3 juri + wasit), sehingga hal seperti "tiga juri menekan
bersamaan" bisa diuji lewat layar. Setiap tes membuat partainya sendiri, jadi bisa dijalankan satu per satu.

| Spec                          | Cakupan                                                                                                                                                  |
| ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `tiga-juri-satu-wasit`        | 3 juri + 1 wasit: semua online, MULAI, tiga juri menekan bersamaan (tepat 1 poin), satu suara menunggu lalu padam, BATAL, JEDA/LANJUT mengunci layar, dua juri bersamaan, SELESAIKAN (tombol skor terkunci) |
| `kompetisi-penuh-ui`          | SATU kompetisi penuh lewat layar: panitia login lewat form, aktivasi kejuaraan, penonton membuka `/bagan` (tersembunyi sebelum aktif, lalu tampil), 3 partai (2 semifinal + final otomatis), juara tampil di publik |
| `sanksi-hasil-khusus`         | sanksi dari konsol wasit (Binaan/Teguran/DQ, batalkan sanksi, sanksi tidak bisa dobel), Peringatan 3 dan diskualifikasi, hasil khusus (KO/TKO/WO/Undur diri) dan dampaknya ke bagan dan medali |
| `referee-bagan-ui`            | wasit melihat bagan kategorinya; partai di gelanggangnya ditandai, gelanggang lain hanya dilihat                                                         |
| `bagan-mobile-ui`             | bagan tampil dan bisa digeser di layar ponsel                                                                                                            |

## Menjalankan

Dari `apps/web`, dengan backend dan Vite dev server sudah berjalan (`E2E_BASE_URL`, lihat [../../README.md](../../README.md)):

```bash
npm run playwright:pw -- e2e/match                 # semua spec folder ini
npm run playwright:pw -- e2e/match/tiga-juri       # satu file (sebagian nama cukup)
npm run playwright:pw:headed -- e2e/match          # lihat 4 jendela browser (grid 2x2)
npm run playwright:pw:ui                           # UI Mode: pilih tes satu per satu
```

Tes wasit dan juri memakai `openAs` dan `openCrew` dari [../../support/fixtures.ts](../../support/fixtures.ts). Helper
data (kejuaraan, kategori, bagan) berasal dari `test/cypress/support/match/`.
