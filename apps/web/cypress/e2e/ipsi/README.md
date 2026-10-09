# E2E: Kategori & Registrasi IPSI

Spec level kontrak API (cy.request, backend nyata, tanpa mock) + satu cek UI. Setiap `describe`
membuat kompetisi sendiri (`E2E <label> <waktu>`) dengan tanggal mulai tetap `2026-06-01`, lalu
membatalkannya di `after`, sehingga data seed tidak tersentuh dan hasil umur selalu deterministik.

| Spec                                  | Cakupan                                                                                                                                        |
| ------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `01-kategori-ipsi`                    | default IPSI, validasi bobot/golongan/preset/mode, satu standar per kejuaraan, reset (409), isolasi                                            |
| `02-registrasi-atlet`                 | batas usia 8 golongan (min/max ±1 tahun, ulang tahun tepat di hari acuan), gender, birth_date, NIK, peran pendaftar                            |
| `03-verifikasi-timbang`               | berkas wajib per gender/golongan, timbang 0 kg toleransi (inklusif), timbang ulang, status, RBAC                                               |
| `04-bagan-per-kategori`               | struktur bagan 2/3/4/5/8/9 peserta, hanya `verified`, isolasi kategori, regenerate, akses                                                      |
| `05-matriks-hak-akses`                | anonim / kontingen / dewan / admin x endpoint kategori-registrasi-verifikasi-bagan                                                             |
| `07-isolasi-data-kontingen`           | kontingen hanya melihat peserta/atlet/profil miliknya + statusnya; tidak bisa mendaftarkan atlet kontingen lain                                |
| `08-bagan-publik`                     | bagan terbuka tanpa login hanya saat kejuaraan aktif; whitelist field + tanpa NIK/tanggal lahir; halaman `/bagan` (daftar, filter, buka bagan) |
| `06-alur-lengkap-registrasi-ke-bagan` | skenario berurutan kontingen -> verifikasi -> bagan -> tampilan publik + UI meja verifikasi                                                    |

## Menjalankan

```bash
# terminal 1 (root repo): backend + seed
go run ./cmd/seed/main.go && go run ./cmd/server
# terminal 2 (apps/web): frontend
npm run dev
# terminal 3 (apps/web)
npx cypress run --spec "cypress/e2e/ipsi/**"
```

Token API di-cache lintas spec lewat `cy.task('tokenCache:*')` (login dibatasi 10/menit). Panggilan yang
terkena rate limit global (429) dicoba ulang otomatis oleh `apiCall` di `support/api.ts`.
