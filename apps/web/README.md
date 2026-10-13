# DigitalScoring Web Application (`apps/web`)

Antarmuka web modern berstandar industri untuk platform **DigitalScoring Pencak Silat (Kategori Tanding)**. Dibangun menggunakan **Vue 3**, **TypeScript** (*strict type-safety*), **Vite**, **Tailwind CSS**, **Pinia**, dan **TanStack Query**.

Dirancang untuk menangani berbagai kebutuhan antarmuka pertandingan secara *real-time*: mulai dari tampilan **Scoreboard TV** untuk penonton venue, **Bagan Turnamen Publik**, **PWA Tablet Juri** dengan input sentuh berlatensi rendah, **Konsol Operator Meja / Wasit**, hingga **Ruang Operasi Panitia (Operations Panel)** yang terpadu.

---

## 🛠️ Tech Stack & Ekosistem

| Lapisan / Komponen | Teknologi | Keterangan |
|---|---|---|
| **Core Framework** | [Vue 3](https://vuejs.org/) (v3.5+) | Composition API dengan `<script setup lang="ts">` |
| **Bahasa & Tipe** | [TypeScript](https://www.typescriptlang.org/) (v5.x / strict) | Aturan ketat tanpa `any` (`@typescript-eslint/no-explicit-any` = error) |
| **Build & Bundler** | [Vite](https://vitejs.dev/) | HMR cepat, pembagian bundel efisien, & konfigurasi proxy API |
| **Server State & Cache** | [TanStack Vue Query](https://tanstack.com/query/latest) (v5) | Sinkronisasi server state, *optimistic updates*, cache invalidation otomatis |
| **Client State** | [Pinia](https://pinia.vuejs.org/) | Manajemen sesi auth, pemilihan kompetisi aktif, & UI state global |
| **Styling & Desain** | [Tailwind CSS](https://tailwindcss.com/) + [clsx](https://github.com/lukeed/clsx) | Utilitas CSS responsif & dinamis |
| **Ikonografi & Font** | [Lucide Vue Next](https://lucide.dev/) + [Fontsource](https://fontsource.org/) | Font *Space Grotesk* & *JetBrains Mono* di-host lokal (offline-first) |
| **Skema & Validasi** | [Zod](https://zod.dev/) | Validasi tipe runtime untuk kontrak API dan form input |
| **Protokol Real-Time** | Native WebSocket | WebSocket client dengan auto-reconnect, heartbeat, & event dispatching |
| **PWA & Offline** | [Vite Plugin PWA](https://vite-pwa-org.netlify.app/) (Workbox) | Offline shell, manifest layar penuh, & service worker caching |
| **Pengujian (Testing)** | [Vitest](https://vitest.dev/), [Cypress](https://www.cypress.io/), [Playwright](https://playwright.dev/) | Unit test, integrasi guard router, hingga simulasi E2E multi-jendela |

---

## 🖥️ Arsitektur Antarmuka (*Multi-Surface Architecture*)

Aplikasi ini melayani 5 kelompok antarmuka utama dengan tata letak (*layout*) dan hak akses (*RBAC*) yang disesuaikan:

```
                          ┌─── /scoreboard/:arena_id  ─── Scoreboard TV (Kiosk Mode)
                          ├─── /bagan/:comp/:category ─── Bagan Turnamen Publik
[DigitalScoring Web] ─────┼─── /judge                 ─── Tablet Juri Sudut (PWA Touch)
                          ├─── /referee               ─── Konsol Operator Meja / Wasit
                          └─── /ops                   ─── Ruang Operasi Panitia (Dashboard)
```

### 1. Scoreboard TV & Live Display (`/scoreboard`, `/scoreboard/:arena_id`)
* **Layout Kiosk**: Tampilan layar penuh tanpa *scroll*, kontras tinggi, dan tipografi angka yang terbaca jelas dari jarak jauh venue pertandingan.
* **Sinkronisasi Real-Time**: Terhubung langsung ke WebSocket gelanggang untuk memperbarui skor sudut merah/biru, nomor babak, hitung mundur timer (*server-side authoritative*), serta sanksi teguran/peringatan.
* **Modal Kemenangan Otomatis**: Menampilkan banner pemenang secara otomatis saat partai berakhir (kemenangan angka, KO, TKO, WO, Diskualifikasi, atau WMP).

### 2. Bagan Publik Interaktif (`/bagan`, `/bagan/:comp_id/:category_id`)
* **Tree Visualizer Bracket**: Visualisasi pohon bagan pertandingan sistem gugur tunggal (*single-elimination*) dengan penanganan otomatis peserta *BYE*.
* **Akses Publik Bebas**: Dapat diakses langsung oleh atlet, ofisial kontingen, dan penonton melalui ponsel pintar tanpa perlu login.
* **Navigasi Cepat**: Filter bagan terkelompok menurut golongan umur dan kelas tanding.

### 3. Tablet Juri PWA (`/judge`, `/judge/login`)
* **Login Cepat PIN-Only**: Layar login sentuh khusus PIN 6 digit unik yang langsung mengenali kejuaraan, gelanggang, dan kursi juri (Juri 1-5).
* **Antarmuka Sentuh Horizontal**: Tombol input skor besar (Pukulan +1, Tendangan +2, Jatuhan +3) untuk meminimalkan salah tekan saat pertandingan berlangsung intens.
* **Indikator Konsensus & Timer**: Dilengkapi lampu konsensus dan status waktu pertandingan yang sinkron dengan operator meja.

### 4. Konsol Operator Meja / Wasit (`/referee`)
* **Kendali Waktu**: Pengendalian timer pertandingan (*Start*, *Pause*, *Resume*, *Selesai Babak*) dengan bunyi gong audio otomatis.
* **Panel Sanksi**: Pemberian dan pembatalan sanksi bertingkat (Teguran I/II, Peringatan I/II/III, Pelanggaran Berat, Diskualifikasi).
* **Keputusan Akhir Pertandingan**: Eksekusi keputusan akhir pertandingan termasuk hasil khusus (KO, TKO, WO, Undur Diri, dan Keputusan Wasit).

### 5. Ruang Operasi Panitia (`/ops`)
Pusat kendali kejuaraan terpadu untuk peran **Admin**, **Dewan Juri**, **Operator Gelanggang**, dan **Manajer Kontingen**:
* 🏆 **Kompetisi**: Manajemen data turnamen, aktivasi kejuaraan, dan pengaturan umum.
* 🥋 **Kategori & Kelas**: Preset template resmi **PB IPSI** (Usia Dini, Pra-Remaja, Remaja, Dewasa, Master) dan tabel kelas bobot tanding resmi.
* ⚖️ **Meja Verifikasi Timbang**: Modul penimbangan atlet tanpa toleransi, pencatatan status keabsahan, dan checklist berkas (NIK, Akta Kelahiran, Surat Sehat).
* 👥 **Peserta & Atlet**: Pendaftaran atlet, penugasan kontingen, perbaikan data, dan mutasi kelas.
* 🌲 **Bagan Turnamen**: Pembentukan bagan otomatis dari atlet yang telah lolos verifikasi timbang dan penegakan kunci draw (*bracket lock*).
* ⏱️ **Jadwal Pertandingan**: Antrean partai per gelanggang, status kesiapan partai, dan pemuatan partai ke arena aktif.
* 🔑 **Perangkat & PIN**: Monitoring status kehadiran WebSocket perangkat juri/wasit secara *live*, auto-provision PIN, serta rotasi PIN darurat.
* 📜 **Regulasi & Ruleset**: Editor ruleset dinamis (IPSI 2024 / PERSILAT 2026), durasi ronde/istirahat per golongan umur, dan aturan ambang selisih poin (WMP).
* 👥 **Akun Petugas & Kontingen**: Manajemen pengguna internal dengan kontrol akses berbasis peran (RBAC).
* 📊 **Laporan & Rekap**: Rekapitulasi perolehan medali kontingen secara instan dan ekspor data kejuaraan.

---

## 📂 Struktur Direktori Proyek

```text
apps/web/
├── public/                 # Favicon, ikon PWA, manifest, & aset statis publik
├── scripts/                # Skrip pengujian simulasi WebSocket & API (e2e-simulation.mjs)
├── src/
│   ├── api/                # HTTP client envelope, Axios instance, error handling helper
│   ├── assets/             # Global stylesheets (Tailwind, typography, themes)
│   ├── components/         # Komponen UI desain sistem (AppButton, AppModal, AppBadge, dll.)
│   ├── features/           # Arsitektur modular berbasis fitur domain:
│   │   ├── arena/          # Manajemen arena, kehadiran WebSocket, & status gelanggang
│   │   ├── athletes/       # Data atlet, pendaftaran, & meja verifikasi timbang
│   │   ├── auth/           # Layanan autentikasi, JWT handler, & helper validasi PIN
│   │   ├── categories/     # Golongan umur, kelas tanding, & template IPSI/PERSILAT
│   │   ├── competitions/   # Manajemen turnamen & kompetisi aktif
│   │   ├── matches/        # Jadwal partai, pemuatan gelanggang, & skor pertandingan
│   │   ├── officials/      # Akun panitia, hak akses peran, & dialog profil
│   │   ├── reports/        # Rekap medali, ringkasan skor, & ekspor laporan
│   │   ├── rulesets/       # Editor regulasi, timing babak, & konfigurasi WMP
│   │   └── tournament/     # Generator bagan sistem gugur & Tree Bracket visualizer
│   ├── layouts/            # Layout shells (DefaultLayout, KioskLayout, TabletLayout)
│   ├── router/             # Vue Router dengan RBAC navigation guard
│   ├── stores/             # Pinia state stores (auth, settings)
│   ├── views/              # Route page components (Scoreboard, Bagan, Judge, Referee, Ops, Auth)
│   ├── App.vue             # Root component dengan Toast container & Dialog host
│   └── main.ts             # Application entrypoint & konfigurasi plugin
├── test/                   # Direktori suite pengujian end-to-end
│   ├── cypress/            # Cypress E2E specs (registrasi, timbang, bagan, flow pertandingan)
│   └── playwright/         # Playwright specs (simulasi juri + wasit multi-jendela, PWA)
├── vite.config.ts          # Konfigurasi bundler Vite, path alias, proxy, & PWA plugin
└── package.json            # Daftar dependensi dan script npm
```

---

## 🚀 Panduan Memulai (*Getting Started*)

### 1. Prasyarat Sistem
* [Node.js](https://nodejs.org/) v20.x atau v24.x LTS
* Pengelola paket `npm` atau `pnpm`
* Backend DigitalScoring berjalan di `http://localhost:8080` (opsional untuk dev UI murni)

### 2. Instalasi Dependensi
Jalankan dari direktori `apps/web`:

```bash
npm install
```

### 3. Menjalankan Server Pengembangan (Dev Mode)
```bash
npm run dev
```
Aplikasi akan tersedia di `http://localhost:5173`. Vite secara otomatis mem-proxy panggilan `/api` dan WebSocket `/ws` ke backend di `http://localhost:8080`.

---

## 📜 Daftar Perintah (*NPM Scripts*)

| Perintah | Deskripsi |
|---|---|
| `npm run dev` | Menjalankan Vite development server dengan HMR |
| `npm run build` | Menjalankan typecheck (`vue-tsc -b`) dan memproduksi bundel produksi ke `dist/` |
| `npm run preview` | Menjalankan pratinjau lokal dari hasil build produksi |
| `npm run typecheck` | Menjalankan validasi tipe TypeScript di seluruh berkas `.ts` dan `.vue` |
| `npm run lint` | Menjalankan ESLint untuk mengecek kepatuhan kode |
| `npm run lint:fix` | Memperbaiki kesalahan gaya kode yang dapat diperbaiki otomatis oleh ESLint |
| `npm run format` | Memformat seluruh kode sumber di `src/` menggunakan Prettier |
| `npm run format:check` | Memeriksa format kode tanpa mengubah berkas |
| `npm test` | Menjalankan unit & component test berbasis **Vitest** |
| `npm run cypress:run` | Menjalankan pengujian E2E headless via Cypress |
| `npm run cypress:open` | Membuka interactive GUI runner Cypress |
| `npm run playwright:pw` | Menjalankan seluruh pengujian E2E headless via Playwright |
| `npm run playwright:pw:ui` | Membuka Playwright Test Runner dengan UI interaktif & tracing |
| `npm run playwright:pw:headed` | Menjalankan simulasi Playwright berjendela (grid multi-perangkat) |
| `npm run playwright:pw:pwa` | Menguji kriteria instalasi dan service worker PWA |

---

## 📡 Integrasi & Pengujian Terpadu

### Simulasi E2E Terhadap Backend
Untuk menguji aliran pertandingan lengkap secara terotomasi terhadap backend Go nyata:

```bash
node scripts/e2e-simulation.mjs
```

Skrip ini memverifikasi:
* Login PIN juri/wasit tanpa kredensial email.
* Koneksi WebSocket (penonton publik, juri terautentikasi, dan pembatasan isolasi arena).
* Konsensus input skor mayoritas (2 dari 3 juri).
* Jeda (*pause*), lanjutkan (*resume*), dan sanksi.
* Integritas data live snapshot pada endpoint REST `/live/scoreboard/{arena_id}`.

### PWA di Jaringan Lokal Venue (Tanpa Internet)
Aplikasi ini mendukung operasional penuh di jaringan LAN tertutup arena olahraga:
* **Font Mandiri**: Seluruh font diunduh dari paket lokal `@fontsource`, bukan dari CDN eksternal.
* **Service Worker**: Modul PWA meng-cache cangkang aplikasi sehingga tetap responsif jika koneksi Wi-Fi arena berfluktuasi.
* **HTTPS Caddy**: Untuk panduan penerbitan sertifikat SSL lokal agar tablet Android/iPad mengizinkan pemasangan PWA, lihat dokumentasi di [`docs/PWA.md`](../../docs/PWA.md).

---

## 📖 Rujukan Dokumentasi Terkait

* Kontrak REST API & Protokol WebSocket: [`docs/openapi.yaml`](../../docs/openapi.yaml)
* Panduan Menjalankan Semua Skenario Uji: [`docs/TESTING.md`](../../docs/TESTING.md)
* Kredensial Uji Coba & Format PIN: [`docs/CREDENTIALS.md`](../../docs/CREDENTIALS.md)
* Panduan Setup PWA & HTTPS Lokal: [`docs/PWA.md`](../../docs/PWA.md)
* Panduan Regulasi & Kategori IPSI: [`docs/rencana_kategori_dan_registrasi_ipsi.md`](../../docs/rencana_kategori_dan_registrasi_ipsi.md)
