# E2E Playwright: Panitia

Spec UI sungguhan untuk ruang operasi panitia (admin dan dewan): bagan, jadwal, kategori, peserta, akun petugas, dan
PIN juri. Setiap spec memakai kejuaraan uji sendiri, dan data dibuat lewat API lalu dicek ulang ke server setelah
aksi di layar.

| Spec                     | Cakupan                                                                                                                                                              |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `administrasi-ui`        | ubah dan hapus kategori (setelah ada peserta hanya nama yang berubah), batal daftar atlet oleh admin dan manajer kontingen (ditolak bila sudah di bagan), reset password kontingen, tambah partai manual, rincian skor per partai |
| `akun-petugas-ui`        | layar Akun Petugas: tambah operator, validasi form, ubah peran, reset password, hapus dengan konfirmasi, batas wewenang Dewan Juri, akun sendiri tidak bisa dihapus, ganti password sendiri |
| `gelanggang-asal-ui`     | gelanggang asal kategori otomatis, bisa diubah dari kartu kategori, dan wasit gelanggang lain ditolak                                                               |
| `kompetisi-daftar-ui`    | angka tab tetap sama di semua filter dan cocok dengan server, daftar dipaginasi, tab Dibatalkan dan pencarian lintas data                                           |
| `panitia-ui`             | bagan (generate, acak ulang, kunci draw bertahan setelah muat ulang), dock bagan "Kirim ke Gelanggang", jadwal (hitung, filter, muat, mulai, jeda, selesaikan), PIN juri kustom dan Reset PIN |

## Menjalankan

Dari `apps/web`, dengan backend dan Vite dev server sudah berjalan:

```bash
npm run playwright:pw -- e2e/panitia               # semua spec folder ini
npm run playwright:pw -- e2e/panitia/akun-petugas  # satu file
npm run playwright:pw:ui                           # UI Mode
```

Login dibatasi 10 kali per menit per IP. Menjalankan beberapa spec berturut-turut dengan cepat bisa memunculkan
"too many requests" di langkah login; tunggu semenit lalu ulangi.
