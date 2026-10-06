# Dokumentasi Progres Pengembangan Frontend (Digital Scoring Pencak Silat)

Dokumen ini melacak secara menyeluruh perkembangan antarmuka (*frontend web & PWA*), status integrasi API/WebSocket, kesiapan modul per halaman, serta daftar kekurangan dan rencana penyempurnaan aplikasi.

---

## 1. Ringkasan Eksekutif & Status Kesiapan

| Metrik | Status Saat Ini | Keterangan |
| :--- | :---: | :--- |
| **Total Progres Frontend** | **88%** | Seluruh tampilan inti operasional & pertandingan sudah terintegrasi ke backend server. |
| **Konektivitas API REST** | **95%** | Menggunakan TanStack Query v5 + Axios/Fetch client terstandarisasi. |
| **Konektivitas WebSocket** | **90%** | Real-time scoring, sync timer, status arena, dan konsensus terpasang pada Juri, Wasit, & Scoreboard. |
| **Kompilasi / Build Status** | **100% PASS** | Bebas error TypeScript (`vue-tsc -b`) dan lulus Vite production build. |
| **Lint, Tes & CI** | **PASS** | ESLint tanpa error (`no-explicit-any` = error, tidak ada `any`), 28 tes Vitest, job `frontend` di CI (lint, typecheck, tes, build). Verifikasi end-to-end ke backend nyata lewat `apps/web/scripts/e2e-simulation.mjs`. |
| **Desain & Responsivitas** | **85%** | Optimal pada layout tablet landscape (Juri/Wasit) dan desktop widescreen (Operator Ops). |

---

## 2. Matriks Status Fitur per Modul

| No | Modul / Halaman | Komponen / View | Status | Integrasi Backend | Tingkat Kesiapan |
| :---: | :--- | :--- | :---: | :---: | :---: |
| 1 | **Autentikasi & Sesi** | `LoginView.vue`, `RegisterView.vue`, `JudgeLoginView.vue` | Selesai | API REST (JWT & PIN) | 95% |
| 2 | **Komando Operasi (Shell)** | `OpsShellView.vue` | Selesai | REST + WS Telemetry | 90% |
| 3 | **Manajemen Kejuaraan** | `CompetitionsView.vue` | Selesai | REST (CRUD & Status) | 95% |
| 4 | **Operasional Gelanggang** | `ArenaOpsView.vue` | Selesai | REST (CRUD Gelanggang) | 90% |
| 5 | **Kategori Tanding** | `CategoryOpsView.vue` | Selesai | REST (CRUD Kategori) | 90% |
| 6 | **Kontingen & Database Atlet** | `AthletesOpsView.vue` | Selesai | REST (CRUD Atlet & Tim) | 88% |
| 7 | **Verifikasi & Timbang Badan** | `ParticipantVerificationView.vue` | Selesai | REST (Lolos/Tolak/Timbang) | 95% |
| 8 | **Bagan Pertandingan (Draw)** | `OpsShellView.vue` (Tab Bagan) | Selesai | REST (Auto Bracket Tree) | 85% |
| 9 | **Jadwal & Running Order** | `ScheduleOpsView.vue` | Selesai | REST (Partai & Arena Load) | 88% |
| 10 | **Perangkat & PIN Juri** | `DevicePinManager.vue` | Selesai | REST (Multi-device Sync) | 95% |
| 11 | **Tablet Skoring Juri** | `JudgeScoringView.vue` | Selesai | WS Real-time + Haptics | 92% |
| 12 | **Meja Wasit & Pengawas** | `RefereeConsoleView.vue` | Selesai | WS Real-time + Timer | 90% |
| 13 | **Layar TV Scoreboard Publik** | `ScoreboardView.vue` | Selesai | WS Real-time (Bebas PIN) | 92% |
| 14 | **Laporan & Ekspor Data** | `ReportsOpsView.vue` | Selesai | REST (CSV & Klasemen) | 85% |

---

## 3. Rincian Fitur yang Sudah Selesai (Completed)

### 3.1. Autentikasi & Keamanan Sesi
- [x] Login terpusat berbasis email dan password untuk Admin, Dewan Hakim, dan Wasit (`/login`).
- [x] Pendaftaran dan login akun delegasi Kontingen (`/register`).
- [x] Keypad login hanya dengan PIN (juri 6 digit, wasit 8 digit) di `/judge/login`; turnamen, gelanggang, role, dan kursi ditentukan server dari PIN. Pesan error Indonesia untuk PIN salah dan terkunci (429).
- [x] Penyimpanan token sesi aman pada `sessionStorage` dengan Pinia store (`stores/auth.ts`).
- [x] RBAC Router Navigation Guard (`router.beforeEach`) memeriksa token **dan** `allowedRoles` per rute (role tidak berhak dialihkan ke halamannya sendiri); `/ops` wajib login (`admin`, `dewan`, `operator`, `contingent`). Dilindungi tes.

### 3.2. Komando Operasi Turnamen (`OpsShellView.vue`)
- [x] Shell operasional modern dengan sidebar navigasi 9 modul kerja.
- [x] Switcher turnamen aktif multi-kejuaraan secara dinamis.
- [x] Header telemetri latensi WebSocket langsung ke server venue.
- [x] Modal fast-login operator terintegrasi tanpa perlu meninggalkan lembar kerja.

### 3.3. Manajemen Gelanggang & Kategori
- [x] Tambah, ubah nama, dan hapus gelanggang (`ArenaOpsView.vue`) dengan validasi nama unik per kompetisi.
- [x] Pengaturan kategori tanding berdasarkan rentang berat badan, jenis kelamin, dan golongan umur (`CategoryOpsView.vue`).
- [x] Preset golongan umur otomatis (Usia Dini, Pra-Remaja, Remaja, Dewasa, Master).

### 3.4. Pendaftaran Atlet & Verifikasi Timbang Badan
- [x] Pencarian cepat, filter jenis kelamin, dan filter kontingen pada database atlet (`AthletesOpsView.vue`).
- [x] Antarmuka Timbang Badan & Verifikasi Berkas (`ParticipantVerificationView.vue`).
- [x] Aksi tegas 3 status: **Loloskan (Sah)**, **Tolak (Ditolak)**, dan **Batalkan**.
- [x] Deteksi kelebihan/kekurangan berat badan (*weight delta check*) terhadap toleransi kelas tanding.

### 3.5. Bagan Pertandingan (Bracket Generator)
- [x] Generator otomatis bagan sistem eliminasi tunggal (*single-elimination tree*) dari atlet yang sah timbang badan.
- [x] Rendering pohon bagan visual interaktif (Babak Penyisihan, Perempat Final, Semifinal, dan Final).
- [x] Status partai terintegrasi (menunggu, berjalan/live, selesai).
- [x] Quick Action Dock: Tombol pemanggilan atlet dan *load match* langsung ke matras gelanggang.
- [x] Kunci bagan (*lock draw*) dan fungsi Cetak PDF bagan resmi.

### 3.6. Manajemen Perangkat & Sinkronisasi PIN (`DevicePinManager.vue`)
- [x] Pengambilan daftar PIN aktif langsung dari database server PostgreSQL (menghapus penyimpanan rentan di `localStorage`).
- [x] Multi-device sync: PIN dapat dilihat dari laptop panitia mana pun secara konsisten.
- [x] Format juri fleksibel: Tombol toggle format **5 Juri** (Standar PB IPSI) vs **3 Juri** (Kejuaraan Daerah/Ringkas).
- [x] Acak ulang massal (*Batch Generate PIN*) untuk seluruh juri dan wasit dalam 1 klik.
- [x] Reset PIN individual per kursi juri.
- [x] Pemisahan kartu Scoreboard TV Display sebagai layar publik bebas PIN.
- [x] Proteksi integritas *foreign key* dan auto-fallback jika ID gelanggang dihapus dari database.

### 3.7. Layar Pertandingan Real-Time
- [x] **Tablet Skoring Juri (`JudgeScoringView.vue`)**:
  - Tombol sentuh responsif Pukulan (+1), Tendangan (+2), Jatuhan (+3) untuk Sudut Merah dan Sudut Biru.
  - Haptic feedback (getaran tablet saat tombol ditekan).
  - Deduplikasi double-tap menggunakan client-side `idempotency_key` (UUID).
  - Indikator kesepakatan konsensus juri per sudut.
- [x] **Meja Pengawas & Wasit (`RefereeConsoleView.vue`)**:
  - Kontrol waktu tanding resmi: Mulai, Jeda, Lanjut, Selesai, dan Penyesuaian waktu (+/- detik).
  - Pencatatan sanksi & pelanggaran (Binaan 1–2, Teguran 1–2, Peringatan 1–2).
  - Keputusan khusus: Diskualifikasi (DQ), Walk Over (WO), Menang Teknik, dan Selisih Poin.
  - Pemicu sinyal Gong arena, panggilan tim medis, dan sinyal VAR.
- [x] **Layar TV Scoreboard Publik (`ScoreboardView.vue`)**:
  - Layout kiosk 16:9 full-screen beresolusi tinggi (bebas login PIN).
  - Tampilan skor besar, babak (ronde), waktu sisa, potongan hukuman, dan notifikasi animasi poin melayang. Skor, babak, dan timer diambil dari `sync_response`/`state_change`/`score_update` (TV yang baru menyala langsung memulihkan skor); data demo sudah dihapus. Jeda terlihat lewat `timer_running`.
  - Modal pengumuman pemenang pertandingan resmi.

### 3.8. Laporan & Ekspor (`ReportsOpsView.vue`)
- [x] Ringkasan pertandingan selesai, rasio penyelesaian turnamen, dan perolehan medali (Emas, Perak, Perunggu).
- [x] Podium Juara Umum 1, 2, dan 3.
- [x] Ekspor data klasemen medali dan riwayat pertandingan ke file CSV.

---

## 4. Apa yang Masih Kurang dari Website (Gap Analysis & Backlog)

Berikut rincian aspek yang masih kurang, belum lengkap, atau perlu ditingkatkan pada antarmuka frontend:

### 🔴 Prioritas Tinggi (P0 - Kritis untuk Kelancaran Pertandingan Venue)
1. **Audio Synthesizer / Suara Gong Riil pada Meja Wasit & Scoreboard**:
   - *Kondisi saat ini*: Menekan tombol Gong di konsol wasit hanya memicu animasi visual (*flash UI*), belum mengeluarkan suara audio file (.mp3 / Web Audio API) ke sound system venue.
   - *Kebutuhan*: Integrasi audio bell/gong sintetis dengan pengaturan volume dan buzzer otomatis saat waktu ronde menyentuh 00:00.
2. **Offline Resilience / Buffer Mode pada Tablet Juri**:
   - *Kondisi saat ini*: Jika koneksi Wi-Fi venue terputus sesaat, skor yang ditekan juri saat offline berisiko tidak terkirim.
   - *Kebutuhan*: Antrian lokal sementara (*offline queue*) di IndexedDB / Local Cache yang otomatis disinkronkan ulang saat koneksi WebSocket tersambung kembali.
3. **Penyusunan Ulang Antrian Partai Tanding (Drag & Drop Running Order)**:
   - *Kondisi saat ini*: Urutan nomor partai pada `ScheduleOpsView.vue` mengandalkan urutan database bawaan; belum ada mekanisme geser nomor urut partai secara visual jika panitia ingin mendahulukan partai tertentu.
   - *Kebutuhan*: Fitur drag-and-drop baris partai untuk menukar nomor partai / jadwal tanding.

### 🟡 Prioritas Menengah (P1 - Efisiensi & Kemudahan Panitia Operasional)
4. **Reseeding Manual Bagan Pertandingan Sebelum Dikunci**:
   - *Kondisi saat ini*: Bagan pertandingan di-generate acak otomatis oleh backend.
   - *Kebutuhan*: Tombol/modal untuk menukar posisi slot peserta secara manual antar bagan (misal menghindari pertemuan sesama atlet satu kontingen di babak awal / *seeding protect*) sebelum bagan resmi dikunci (*Lock Draw*).
5. **Ekspor Berkas PDF Resmi Pertandingan (Bagan & Lembar Skor)**:
   - *Kondisi saat ini*: Tombol Cetak PDF masih mengandalkan `window.print()` standar browser.
   - *Kebutuhan*: Generator berkas PDF terformat rapi (kop surat kejuaraan, bagan pohon vektor, tanda tangan digital Delegasi Teknik & Ketua Pertandingan).
6. **Import Bulk Data Atlet & Kategori via File Excel / CSV**:
   - *Kondisi saat ini*: Pendaftaran atlet dan kategori dimasukkan satu per satu melalui formulir antarmuka.
   - *Kebutuhan*: Form upload template file Excel (.xlsx / .csv) untuk mengimpor ratusan atlet kontingen sekaligus.
7. **Portal Khusus Delegasi Kontingen**:
   - *Kondisi saat ini*: Akun kontingen dapat mendaftar dan login, tetapi belum memiliki antarmuka khusus untuk memantau status atletnya sendiri secara mandiri.
   - *Kebutuhan*: Dashboard kontingen yang hanya menampilkan data kontingen tersebut (daftar atlet terdaftar, jadwal tanding kontingen mereka, dan raihan medali).

### 🟢 Prioritas Rendah / Penyempurnaan UX (P2 - Visual & Polish)
8. **Live Telemetry Ping & Battery Real-Time dari Tablet Juri**:
   - *Kondisi saat ini*: Indikator baterai dan latensi di kartu `DevicePinManager.vue` masih berupa simulasi visual.
   - *Kebutuhan*: Membaca Battery API peramban tablet juri dan mengirimkannya berkala via WebSocket ke laptop operator.
9. **Upload Foto Profil Atlet & Logo Kontingen**:
   - *Kondisi saat ini*: Atlet ditampilkan dengan avatar inisial nama.
   - *Kebutuhan*: Input upload foto atlet untuk ditampilkan pada layar Scoreboard TV LED saat atlet memasuki arena.
10. **Responsivitas Perangkat Mobile Layar Kecil pada Ops Shell**:
    - *Kondisi saat ini*: Tampilan `OpsShellView.vue` sangat nyaman pada layar laptop/desktop (1366x768 ke atas), tetapi padat jika dibuka lewat smartphone.
    - *Kebutuhan*: Tampilan *drawer responsive* khusus navigasi jika operator membuka dashboard via ponsel.
11. **Panel Sanksi & Panel Juri pada Konsol Wasit Masih Berisi Teks Tetap**:
    - *Kondisi saat ini*: Nilai awal skor, teknik, hukuman, log kejadian, dan daftar juri di `RefereeConsoleView.vue` sudah dinetralkan, tetapi template masih memuat teks tertanam (mis. "Teguran 1 (Aktif)", "LOCK 3/3 SAH", "LAN JURI OK"). Pada tablet juri (`JudgeScoringView.vue`), titik indikator konsensus (`redConsensusDots`/`blueConsensusDots`) kini kosong dan belum terhubung ke data server.
    - *Kebutuhan*: Turunkan seluruh panel tersebut dari data server (`score_update`, `penalty`, daftar juri terhubung). Backend hanya menyimpan total poin potongan, belum rincian binaan/teguran/peringatan; scoreboard saat ini menampilkan satu chip "POTONGAN".
12. **Login Tablet via QR / Tautan**:
    - *Kebutuhan*: Tombol "Buka Tablet" pada `DevicePinManager.vue` menghasilkan tautan/QR yang langsung memasukkan perangkat (tanpa mengetik PIN); PIN tetap sebagai cadangan.
13. **Label Arena/Kategori/Partai pada Scoreboard**:
    - *Kondisi saat ini*: Judul turnamen (`KEJUARAAN NASIONAL PENCAK SILAT 2025`) tertanam di `ScoreboardView.vue`, sedangkan label gelanggang, kategori, dan nomor partai berisi teks generik karena `GET /live/scoreboard` belum mengembalikannya.
    - *Kebutuhan*: Tambahkan nama turnamen, gelanggang, kategori, dan nomor partai pada snapshot backend lalu tampilkan.
14. **Format Prettier Belum Dipaksa di CI**:
    - *Kondisi saat ini*: Prettier terpasang (`npm run format:check`) tetapi kode lama belum diformat sehingga belum dijadikan syarat CI.
    - *Kebutuhan*: Jalankan `npm run format` sekali dalam commit terpisah, lalu tambahkan `format:check` ke job `frontend`.

---

## 5. Rencana Tindak Lanjut & Checklist Pengerjaan

- [ ] **Langkah 1**: Menambahkan Web Audio gong & buzzer sound effect pada `RefereeConsoleView.vue` dan `ScoreboardView.vue`.
- [ ] **Langkah 2**: Mengembangkan fungsionalitas drag & drop reorder partai pada `ScheduleOpsView.vue`.
- [ ] **Langkah 3**: Menambahkan dialog swap/reseed atlet pada bagan pertandingan sebelum penguncian bagan resmi.
- [ ] **Langkah 4**: Mengimplementasikan template impor data atlet dari spreadsheet CSV.

---

## 6. Riwayat Pembaruan Frontend (Changelog)

- **2026-10-06**:
  - Rute `/ops` kini wajib login dan guard router menegakkan `allowedRoles`; kode ganda dihapus (`lib/api`, `lib/competitionApi`, `lib/authApi`, wrapper `CompetitionsView`); `KioskLayout` dan `TabletLayout` ditambahkan.
  - Seluruh tipe `any` dihapus (helper `http` bertipe, DTO mengikuti backend, peta `WsServerMessages`); ESLint, Prettier, Vitest, dan job CI `frontend` ditambahkan.
  - Login tablet hanya dengan PIN (juri 6 digit, wasit 8 digit); pilihan arena, ID turnamen, dan sampel PIN palsu dihapus.
  - Scoreboard menampilkan data live (skor, babak, timer, jeda, nama atlet) dan tidak lagi berisi data demo; tombol MULAI RONDE wasit diperbaiki (status awal `idle`).
  - Verifikasi ke backend nyata (Postgres + simulasi e2e + Chromium) menemukan dan memperbaiki bug backend WebSocket (lihat `docs/task_update.md` butir 36-37).

- **2026-10-05**:
  - Menyelesaikan integrasi persisten PIN gelanggang dari backend database (`authApi.listPins`), mengeliminasi mock PIN di `localStorage`.
  - Menghilangkan redundansi pemilih gelanggang di Topbar (`hasArena: false`).
  - Menambahkan toggle format juri (5 Juri PB IPSI vs 3 Juri Daerah).
  - Memisahkan kartu Scoreboard TV Display sebagai layar publik bebas PIN.
  - Memperbaiki penanganan verifikasi peserta dengan tombol aksi tegas: Loloskan vs Tolak.
  - Memperbaiki proteksi foreign key ID gelanggang dengan hook `onMounted` dan auto-fallback valid arena.
  - Membuat dokumen panduan progres frontend terpadu (`docs/frontend_progress.md`).
