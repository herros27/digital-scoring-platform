# Digital Scoring Pencak Silat

**Sistem penilaian digital, jadwal, dan bagan pertandingan untuk kejuaraan Pencak Silat kategori Tanding.**

Seluruh proses kejuaraan, dari pendaftaran kontingen sampai perolehan medali, berjalan dalam satu aplikasi. Juri menilai
lewat tablet, skor sah muncul seketika di layar penonton, dan panitia tidak perlu lagi merekap kertas penilaian.

---

## 1. Masalah yang diselesaikan

| Cara manual | Dengan Digital Scoring |
|---|---|
| Juri menulis skor di kertas, direkap panitia setelah ronde | Juri menekan tombol di tablet, skor sah langsung terhitung |
| Perbedaan hitungan antar juri diputuskan secara lisan | Skor dihitung otomatis dengan **konsensus juri** (mayoritas menekan hal yang sama dalam beberapa detik) |
| Pendaftaran lewat berkas dan chat, rawan salah kategori | Kontingen mendaftar sendiri, sistem menentukan kelas usia dan berat badan |
| Bagan digambar tangan, berubah-ubah | Bagan dibuat otomatis, pemenang naik sendiri ke babak berikutnya |
| Penonton dan pelatih bertanya papan skor ke meja panitia | Scoreboard, bagan, dan jadwal dapat dibuka di layar atau ponsel tanpa login |
| Rekap medali dan laporan menghabiskan waktu setelah acara | Klasemen medali dan arsip hasil tersedia otomatis |

---

## 2. Siapa yang memakai

| Pengguna | Perangkat | Yang dikerjakan |
|---|---|---|
| **Admin / Dewan Juri** | Laptop | Menyiapkan kejuaraan, kategori, gelanggang, akun, memverifikasi peserta, membuat bagan dan jadwal |
| **Operator** | Laptop | Memantau jadwal dan jalannya pertandingan |
| **Wasit** | Tablet / laptop di meja wasit | Mengendalikan pertandingan: mulai, jeda, sanksi, keputusan |
| **Juri** | Tablet, satu per kursi | Memberi nilai teknik secara langsung |
| **Manajer kontingen** | Ponsel / laptop | Mendaftarkan tim dan atlet, memantau status verifikasi |
| **Penonton, pelatih, orang tua** | Layar TV atau ponsel (tanpa login) | Melihat skor langsung, jadwal, dan bagan |

---

## 3. Fitur lengkap

### 3.1 Penilaian langsung di gelanggang

- **Tablet juri.** Tombol besar per jenis teknik (Pukulan, Tendangan, Jatuhan, Tangkapan + Jatuhan, Kombinasi, dan lainnya)
  untuk sudut merah dan biru. Jumlah juri fleksibel (1, 3, 5, dst), tiap juri menempati satu kursi.
- **Konsensus juri.** Sebuah poin sah hanya bila cukup juri (misalnya 2 dari 3) menekan teknik dan sudut yang sama dalam
  jendela waktu singkat (bawaan 3 detik). Tablet juri menampilkan lampu kemajuan konsensus tanpa membuka identitas juri lain.
  Input ganda dan tekanan beruntun yang tidak disengaja difilter otomatis.
- **Koreksi (undo)** oleh juri bila salah menekan.
- **Konsol wasit.** Wasit memegang kendali penuh:
  - mulai, jeda, lanjut, dan akhiri pertandingan, serta koreksi waktu dan ulang ronde;
  - sanksi (Binaan, Teguran, Peringatan) dan pembatalan sanksi terakhir, dengan potongan nilai otomatis;
  - diskualifikasi;
  - hasil khusus: **KO, TKO, WO (walkover), dan undur diri**;
  - sinyal **gong**, **dokter (medic)**, dan **VAR** yang tampil juga di scoreboard;
  - antrean partai, panel status juri (siapa yang terhubung dan aktif), dan **log kejadian** seluruh pertandingan.
- **Timer dari server.** Waktu ronde dan istirahat dihitung di server, sehingga wasit, juri, dan scoreboard selalu
  menampilkan waktu yang sama walaupun tab browser diperlambat.
- **Aturan WMP** (wasit menghentikan pertandingan karena selisih poin): sistem menghentikan partai otomatis dan
  memenangkan sudut yang unggul. Batas selisih mengikuti golongan umur dan dapat diubah atau dimatikan.
- **Penentuan pemenang saat skor seri**, berurutan sesuai regulasi:
  1. potongan sanksi paling sedikit menang;
  2. teknik bernilai tertinggi terbanyak menang (urutan prioritas teknik dapat diatur);
  3. bila masih seri, wasit mencatat keputusan: babak tambahan, timbang ulang, undian koin, atau keputusan wasit.
  Sistem tidak pernah memenangkan sudut tertentu secara otomatis.
- **Riwayat lengkap (audit trail).** Setiap skor, sanksi, dan perubahan status tercatat dan dapat ditelusuri kembali.

### 3.2 Scoreboard untuk penonton

- Tampilan layar penuh per gelanggang untuk TV atau proyektor: nama atlet dan kontingen, skor, ronde, sisa waktu, sanksi,
  serta sinyal dokter/VAR.
- Daftar seluruh gelanggang yang sedang bertanding, dibuka tanpa login.
- Bagan dan jadwal dapat dilihat penonton lewat ponsel masing-masing.

### 3.3 Pendaftaran kontingen dan atlet

- **Portal kontingen.** Manajer tim mendaftar akun sendiri (dapat dibatasi dengan **kode undangan** dari panitia) lalu
  mendaftarkan atlet asuhannya ke kejuaraan.
- **Penentuan kategori otomatis.** Dari tanggal lahir, jenis kelamin, dan berat badan, sistem menentukan golongan umur
  dan kelas berat badan. Preset kategori mengacu pada **Peraturan Pertandingan PB IPSI 2025**, dengan kelas Under, A, B, C,
  dan seterusnya, serta kelas Open.
- **Golongan umur** yang didukung: Pra Usia Dini, Usia Dini 1 dan 2, Pra-Remaja, Remaja, Dewasa, Master A dan B.
- **Verifikasi panitia (meja timbang).** Atlet yang didaftarkan kontingen berstatus *menunggu*, lalu panitia
  memverifikasi berkas dan hasil timbang menjadi *terverifikasi* atau *ditolak*.
- **Pengaman bagan.** Hanya atlet yang sudah terverifikasi yang dapat masuk bagan.
- Kontingen hanya dapat melihat dan mengelola atletnya sendiri, dan dapat membatalkan pendaftaran selama belum diverifikasi.

### 3.4 Kategori, bagan, dan jadwal

- **Kategori pertandingan** dibuat sekali klik dari preset (umur, jenis kelamin, kelas berat), dapat disesuaikan, dan
  dikunci setelah ada peserta agar data tidak berubah di tengah acara.
- **Bagan sistem gugur** dibuat otomatis: penempatan unggulan standar, penanganan **BYE**, dan pemenang naik sendiri ke
  babak berikutnya. Walkover dan undur diri didukung.
- **Beberapa gelanggang sekaligus.** Tiap kategori dibagikan ke gelanggang dengan beban paling ringan, dan seluruh
  partai kategori itu dimainkan di gelanggang yang sama.
- **Jadwal partai**: membuat, memuat ke gelanggang, membatalkan, dan menjadwalkan ulang partai (skor kembali 0-0).
  Sistem mencegah dua partai berjalan bersamaan di satu gelanggang.
- Bagan publik dapat dibuka penonton tanpa login, hanya menampilkan nama atlet dan kontingen (data pribadi tidak tampil).

### 3.5 Regulasi yang bisa diatur sendiri (Ruleset)

Aturan pertandingan tidak tertanam di program, melainkan dapat diatur dari layar **Regulasi & Ruleset**:

- durasi ronde, durasi istirahat, dan jumlah ronde;
- daftar teknik beserta nilainya;
- daftar sanksi beserta potongan nilai, termasuk sanksi yang langsung mendiskualifikasi;
- jumlah juri, ambang mayoritas, dan jendela waktu konsensus;
- aturan seri dan urutan prioritas teknik;
- batas selisih poin WMP.

Tersedia preset **IPSI 2024** dan **PERSILAT International**. Preset dapat diduplikasi lalu disesuaikan dengan
peraturan kejuaraan (misalnya aturan khusus event daerah). Ruleset dipilih per kejuaraan, dan perubahan tidak mengganggu
partai yang sedang berjalan.

### 3.6 Laporan dan medali

- Ringkasan kejuaraan: jumlah peserta, gelanggang, dan partai.
- **Klasemen perolehan medali** (emas, perak, perunggu) per kontingen.
- **Ekspor arsip** lengkap hasil kejuaraan.

### 3.7 Manajemen akun dan keamanan

- **Peran berjenjang**: Admin, Dewan Juri, Wasit, Operator, Juri, dan Kontingen, masing-masing hanya melihat dan
  mengerjakan yang menjadi wewenangnya.
- **Login juri dan wasit dengan PIN** yang dibuat panitia. PIN menentukan gelanggang dan kursi, tidak bisa dipakai di
  gelanggang lain, dan memiliki masa berlaku.
- **Perlindungan tebak PIN**: percobaan salah berulang akan mengunci sementara perangkat dan alamat jaringan terkait.
- Password disimpan terenkripsi (bcrypt), sesi memakai token bertanda tangan, dan ada pembatasan laju permintaan
  terhadap serangan beruntun.
- Data pribadi atlet (tanggal lahir, NIK) tidak tampil di halaman publik.
- Admin dapat menambah, mengubah peran, mereset password, dan menghapus akun petugas, serta mereset password manajer kontingen.

### 3.8 Ketahanan di hari pertandingan

- **Tahan restart server.** Keadaan partai yang sedang berjalan tersimpan terus-menerus. Bila server mati atau
  dihidupkan ulang, partai kembali dalam keadaan **dijeda** dengan sisa waktu terakhir, lalu wasit memeriksa dan melanjutkan.
- **Tablet tersambung ulang otomatis** dan menyinkronkan keadaan terakhir pertandingan.
- **Dapat dipasang sebagai aplikasi (PWA)** di tablet: ikon sendiri, layar penuh tanpa bilah alamat, dan tampilan awal
  tetap terbuka walau jaringan sempat putus.
- **Cadangan database** dengan skrip siap pakai untuk dijalankan terjadwal.
- **Pemantauan kesehatan** sistem untuk memastikan server dan database dalam keadaan baik.

---

## 4. Alur kerja satu kejuaraan

```
1. Admin membuat kejuaraan, memilih ruleset, dan menambah gelanggang
2. Admin membuat kategori (dari preset umur dan kelas berat badan)
3. Kontingen mendaftar akun lalu mendaftarkan atlet
4. Panitia memverifikasi dan menimbang peserta di meja timbang
5. Admin membuat bagan, partai otomatis tersusun per gelanggang
6. Admin membuat PIN untuk juri dan wasit tiap gelanggang
7. Hari-H: wasit memuat partai ke gelanggang, juri menilai, penonton melihat scoreboard
8. Pemenang naik ke babak berikutnya otomatis
9. Selesai: klasemen medali dan arsip hasil siap diunduh
```

---

## 5. Cara aplikasi dijalankan

Dua pilihan, tanpa perubahan pada aplikasinya:

| | Mode lomba (di venue) | Mode demo (lewat internet) |
|---|---|---|
| Server | Laptop panitia di jaringan lokal (LAN) venue | Server terpisah di internet |
| Akses | Tablet dan layar terhubung ke Wi-Fi atau router venue | Dibuka dari browser mana pun lewat alamat web |
| Data | Data kejuaraan asli | Data contoh, aman untuk dicoba |
| Internet saat acara | Tidak wajib bila sertifikat sudah disiapkan sebelumnya | Wajib |

Seluruh perangkat cukup membuka satu alamat web; tidak ada aplikasi tambahan yang perlu dipasang selain pemasangan PWA
di tablet (opsional). Pemasangan memakai Docker sehingga mudah dipindahkan dan dipulihkan.

---

## 6. Teknologi

| Bagian | Teknologi |
|---|---|
| Server | Go |
| Antarmuka | Vue 3 + TypeScript (responsif: laptop, tablet, ponsel) |
| Komunikasi langsung | WebSocket (skor, timer, dan status tampil tanpa muat ulang halaman) |
| Database | PostgreSQL 16 dengan migrasi otomatis |
| Pemasangan | Docker Compose, HTTPS otomatis lewat Caddy |
| Pemantauan | Endpoint kesehatan dan metrik Prometheus |
| Pengujian | Tes unit dan integrasi sisi server, tes otomatis browser (Playwright) |

---

## 7. Cakupan saat ini

Agar ekspektasi jelas:

- Fokus pada **kategori Tanding**. Kategori seni (Tunggal, Ganda, Regu) dan Jurus belum termasuk.
- Sistem bagan yang tersedia adalah **sistem gugur (single elimination)**. Sistem pool atau round-robin belum termasuk.
- **Babak tambahan, timbang ulang, dan undian koin** untuk skor seri dilakukan secara fisik di gelanggang; hasilnya
  dicatat oleh wasit ke dalam sistem.
- Penilaian, login PIN, dan scoreboard membutuhkan server tetap menyala dan terjangkau lewat jaringan. Mode offline pada
  PWA hanya menjaga tampilan awal aplikasi, bukan pengganti server.
- Pengunggahan dan penyimpanan berkas dokumen atlet (akta, ijazah, surat sehat) belum ada di aplikasi; pemeriksaan
  berkas dilakukan panitia di meja verifikasi dan hasilnya ditandai di sistem.

---

## 8. Yang dapat disesuaikan

- Aturan pertandingan (ruleset) mengikuti peraturan kejuaraan Anda.
- Kategori dan kelas berat badan.
- Jumlah gelanggang dan juri.
- Alamat web dan identitas tampilan sesuai penyelenggara.
- Pembatasan pendaftaran kontingen dengan kode undangan.

---

*Dokumen ini adalah gambaran fungsional untuk calon pengguna. Dokumentasi teknis tersedia terpisah di README dan folder `docs/`.*
