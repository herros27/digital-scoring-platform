# MASTER PROMPT & ARSITEKTUR DESAIN FRONTEND DIGITALSCORING PENCAK SILAT

> **Tujuan Dokumen:**  
> Dokumen ini merupakan *Engineering Master Prompt & Technical Architecture Specification* yang lengkap, presisi, dan komprehensif untuk pengembangan seluruh modul Frontend aplikasi **DigitalScoring Pencak Silat (Kategori Tanding)**.  
> Seluruh kontrak endpoint REST, protokol real-time WebSocket, alur autentikasi multi-peran, skema data TypeScript, dan antarmuka pengguna (UI/UX) di dalam dokumen ini **100% selaras dengan implementasi backend Go** yang terdapat pada `architecture-backend.md` dan `docs/openapi.yaml`.

---

## 1. Ringkasan Eksekutif & Karakteristik Sistem

Aplikasi Frontend DigitalScoring dirancang untuk menangani pertandingan pencak silat profesional dengan keandalan tinggi (*zero-fail*), latensi minimal (*sub-200ms latency*), dan multi-arena paralel.

### Pilar Kunci Frontend:
1. **Multi-Role & Multi-Interface**:
   - **Admin / Dewan / Operator Dashboard**: Manajemen turnamen, verifikasi berkas/timbang badan, pembuatan bagan, pengaturan gelanggang, manajemen PIN juri, dan laporan.
   - **Contingent Portal (Manajer Kontingen)**: Registrasi mandiri kontingen, pendaftaran atlet ke kategori tanding, dan pemantauan status verifikasi berkas.
   - **Judge Tablet PWA**: Aplikasi tablet touch-first berlatensi rendah untuk juri sudut gelanggang. Mencegah bias (juri tidak melihat input juri lain), mendukung pencegahan duplikasi (*idempotency*), dan tangguh terhadap *network drop*.
   - **Referee (Wasit) Control Pad**: Konsol wasit gelanggang untuk kendali timer/ronde (*start*, *pause*, *resume*, *finish*, *cancel*) dan penjatuhan teguran/peringatan/hukuman.
   - **Scoreboard Display Gelanggang**: Tampilan fullscreen resolusi tinggi untuk layar penonton / LED arena / broadcast stream tanpa autentikasi (publik).
   - **Public Tournament Hub**: Portal penonton untuk melihat klasemen medali, jadwal pertandingan, dan visualisasi pohon bagan (*bracket tree*).
2. **Server-Authoritative Real-Time Architecture**:
   - Timer pertandingan dihitung dan dikendalikan murni di backend (Go ticker). Frontend melakukan interpolasi lokal mulus berbasis paket `timer_sync` dan `state_change`.
   - Seluruh penilaian juri dikumpulkan ke dalam *Sliding-Window Consensus Engine* di backend (kuorum 2/3 juri). Frontend juri hanya mengirimkan tap teknik dan menerima konfirmasi lokal.
3. **Resiliensi & Idempotensi Jaringan**:
   - Setiap ketukan juri menghasilkan *UUID v4* sebagai `idempotency_key` (mencegah penambahan poin ganda jika juri menekan tombol berulang kali saat sinyal tersendat).
   - *Offline buffer queue* dan auto-reconnect dengan `sync_request` untuk pemulihan instan saat koneksi terputus.

---

## 2. Tech Stack & Standar Frontend

Frontend berada di dalam folder `apps/web/` dengan spesifikasi teknologi:

- **Framework**: [Vue 3](https://vuejs.org/) (Composition API, `<script setup lang="ts">`)
- **Language**: [TypeScript](https://www.typescriptlang.org/) (Strict Mode)
- **Build Tool**: [Vite 8](https://vitejs.dev/)
- **Routing**: [Vue Router 5](https://router.vuejs.org/)
- **State Management**: [Pinia 4](https://pinia.vuejs.org/)
- **Styling**: Modern CSS / Tailwind CSS / Utility classes yang responsif
- **Iconography**: Lucide Icons (`lucide-vue-next`)
- **PWA Capabilities**: PWA Manifest, Service Worker offline caching untuk tablet juri & scoreboard
- **Protokol Real-Time**: Native WebSocket Browser API (`new WebSocket(...)`) dengan reconnect backoff otomatis

---

## 3. Matriks Hak Akses & Tiga Alur Autentikasi (Auth Flows)

Backend menyediakan 3 mekanisme autentikasi berbeda yang harus ditangani oleh modul `useAuth` pada frontend:

| Flow / Aktor | Endpoint Login | Kredensial | Tipe Token & Payload | Masa Berlaku | Destinasi Antarmuka |
|---|---|---|---|---|---|
| **Official Auth** (Admin, Dewan, Operator, Wasit) | `POST /api/v1/auth/login` | Email & Password | Bearer JWT (claims: `sub`, `role: "admin" \| "dewan" \| "operator" \| "wasit"`) | 24 Jam | `/admin/*` & `/referee/*` |
| **Judge & Wasit PIN Auth** (Tablet Juri / Meja Wasit) | `POST /api/v1/auth/judge-login` | `pin` saja (juri 6 digit, wasit 8 digit; server menentukan turnamen, gelanggang, role, kursi) | Scoped JWT (claims: `pin_id`, `comp_id`, `arena_id`, `role: "judge"`, `seat_number: 1..N`) | Sesuai TTL PIN (default 12 jam) | `/judge/*` (Tablet Scoring PWA) |
| **Contingent Auth** (Manajer Tim Kontingen) | `POST /api/v1/auth/contingent/login` | Email & Password | Bearer JWT (claims: `sub`, `role: "contingent"`) | 24 Jam | `/contingent/*` (Portal Pendaftaran) |
| **Scoreboard & Publik** | *Tanpa Auth* | Publik | Tidak memerlukan token (akses live feed) | - | `/scoreboard/:arena_id` & `/live/*` |

### Penanganan Token pada WebSocket:
Browser WebSocket API tidak mendukung custom header `Authorization: Bearer <token>` saat *handshake*.  
Frontend **wajib** mengirimkan token melalui Query Parameter:
```
wss://<HOST>/ws/arena/{arena_id}?token=<JWT_TOKEN>
```

---

## 4. Desain Struktur Folder & Modularitas Frontend

Susunan kode di dalam `apps/web/src/` dirancang modular mengikuti prinsip Clean Architecture:

```
apps/web/src/
├── assets/                  # CSS global, font, logo pencak silat, sound effects (buzzer, bell)
│   └── main.css
├── components/              # Reusable Shared UI Components
│   ├── common/              # Button, Modal, Badge, Card, Input, Toast, Alert
│   ├── bracket/             # SingleEliminationTree.vue, BracketNode.vue, MatchCard.vue
│   ├── scoreboard/          # DigitalTimer.vue, CornerScoreCard.vue, RoundBadge.vue, WinnerBanner.vue
│   └── judge/               # TouchScoreButton.vue, HapticFeedback.vue, ConnectionStatus.vue
├── composables/             # Stateful Logic Composables
│   ├── useAuth.ts           # Manajemen login 3 alur, JWT storage, role checker
│   ├── useWebSocket.ts      # Koneksi arena WS, heartbeat, auto-reconnect, message routing
│   ├── useTimer.ts          # Sinkronisasi countdown timer server dengan interpolasi lokal
│   └── useHaptics.ts        # Navigator.vibrate untuk tablet juri saat tombol ditekan
├── layouts/                 # Master Layouts
│   ├── AdminLayout.vue      # Sidebar navigasi, topbar profil, status arena mini
│   ├── ContingentLayout.vue # Header portal kontingen, tab atlet & medali
│   ├── JudgeLayout.vue      # Fullscreen landscape kuncian layar (orientation lock)
│   └── ScoreboardLayout.vue # High-contrast broadcast layout (dark stadium theme)
├── lib/                     # Utilitas & Network Client
│   ├── api.ts               # Fetch client pembungkus REST API, auth header, & error handler
│   └── uuid.ts              # Pembangkit UUID v4 untuk idempotency key
├── router/                  # Definisi rute navigasi & Navigation Guards
│   └── index.ts
├── stores/                  # Pinia Global State Stores
│   ├── auth.ts              # State user, token, role, dan seat_number
│   ├── competition.ts       # Cache data kejuaraan aktif
│   ├── match.ts             # State pertandingan aktif, scores, rounds, dan status
│   └── arena.ts             # Status gelanggang & koneksi WebSocket
├── types/                   # TypeScript Contract Interfaces (Mirror Go Domain)
│   ├── auth.ts
│   ├── competition.ts
│   ├── match.ts
│   ├── ruleset.ts
│   ├── bracket.ts
│   └── websocket.ts
└── views/                   # Tampilan Halaman Per Modul
    ├── admin/               # Modul Admin / Dewan
    │   ├── CompetitionsView.vue
    │   ├── CompetitionDetailView.vue
    │   ├── ArenasView.vue
    │   ├── AthletesView.vue
    │   ├── ParticipantsVerificationView.vue
    │   ├── MatchesScheduleView.vue
    │   ├── BracketManagerView.vue
    │   ├── RulesetsView.vue
    │   ├── OfficialsAndPinsView.vue
    │   └── ReportsView.vue
    ├── contingent/          # Modul Manajer Kontingen
    │   ├── ContingentRegisterView.vue
    │   ├── ContingentLoginView.vue
    │   ├── ContingentDashboardView.vue
    │   └── AthleteRegistrationView.vue
    ├── judge/               # Modul Juri Gelanggang (PWA)
    │   ├── JudgePinLoginView.vue
    │   └── JudgeTabletScoringView.vue
    ├── referee/             # Modul Wasit Pertandingan
    │   └── RefereeConsoleView.vue
    ├── scoreboard/          # Modul Layar Scoreboard Publik
    │   └── LiveScoreboardView.vue
    └── public/              # Modul Penonton / Portal Turnamen
        ├── PublicMedalStandingsView.vue
        └── PublicBracketView.vue
```

---

## 5. Spesifikasi Lengkap Kontrak REST API Backend

Seluruh panggilan REST menggunakan wrapper pada `src/lib/api.ts`. Response format standar backend:
- **Sukses**: `{ "success": true, "data": T, "meta"?: { "page": 1, "per_page": 20, "total": 100 } }`
- **Gagal**: `{ "success": false, "error": { "code": string, "message": string, "details"?: any } }`

### 5.1. Authentication & PIN Management
1. **`POST /api/v1/auth/login`**
   - **Tujuan**: Login Official (Admin, Dewan, Operator, Wasit).
   - **Request**: `{ "email": "admin@silat.id", "password": "test_password_1" }`
   - **Response**: `{ "token": string, "user": { "id": uuid, "name": string, "email": string, "role": string } }`
2. **`POST /api/v1/auth/judge-login`**
   - **Tujuan**: Login tablet juri (PIN 6 digit) atau meja wasit (PIN 8 digit). Tidak perlu memilih gelanggang atau mengisi ID turnamen.
   - **Request**: `{ "pin": "123456" }`
   - **Response**: `{ "token": string, "role": "judge" | "wasit", "seat_number"?: number, "arena_id": uuid, "competition_id": uuid }`
   - **Error**: `401 INVALID_PIN`, `429 TOO_MANY_ATTEMPTS` (+ `Retry-After`)
3. **`POST /api/v1/auth/create-pin`** *(Protected: Admin / Dewan)*
   - **Tujuan**: Membuat PIN akses baru untuk juri/wasit gelanggang.
   - **Request**: `{ "competition_id": uuid, "arena_id": uuid, "role": "juri" | "wasit", "seat_number": 1..5, "pin"?: string, "ttl_hours"?: 12 }`
   - **Response**: `{ "id": uuid, "competition_id": uuid, "arena_id": uuid, "role": string, "pin": string, "seat_number": number, "expires_at": string }`
4. **`POST /api/v1/auth/contingent/register`** *(Publik)*
   - **Tujuan**: Pendaftaran akun mandiri untuk manajer kontingen/perguruan.
   - **Request**: `{ "name": "Tapak Suci Sleman", "pic_name": "Budi Santoso", "email": "ts@event.com", "phone": "081234567890", "password": "SecretPassword!" }`
   - **Response 201**: `{ "success": true, "data": { ... } }`
5. **`POST /api/v1/auth/contingent/login`** *(Publik)*
   - **Tujuan**: Login akun manajer kontingen.
   - **Request**: `{ "email": "ts@event.com", "password": "SecretPassword!" }`
   - **Response**: `{ "token": string, "contingent": { "id": uuid, "name": string, "email": string } }`

---

### 5.2. Competition Lifecycle & Categories
1. **`GET /api/v1/competitions`**
   - **Query Params**: `?page=1&limit=20&status=active`
   - **Response**: Paged list of `Competition` (`id`, `name`, `location`, `start_date`, `end_date`, `ruleset_name`, `status`).
2. **`POST /api/v1/competitions`** *(Admin / Dewan)*
   - **Request**: `{ "name": string, "location": string, "start_date": "YYYY-MM-DD", "end_date": "YYYY-MM-DD", "ruleset_id": uuid }`
3. **`GET /api/v1/competitions/{id}`**
   - **Response**: Detail kompetisi termasuk status (`draft`, `active`, `completed`, `cancelled`).
4. **`PUT /api/v1/competitions/{id}`** *(Admin / Dewan)*
   - **Request**: Perubahan nama, lokasi, tanggal, dan ruleset.
5. **Aksi Transisi Lifecycle Kompetisi** *(Admin / Dewan)*:
   - `POST /api/v1/competitions/{id}/activate` (Mengaktifkan turnamen)
   - `POST /api/v1/competitions/{id}/pause` (Menjeda turnamen)
   - `POST /api/v1/competitions/{id}/finish` (Menyelesaikan turnamen)
   - `POST /api/v1/competitions/{id}/cancel` (Membatalkan turnamen)
6. **`GET /api/v1/competitions/{comp_id}/categories`** & **`POST ...`**
   - Mengelola kelas tanding (contoh: Kategori Tanding Dewasa Putra Kelas A: 45kg-50kg).
   - Fields: `name`, `gender` (`male`/`female`), `weight_class`, `ruleset_id`.

---

### 5.3. Atlet, Kontingen & Verifikasi Peserta (Participants)
1. **`GET /api/v1/athletes`** & **`POST /api/v1/athletes`**
   - Master data atlet (`name`, `gender`, `team`, `contingent_id`).
   - Dapat diinput oleh Admin, Dewan, atau Kontingen.
2. **`POST /api/v1/competitions/{comp_id}/athletes`**
   - Mendaftarkan atlet ke kategori kejuaraan tertentu.
3. **`GET /api/v1/competitions/{comp_id}/participants`**
   - **Query**: `?status=pending|verified|rejected`
   - List peserta kejuaraan yang menunggu verifikasi timbang badan & berkas fisik.
4. **`PUT /api/v1/competitions/{comp_id}/participants/{id}/verify`** *(Admin / Dewan)*
   - **Request**: `{ "registration_status": "verified" | "rejected" }`
   - **Krusial**: Hanya peserta berstatus `verified` yang dapat dimasukkan ke dalam generator bagan pertandingan.
5. **`GET /api/v1/contingents`** & **`GET /api/v1/contingents/{id}`**
   - Master data kontingen / perguruan yang terdaftar.

---

### 5.4. Gelanggang (Arenas)
1. **`GET /api/v1/competitions/{comp_id}/arenas`**
   - List gelanggang (Arena A, Arena B, Arena C).
2. **`POST /api/v1/competitions/{comp_id}/arenas`**
   - Buat gelanggang baru (`name: "Gelanggang 1"`).
3. **`GET /api/v1/competitions/{comp_id}/arenas/{id}`**
4. **`PUT /api/v1/competitions/{comp_id}/arenas/{id}`**
5. **`DELETE /api/v1/competitions/{comp_id}/arenas/{id}`**

---

### 5.5. Pertandingan (Matches) & Live Controls
1. **`GET /api/v1/competitions/{comp_id}/matches`**
   - **Query**: `?page=1&limit=20&status=idle|ready|running|paused|finished|cancelled`
2. **`POST /api/v1/competitions/{comp_id}/matches`** *(Admin / Dewan)*
   - Menjadwalkan laga baru.
   - Body: `{ "arena_id": uuid, "athlete_red_id": uuid, "athlete_blue_id": uuid, "bracket_slot_id"?: uuid }`
3. **`POST /api/v1/competitions/{comp_id}/matches/{id}/load-arena`** *(Admin, Dewan, Wasit)*
   - **Body**: `{ "arena_id": uuid, "ruleset_path"?: string }`
   - **Penanganan Error Khusus Frontend**:
     - Jika gelanggang sedang dipakai oleh partai lain yang aktif, backend mengembalikan status **HTTP 409 Conflict** (`error.code: "ARENA_BUSY"`).  
     - Frontend **wajib** menangkap error ini dan menampilkan modal konfirmasi penolakan dengan pesan: *"Gelanggang sedang digunakan untuk partai aktif lain! Selesaikan partai sebelumnya terlebih dahulu."*
4. **Kontrol Pertandingan REST (Alternatif Wasit/Admin)**:
   - `POST .../matches/{id}/start` (`status: "running"`)
   - `POST .../matches/{id}/pause` (`status: "paused"`)
   - `POST .../matches/{id}/resume` (`status: "running"`)
   - `POST .../matches/{id}/finish` (`status: "finished"` -> otomatis mengevaluasi pemenang & memajukan bagan turnamen)
   - `POST .../matches/{id}/cancel` (`status: "cancelled"`)
5. **`GET /api/v1/competitions/{comp_id}/matches/{id}/scores`**
   - Mengambil breakdown skor juri (per kursi/seat_number 1..5), agregasi konsensus, dan riwayat skor per ronde.
6. **`POST /api/v1/competitions/{comp_id}/matches/{id}/scores`** (Fallback REST Scoring Juri)
   - Mendukung header `Idempotency-Key: <UUID>`
   - Body: `{ "judge_id": uuid, "score_type": "technique" | "knockdown" | "fall", "target": "red" | "blue" }`
7. **`GET /api/v1/competitions/{comp_id}/matches/{id}/events`**
   - Riwayat audit trail event sourcing pertandingan (semua input juri, teguran, dan point confirmation).

---

### 5.6. Bagan Turnamen (Brackets)
1. **`POST /api/v1/competitions/{comp_id}/brackets/generate`** *(Admin / Dewan)*
   - **Request**: `{ "category_id": uuid, "weight_class": string, "gender": "male" | "female" }`
   - **Logika**: Backend menyaring peserta yang berstatus `verified`, menghitung BYE (pangkat 2 terdekat), dan menginstansiasi struktur bagan single-elimination.
2. **`GET /api/v1/competitions/{comp_id}/brackets`**
   - **Query**: `?weight_class=Kelas A&gender=male`
   - **Response**: Struktur pohon bagan lengkap (`BracketSlot[]`) berisi `round`, `slot_index`, `participant_id`, `is_bye`, `winner_id`, `next_slot_id`.
   - **Catatan Caching**: Endpoint ini memiliki in-memory cache Otter 10 detik di backend, yang otomatis di-invalidasi ketika pertandingan selesai atau bagan baru di-generate.

---

### 5.7. Rulesets Pertandingan
1. **`GET /api/v1/rulesets`**
   - Mengambil daftar aturan penilaian yang tersedia (`IPSI 2024`, `PERSILAT v7`, atau custom).
2. **`POST /api/v1/rulesets`** *(Admin / Dewan)*
   - Menambahkan ruleset kustom berbasis JSON.
   - Konfigurasi mencakup: `round_duration` (detik), `break_duration` (detik), `technique_points` (+1), `knockdown_points` (+2), `fall_points` (+3), dan batas diskualifikasi.
3. **`GET /api/v1/rulesets/{id}`**, **`PUT ...`**, **`DELETE ...`**

---

### 5.8. Laporan & Statistik Turnamen (Cached Endpoints)
1. **`GET /api/v1/competitions/{comp_id}/report/summary`** (TTL: 10s)
   - Ringkasan total gelanggang, total atlet, total kategori, dan statistik status partai.
2. **`GET /api/v1/competitions/{comp_id}/report/medals`** (TTL: 10s)
   - Klasemen medali per kontingen (`team`, `gold`, `silver`, `bronze`, `total`).
3. **`GET /api/v1/competitions/{comp_id}/report/export`** (TTL: 10s)
   - Full dump ekspor turnamen JSON untuk keperluan arsip/backup.

---

### 5.9. Live Scoreboard Publik (REST Feed)
1. **`GET /api/v1/live/scoreboard/{arena_id}`** *(Publik / Bebas Auth)*
   - Snapshot data live scoring arena saat ini. Digunakan oleh Scoreboard untuk rendering instan awal sebelum koneksi WebSocket terpasang.

---

## 6. Protokol Real-Time WebSocket (`/ws/arena/{arena_id}`)

### 6.1. Koneksi & Handshake
- **URL**: `wss://<HOST>/ws/arena/{arena_id}?token=<JWT>`
- **Token**: Wajib untuk juri, wasit, dan admin. Scoreboard publik dapat terhubung untuk menerima siaran live.

---

### 6.2. Kontrak Pesan Client ➔ Server

1. **Input Penilaian Juri (`judge_input`)**:
   Dikirimkan segera saat juri menyentuh tombol nilai di tablet.
   ```json
   {
     "type": "judge_input",
     "payload": {
       "event_id": "7f8b9a2c-1234-4a5b-bc67-890abcdef123",
       "idempotency_key": "7f8b9a2c-1234-4a5b-bc67-890abcdef123",
       "score_type": "technique",
       "target": "red",
       "round": 1
     }
   }
   ```
   - `score_type`: `"technique"` (Pukulan, +1 poin), `"knockdown"` (Tendangan, +2 poin), `"fall"` (Bantingan/Jatuhan sah, +3 poin).
   - `target`: `"red"` atau `"blue"`.
   - `round`: Nomor ronde berjalan (1, 2, atau 3).
   - `idempotency_key`: UUID v4 unik yang dibangkitkan pada sisi browser tablet.

2. **Kendali Pertandingan Wasit (`state_change`)**:
   Dikirimkan oleh Wasit atau Admin untuk mengatur jalannya pertandingan.
   ```json
   {
     "type": "state_change",
     "payload": {
       "action": "start"
     }
   }
   ```
   - `action`: `"start"`, `"pause"`, `"resume"`, `"finish"`, `"cancel"`.

3. **Penjatuhan Hukuman Wasit (`penalty`)**:
   Dikirimkan saat wasit memberikan teguran, peringatan, atau potongan nilai.
   ```json
   {
     "type": "penalty",
     "payload": {
       "target": "blue",
       "level": 1
     }
   }
   ```
   - `target`: `"red"` atau `"blue"`.
   - `level`: Jumlah poin potongan (contoh: 1 untuk Teguran I/II, 5 untuk Peringatan I, 10 untuk Peringatan II).

4. **Permintaan Re-sinkronisasi (`sync_request`)**:
   Dikirimkan oleh klien saat baru tersambung atau pulih dari diskoneksi.
   ```json
   {
     "type": "sync_request"
   }
   ```

---

### 6.3. Kontrak Pesan Server ➔ Client (Broadcast Siaran)

1. **Balasan Status Lengkap (`sync_response`)**:
   Diterima saat inisiasi atau setelah mengirimkan `sync_request`.
   ```json
   {
     "type": "sync_response",
     "payload": {
       "match_id": "482ee982-dccc-4f27-8ba4-f9672d1827bb",
       "arena_id": "c48ee982-dccc-4f27-8ba4-f9672d1827bb",
       "state": "round_1",
       "round": 1,
       "timer_remaining_ms": 118500,
       "timer_running": true,
       "scores": [
         { "RedTechnical": 3, "BlueTechnical": 2, "RedPenalties": 0, "BluePenalties": 0 },
         { "RedTechnical": 0, "BlueTechnical": 0, "RedPenalties": 0, "BluePenalties": 0 },
         { "RedTechnical": 0, "BlueTechnical": 0, "RedPenalties": 0, "BluePenalties": 0 }
       ],
       "red_total": 3,
       "blue_total": 2,
       "server_time": "2026-10-04T08:30:00Z"
     }
   }
   ```

2. **Perubahan Status & Timer (`state_change`)**:
   ```json
   {
     "type": "state_change",
     "payload": {
       "state": "round_1",
       "round": 1,
       "timer_remaining_ms": 120000
     }
   }
   ```

3. **Pembaruan Agregasi Nilai Ronde (`score_update`)**:
   ```json
   {
     "type": "score_update",
     "payload": {
       "round": 1,
       "red_technical": 3,
       "blue_technical": 2,
       "red_penalties": 0,
       "blue_penalties": 1
     }
   }
   ```

4. **Pemberitahuan Poin Sah Konsensus (`score_confirmed`)**:
   Muncul ketika kuorum minimal juri (misal 2 dari 3 juri) tercapai dalam sliding window (~1.5 detik).
   ```json
   {
     "type": "score_confirmed",
     "payload": {
       "score_type": "knockdown",
       "target": "red",
       "points": 2,
       "count": 2,
       "round": 1
     }
   }
   ```

5. **Hasil Akhir Laga (`match_result`)**:
   ```json
   {
     "type": "match_result",
     "payload": {
       "winner": "red",
       "corner": "red",
       "result_type": "point_win",
       "scores": [ ... ]
     }
   }
   ```

6. **Notifikasi Error Server (`error`)**:
   ```json
   {
     "type": "error",
     "payload": {
       "code": "INVALID_STATE_TRANSITION",
       "message": "cannot pause match in ready state"
     }
   }
   ```

---

## 7. Desain Spesifikasi Setiap Modul Antarmuka (Views)

---

### MODUL 1: Judge Tablet PWA (`/judge/scoring`)
Antarmuka sentuh khusus tablet berposisi horizontal (*Landscape Mode*) untuk juri di sisi gelanggang.

#### Kebutuhan UX & Perilaku:
1. **Layar Login PIN Cepat**:
   - Tampilan numpad besar untuk memasukkan 6-digit PIN.
   - Pilihan gelanggang (Arena A, B, C) dan kejuaraan.
   - Saat berhasil login, simpan token di sessionStorage dan tampilkan info juri: *"Juri Kursi #1 — Gelanggang 1"*.
2. **Tata Letak Penilaian Ergonomis**:
   - Layar dibagi dua secara vertikal: **Kiri = Sudut Merah (`#DC2626`)**, **Kanan = Sudut Biru (`#2563EB`)**.
   - Setiap sudut memiliki 3 tombol sentuh berukuran ekstra besar (min height 120px):
     - **Pukulan (+1)**: Label jelas, ikon kepalan tangan / *fist*.
     - **Tendangan (+2)**: Ikon kaki / *foot*.
     - **Jatuhan/Bantingan (+3)**: Ikon bantingan.
3. **Respon Sentuh & Haptik**:
   - Begitu tombol disentuh (`touchstart` / `mousedown`), aktifkan getaran perangkat (*Haptic Feedback* via `navigator.vibrate(50)`).
   - Munculkan efek animasi riak (*ripple / flash border*) di tombol yang ditekan.
   - Segera kirimkan pesan `judge_input` ke WebSocket dengan `idempotency_key` baru.
4. **Prinsip Anti-Bias Juri**:
   - Sesuai aturan IPSI/PERSILAT: **Juri sama sekali TIDAK BOLEH melihat skor total atau tombol juri lain**.
   - Tablet juri hanya menampilkan:
     - Nomor ronde saat ini (*Round 1, 2, 3*) & status laga (*Running / Paused / Break*).
     - Status koneksi (Hijau = Online, Merah = Reconnecting).
     - Log feedback personal mini: *"Input Terkirim: Tendangan Merah"* (pudar dalam 1 detik).
5. **Mode Offline & Pemulihan Jaringan**:
   - Jika koneksi WebSocket drop, jangan kunci tombol! Simpan input juri dalam array lokal (*indexed/memory queue*).
   - Begitu koneksi terhubung kembali, segera *flush* antrean dan kirimkan `sync_request`.

---

### MODUL 2: Referee (Wasit) Control Pad (`/referee/console`)
Antarmuka untuk Wasit Pertandingan atau Dewan Hakim yang bertugas di meja pengawas gelanggang.

#### Kebutuhan UX:
1. **Panel Kendali Pertandingan**:
   - Tombol kontrol besar dengan warna standar:
     - **Mulai Ronde** (Hijau): Mengirimkan `action: "start"`.
     - **Jeda / Berhenti Sementara** (Kuning/Oranye): Mengirimkan `action: "pause"`.
     - **Lanjutkan** (Biru): Mengirimkan `action: "resume"`.
     - **Selesaikan Pertandingan** (Merah): Mengirimkan `action: "finish"`.
2. **Panel Penjatuhan Hukuman**:
   - Tab Sudut Merah & Sudut Biru:
     - Tombol **Teguran 1 (-1)**, **Teguran 2 (-1)**.
     - Tombol **Peringatan 1 (-5)**, **Peringatan 2 (-10)**.
     - Tombol **Diskualifikasi** (Mengirimkan `cancel` atau penalti khusus).
3. **Live Indicator Juri**:
   - Status 3 atau 5 juri (terhubung / terputus) dengan penanda kursi (*Seat 1, Seat 2, Seat 3*).
   - Tampilan visual ketika terjadi konsensus poin sah secara real-time.

---

### MODUL 3: Layar Scoreboard Publik (`/scoreboard/:arena_id`)
Tampilan resolusi tinggi layar lebar (1080p / 4K / TV LED Stadium) untuk penonton dan official tim.

#### Kebutuhan UX:
1. **Desain Visual Broadcast Grade**:
   - Tema gelap berlatar belakang kontras tinggi (*Dark Stadium Theme*).
   - Tipografi digital tegas dan masif (angka skor terlihat jelas dari jarak 25-50 meter).
2. **Komponen Layar**:
   - **Header**: Nama Kejuaraan, Nama Gelanggang (*"GELANGGANG 1"*), Kategori Kelas Tanding (*"Tanding Dewasa Putra - Kelas C (55kg - 60kg)"*).
   - **Tengah Atas**: **Digital Timer Masif (MM:SS)**. Berwarna hijau saat berjalan, merah/berkedip saat jeda (*PAUSED*), oranye saat istirahat (*BREAK*).
   - **Badge Ronde**: Penanda visual Ronde 1, Ronde 2, Ronde 3 aktif.
   - **Sisi Kiri (Sudut Merah)**:
     - Nama Atlet & Kontingen/Perguruan.
     - Skor Akumulasi Total Raksasa.
     - Breakdown per ronde (Ronde 1: 4 | Ronde 2: 6 | Ronde 3: 2).
     - Badge Hukuman (Teguran: 1, Peringatan: -).
   - **Sisi Kanan (Sudut Biru)**:
     - Struktur simetris dengan sudut merah.
3. **Animasi & Efek Poin Masuk**:
   - Saat menerima event `score_confirmed`, tampilkan notifikasi animasi melayang (*popup toast point*) di atas skor sudut: `+1 PUKULAN` atau `+2 TENDANGAN`.
4. **Modal Pengumuman Pemenang**:
   - Saat menerima `match_result`, layar secara dramatis memunculkan spanduk kemenangan:
     - Warna sudut pemenang memenuhi layar.
     - Menampilkan nama juara, jenis kemenangan (*Menang Angka / TKO / Diskualifikasi*), dan skor akhir.

---

### MODUL 4: Admin & Dewan Operations Dashboard (`/admin/*`)
Pusat komando dan manajemen seluruh aspek kejuaraan digital pencak silat.

#### Halaman & Fitur:
1. **Manajemen Kejuaraan (`/admin/competitions`)**:
   - Buat kejuaraan baru, atur rentang tanggal, lokasi, dan pilih ruleset default.
   - Kendali status kompetisi: Draft ➔ Aktif ➔ Selesai.
2. **Manajemen Gelanggang (`/admin/arenas`)**:
   - Pembuatan dan monitor semua gelanggang aktif.
   - Pemantauan pertandingan mana yang sedang aktif di setiap gelanggang.
3. **Verifikasi Peserta & Timbang Badan (`/admin/participants`)**:
   - Tabel pendaftaran atlet dengan filter status (`pending`, `verified`, `rejected`).
   - Tombol aksi verifikasi timbang badan dan berkas fisik (mengubah status menjadi `verified`).
4. **Generator & Visualisasi Bagan (`/admin/brackets`)**:
   - Pilih kategori tanding ➔ Klik tombol *"Generate Bagan Single-Elimination"*.
   - Visualisasi bagan pohon interaktif (*bracket tree*) dengan status atlet, BYE, dan pemenang yang otomatis terhubung ke pertandingan selanjutnya.
5. **Jadwal & Antrean Pertandingan (`/admin/matches`)**:
   - Penjadwalan partai pertandingan.
   - Tombol aksi *"Muat ke Gelanggang"* (`load-arena`) dengan penanganan error jika arena sibuk (*409 ARENA_BUSY*).
   - Monitor audit trail event log dan rincian breakdown skor per juri.
6. **Manajemen Akun Official & Pembuat PIN Juri (`/admin/officials`)**:
   - Tambah akun Operator/Dewan/Wasit.
   - Generator PIN juri: Pilih kejuaraan, gelanggang, peran, dan nomor kursi (1..5) ➔ Cetak lembar PIN atau salin PIN acak untuk dibagikan ke juri.
7. **Rulesets Engine Editor (`/admin/rulesets`)**:
   - Konfigurasi aturan penilaian IPSI 2024, PERSILAT v7, atau aturan kustom lokal.
8. **Laporan & Ekspor (`/admin/reports`)**:
   - Unduh klasemen medali kontingen, ringkasan metrik statistik pertandingan, dan ekspor data turnamen lengkap dalam format JSON.

---

### MODUL 5: Portal Manajer Kontingen (`/contingent/*`)
Antarmuka mandiri untuk pelatih / manajer tim kontingen.

#### Halaman & Fitur:
1. **Registrasi & Login Mandiri**: Form pendaftaran kontingen dengan verifikasi PIC, email, nomor telepon/WhatsApp, dan password.
2. **Dashboard Kontingen**:
   - Profil kontingen / nama perguruan.
   - Daftar atlet yang dimiliki.
3. **Formulir Pendaftaran Atlet**:
   - Pendaftaran atlet baru ke master data.
   - Mendaftarkan atlet ke kategori tanding yang dibuka pada kejuaraan aktif.
   - Pemantauan status verifikasi berkas: *Pending (Menunggu Timbang Badan)* / *Verified (Siap Tanding)* / *Rejected*.
4. **Live Bracket & Perolehan Medali**:
   - Memantau bagan pertandingan atletnya dan perolehan medali kontingen secara real-time.

---

### MODUL 6: Portal Publik & Penonton (`/public/*`)
1. **Klasemen Medali Turnamen (`/public/medals`)**: Tabel perolehan medali emas, perak, perunggu per kontingen yang auto-update.
2. **Bagan Interaktif Publik (`/public/brackets`)**: Bagan turnamen yang dapat diperbesar/geser (*zoom/pan*) untuk melihat jalur atlet menuju final.

---

## 8. Skema Tipe Data TypeScript (`src/types/`)

Seluruh tipe data berikut **wajib** digunakan pada seluruh aplikasi frontend dan mencerminkan domain Go backend secara presisi:

```typescript
// src/types/competition.ts
export type CompetitionStatus = 'draft' | 'active' | 'completed' | 'cancelled'

export interface Competition {
  id: string
  name: string
  location: string
  start_date: string
  end_date: string
  ruleset_id?: string
  ruleset_name: string
  status: CompetitionStatus
  created_at: string
}

export interface Category {
  id: string
  competition_id: string
  name: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  weight_class: string
  ruleset_id?: string
  created_at: string
}

export interface Arena {
  id: string
  competition_id: string
  name: string
  created_at: string
}

// src/types/athlete.ts
export type RegistrationStatus = 'pending' | 'verified' | 'rejected'

export interface Athlete {
  id: string
  contingent_id?: string
  name: string
  gender: 'male' | 'female' | 'putra' | 'putri'
  team: string
  created_at: string
}

export interface Participant {
  id: string
  competition_id: string
  athlete_id?: string
  contingent_id?: string
  name?: string
  club?: string
  weight_class: string
  gender: string
  registration_status: RegistrationStatus
  created_at: string
}

export interface Contingent {
  id: string
  name: string
  pic_name: string
  email: string
  phone: string
  created_at: string
}

// src/types/match.ts
export type MatchStatus = 'idle' | 'ready' | 'round_1' | 'break_1' | 'round_2' | 'break_2' | 'round_3' | 'paused' | 'finished' | 'cancelled'
export type AthleteCorner = 'red' | 'blue'

export interface MatchScoreRound {
  id?: string
  match_id?: string
  round: number
  athlete_red_score: number
  athlete_blue_score: number
  red_penalties: number
  blue_penalties: number
  updated_at?: string
}

export interface JudgeScoreSummary {
  judge_id: string
  role: string
  seat_number: number
  connected: boolean
  red_score: number
  blue_score: number
}

export interface MatchScoreDetail {
  match_id: string
  total_red: number
  total_blue: number
  rounds: MatchScoreRound[]
  judges?: JudgeScoreSummary[]
  consensus?: {
    red_score: number
    blue_score: number
  }
}

export interface MatchEvent {
  id: string
  match_id: string
  event_type: string
  payload: string
  judge_id: string
  created_at: string
}

export interface Match {
  id: string
  competition_id: string
  arena_id: string
  bracket_slot_id?: string
  athlete_red_id: string
  athlete_blue_id: string
  current_round: number
  status: MatchStatus
  winner_id?: string
  created_at: string
  started_at?: string
  finished_at?: string
}

// src/types/bracket.ts
export interface BracketSlot {
  id: string
  competition_id: string
  weight_class: string
  gender: string
  round: number
  slot_index: number
  participant_id?: string
  is_bye: boolean
  winner_id?: string
  next_slot_id?: string
}

// src/types/ruleset.ts
export interface RulesetScoringConfig {
  round_duration: number
  break_duration: number
  technique_points: number
  knockdown_points: number
  fall_points: number
  disqualification_threshold?: number
}

export interface Ruleset {
  id: string
  name: string
  version: string
  scoring: RulesetScoringConfig
  status: string
  created_at: string
}

// src/types/websocket.ts
export type WsScoreType = 'technique' | 'knockdown' | 'fall'
export type WsActionType = 'start' | 'pause' | 'resume' | 'finish' | 'cancel'

export interface WsJudgeInputPayload {
  event_id?: string
  idempotency_key?: string
  score_type: WsScoreType
  target: AthleteCorner
  round: number
}

export interface WsRefereeControlPayload {
  action: WsActionType
}

export interface WsPenaltyPayload {
  target: AthleteCorner
  level: number
}

export interface WsScoreUpdatePayload {
  round: number
  red_technical: number
  blue_technical: number
  red_penalties: number
  blue_penalties: number
}

export interface WsScoreConfirmedPayload {
  score_type: WsScoreType
  target: AthleteCorner
  points: number
  count: number
  round: number
}

export interface WsSyncResponsePayload {
  match_id: string
  arena_id: string
  state: MatchStatus
  round: number
  timer_remaining_ms: number
  timer_running: boolean
  scores: Array<{
    RedTechnical: number
    BlueTechnical: number
    RedPenalties: number
    BluePenalties: number
  }>
  red_total: number
  blue_total: number
  server_time: string
}
```

---

## 9. Desain Implementasi Composables Kunci (Pinia & Hooks)

### 9.1. Composable `useWebSocket.ts`
Mengelola siklus hidup koneksi WebSocket, auto-reconnect dengan exponential backoff, dan pengiriman event.

```typescript
// apps/web/src/composables/useWebSocket.ts
import { ref, onMounted, onUnmounted } from 'vue'

export function useWebSocket(arenaId: string, token?: string) {
  const ws = ref<WebSocket | null>(null)
  const isConnected = ref(false)
  const lastError = ref<string | null>(null)
  const messageListeners = new Map<string, Array<(payload: any) => void>>()

  let reconnectTimeout: any = null
  let reconnectAttempts = 0

  function connect() {
    if (!arenaId) return
    const baseUrl = import.meta.env.VITE_WS_BASE_URL || `ws://${window.location.host}`
    const query = token ? `?token=${encodeURIComponent(token)}` : ''
    const url = `${baseUrl}/ws/arena/${arenaId}${query}`

    ws.value = new WebSocket(url)

    ws.value.onopen = () => {
      isConnected.value = true
      lastError.value = null
      reconnectAttempts = 0
      // Request initial state synchronization
      sendMessage('sync_request', {})
    }

    ws.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        const handlers = messageListeners.get(data.type) || []
        handlers.forEach(fn => fn(data.payload))
      } catch (err) {
        console.error('Failed to parse WS message:', err)
      }
    }

    ws.value.onclose = () => {
      isConnected.value = false
      scheduleReconnect()
    }

    ws.value.onerror = (err) => {
      lastError.value = 'WebSocket connection error'
      ws.value?.close()
    }
  }

  function scheduleReconnect() {
    if (reconnectTimeout) clearTimeout(reconnectTimeout)
    // Exponential backoff: min 1s, max 10s
    const delay = Math.min(1000 * Math.pow(1.5, reconnectAttempts), 10000)
    reconnectAttempts++
    reconnectTimeout = setTimeout(connect, delay)
  }

  function sendMessage(type: string, payload: any) {
    if (ws.value && isConnected.value) {
      ws.value.send(JSON.stringify({ type, payload }))
    }
  }

  function onMessage<T>(type: string, callback: (payload: T) => void) {
    if (!messageListeners.has(type)) {
      messageListeners.set(type, [])
    }
    messageListeners.get(type)!.push(callback)
  }

  onMounted(connect)
  onUnmounted(() => {
    if (reconnectTimeout) clearTimeout(reconnectTimeout)
    ws.value?.close()
  })

  return {
    isConnected,
    lastError,
    sendMessage,
    onMessage,
  }
}
```

---

### 9.2. Composable `useTimer.ts`
Menghitung waktu mundur yang disinkronisasi dari server dengan interpolasi lokal yang mulus.

```typescript
// apps/web/src/composables/useTimer.ts
import { ref, computed, onUnmounted } from 'vue'

export function useTimer() {
  const remainingMs = ref(0)
  const isRunning = ref(false)
  let ticker: any = null

  function sync(ms: number, running: boolean) {
    remainingMs.value = ms
    isRunning.value = running

    if (ticker) clearInterval(ticker)

    if (running && ms > 0) {
      ticker = setInterval(() => {
        if (remainingMs.value > 100) {
          remainingMs.value -= 100
        } else {
          remainingMs.value = 0
          isRunning.value = false
          clearInterval(ticker)
        }
      }, 100)
    }
  }

  const formattedTime = computed(() => {
    const totalSecs = Math.max(0, Math.floor(remainingMs.value / 1000))
    const minutes = Math.floor(totalSecs / 60)
    const seconds = totalSecs % 60
    return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
  })

  onUnmounted(() => {
    if (ticker) clearInterval(ticker)
  })

  return {
    remainingMs,
    isRunning,
    formattedTime,
    sync,
  }
}
```

---

### 9.3. Client Fetch `src/lib/api.ts`
Pusat komunikasi REST API yang menangani otentikasi Bearer JWT dan penanganan error terpusat:

```typescript
// apps/web/src/lib/api.ts
const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

export interface ApiResponse<T> {
  success: boolean
  data: T
  meta?: {
    page: number
    per_page: number
    total: number
  }
}

export async function api<T>(path: string, init?: RequestInit): Promise<ApiResponse<T>> {
  const token = sessionStorage.getItem('token') || localStorage.getItem('token')

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string>),
  }

  if (token && !headers['Authorization']) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
  })

  const json = await res.json()

  if (!res.ok || json.success === false) {
    const errorMsg = json?.error?.message || `API Error (${res.status})`
    const errorCode = json?.error?.code || 'UNKNOWN_ERROR'
    const error = new Error(errorMsg) as any
    error.code = errorCode
    error.status = res.status
    error.details = json?.error?.details
    throw error
  }

  return json
}
```

---

## 10. Desain Tokens, Styling & Responsivitas (UI Guidelines)

1. **Palet Warna Identitas Pencak Silat**:
   - **Sudut Merah**: `#DC2626` (Red-600) | Hover: `#B91C1C` | Background Accent: `rgba(220, 38, 38, 0.15)`
   - **Sudut Biru**: `#2563EB` (Blue-600) | Hover: `#1D4ED8` | Background Accent: `rgba(37, 99, 235, 0.15)`
   - **Gelanggang Theme (Dark Mode Stadium)**: `#0F172A` (Slate-900), Surface: `#1E293B` (Slate-800), Border: `#334155`
   - **Aksen Timer Running**: `#10B981` (Emerald-500)
   - **Aksen Timer Paused / Break**: `#F59E0B` (Amber-500)
2. **Tipografi**:
   - Digital Clock & Big Scores: Font monospaced / athletic bold sans-serif (`Oswald`, `Chakra Petch`, atau `Inter` dengan font-weight 900).
3. **Viewport & Breakpoints**:
   - **Judge PWA**: Landscape Tablet Viewport (min 1024x768). Non-aktifkan pinch-zoom (`user-scalable=no`) dan scroll body.
   - **Scoreboard**: Fullscreen display rasio 16:9 tanpa scrollbar.
   - **Admin & Contingent**: Responsif Desktop & Mobile Tablet.

---

## 11. Panduan Implementasi Langkah-demi-Langkah (Roadmap Pengembangan)

Pengembang atau AI Agent yang mengimplementasikan frontend ini dapat mengeksekusi secara bertahap:

1. **Tahap 1: Setup Domain Types & Network Layer**
   - Lengkapi file di `src/types/` dengan interface dari Bab 8.
   - Konfigurasi `src/lib/api.ts` dan utilitas generator UUID di `src/lib/uuid.ts`.
2. **Tahap 2: Autentikasi & Routing**
   - Buat store `useAuth` di `src/stores/auth.ts`.
   - Konfigurasi rute di `src/router/index.ts` dengan Navigation Guards (`requiresAuth`, `roleCheck`).
   - Buat halaman login: Admin (`/admin/login`), Juri PIN (`/judge/login`), dan Kontingen (`/contingent/login`).
3. **Tahap 3: Implementasi Modul Juri Tablet PWA (`/judge/scoring`)**
   - Susun komponen tombol skor sudut merah dan biru.
   - Sambungkan `useWebSocket` dengan token scoped juri.
   - Terapkan getaran haptik dan pengiriman `judge_input` ber-idempotency key.
4. **Tahap 4: Implementasi Scoreboard Publik (`/scoreboard/:arena_id`)**
   - Susun antarmuka resolusi tinggi timer, nama atlet, skor per ronde, dan notifikasi poin masuk.
   - Sinkronisasi dengan event `state_change`, `timer_sync`, `score_confirmed`, dan `match_result`.
5. **Tahap 5: Implementasi Wasit Control Pad (`/referee/console`)**
   - Panel kontrol timer (start, pause, resume, finish) dan penjatuhan teguran/peringatan.
6. **Tahap 6: Implementasi Admin & Dewan Dashboard**
   - CRUD Kejuaraan, Gelanggang, Kategori, Atlet, dan Official.
   - Halaman Verifikasi Peserta & Timbang Badan.
   - Generator & Visualizer Bagan Single-Elimination.
   - Penjadwalan Laga & Aksi `load-arena` (tangani konflik 409 ARENA_BUSY).
7. **Tahap 7: Implementasi Portal Manajer Kontingen & Halaman Publik**
   - Pendaftaran kontingen & atlet mandiri.
   - Klasemen medali dan bagan turnamen publik.

---
*Dokumen ini merupakan spesifikasi final desain frontend yang mengikat dan siap dieksekusi secara langsung.*
