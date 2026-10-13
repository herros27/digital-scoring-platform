# E2E Playwright: Publik

Spec untuk tampilan yang dilihat penonton tanpa login: layar TV scoreboard gelanggang. Data partai dibuat lewat API,
lalu wasit dan juri menggerakkannya lewat layar, dan layar penonton dicek secara realtime.

| Spec              | Cakupan                                                                                                                                                                                  |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `scoreboard-tv`   | daftar "Sedang Berlangsung" memuat partai live, atlet/kontingen/jumlah babak sesuai ruleset, timer turun, poin juri realtime, sanksi/potongan, jeda, sinyal dokter, dan layar pemenang setelah SELESAIKAN |

## Menjalankan

Dari `apps/web`, dengan backend dan Vite dev server sudah berjalan:

```bash
npm run playwright:pw -- e2e/publik
```
