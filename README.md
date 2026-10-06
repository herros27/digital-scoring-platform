# Pencak Silat Digital Scoring System

Backend system untuk **Pencak Silat Kategori Tanding** dengan real-time scoring, consensus juri, multi-arena support, dan tournament bracket management.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend API + WS | Go 1.27 (stdlib `net/http` + `gorilla/websocket`) |
| Rules Engine | Go struct-based, dynamic rule set dari JSON/DB |
| Database | PostgreSQL 16 (`jackc/pgx/v5` driver via stdlib) |
| Auth & Security | PIN (lookup HMAC-SHA256 + bcrypt) + JWT (HS256) + Token Bucket Rate Limiting (`golang.org/x/time/rate`) + kunci IP setelah PIN salah |
| Validation | `go-playground/validator/v10` |
| Caching & Metrics | In-Memory TTL Cache (`internal/cache`) + Prometheus (`internal/metrics`) |
| Deployment | Single binary + systemd + Caddy reverse proxy |

---

## Project Structure

```
cmd/server/main.go           # Entry point & graceful shutdown
internal/
  api/                       # HTTP handlers, router, middleware (JWT, RBAC, CORS, RateLimit)
    handler/                 # REST handlers (auth, competition, match, arena, bracket, official, contingent, report, live)
    swagger.go               # Swagger UI + OpenAPI spec
  auth/                      # PIN (crypto/rand, lookup HMAC, validasi per role), pembatas percobaan gagal, bcrypt, JWT claims (role & seat)
  bracket/                   # Single-elimination bracket generator & tree traversal
  cache/                     # In-memory TTL report cache with concurrent invalidation
  config/                    # Centralized environment & app settings (godotenv + env/v11)
  consensus/                 # Dynamic judge consensus engine (sliding window voting)
  domain/                    # Core domain models & sentinel errors (zero dependency)
  hub/                       # WebSocket hub per arena (role-based broadcast & reconnect sync)
  logger/                    # Structured logging (slog)
  metrics/                   # Prometheus observability (/metrics)
  orchestrator/              # Match orchestrator, server-side timer, event deduplication
  rules/                     # Rules engine (data-driven dari JSON & dynamic DB resolver)
  service/                   # Business logic layer (match, comp, arena, bracket, official, contingent, report)
  statemachine/              # Match state machine (idle → ready → rounds → finished → cancelled)
  store/                     # Repository layer (PostgreSQL connection pool & pgx/v5 queries)
rulesets/
  ipsi_2024.json             # IPSI 2024 ruleset
  persilat_v7.json           # PERSILAT ruleset
migrations/                  # SQL migrations (001 s/d 018) + Auto-migration engine
tests/                       # End-to-end integration test scenarios (tournament lifecycle & workflows)
docker-compose.yml           # Local PostgreSQL
docs/
  openapi.yaml               # OpenAPI 3.0 spec & WebSocket protocol contracts
  task_update.md             # Log progres & status fitur backend
  frontend_progress.md       # Log progres, status modul & gap analysis frontend
  CREDENTIALS.md             # Kredensial akun uji coba & format otorisasi
```

---

## Core Features

### 1. Match State Machine
```
IDLE → READY → ROUND_1 → BREAK_1 → ROUND_2 → BREAK_2 → ROUND_3 → FINISHED
                    ↑_____________|_____________|_____________|
                    Pause/Resume  Cancel       Early Win (KO/TKO/Point Diff)
```
- Thread-safe dengan `sync.Mutex`
- Timer server-side (single source of truth)
- Event sourcing: setiap transition = audit log

### 2. Rules Engine (Data-Driven)
Rule set dimuat dari JSON (`rulesets/*.json` atau DB):
```json
{
  "name": "IPSI 2024",
  "version": "2024.1",
  "scoring": {
    "technique_points": 1,
    "knockdown_points": 2,
    "fall_points": 3,
    "max_penalties": 3,
    "penalty_threshold": 1,
    "round_duration": 120,
    "break_duration": 60,
    "total_rounds": 3
  }
}
```
- `LoadRuleset(name)` → file JSON → fallback hardcoded defaults
- Bisa extend: tambah file JSON baru, tidak perlu ganti code

### 3. Consensus Engine
- **Sliding time window** (default 3 detik)
- **Majority voting**: ≥2 dari 3 juri input sama dalam window = valid
- Anti-duplicate & anti-spam per judge
- Penalty authority: referee only (configurable via ruleset)

### 4. WebSocket Hub (Per-Arena Isolation)
```
Arena A Hub          Arena B Hub
  ├─ Judge 1           ├─ Judge 1
  ├─ Judge 2           ├─ Judge 2
  ├─ Judge 3           ├─ Judge 3
  ├─ Referee           ├─ Referee
  └─ Scoreboard(s)     └─ Scoreboard(s)
```
- Role-based broadcast: judge hanya terima ack sendiri, referee/scoreboard terima semua
- Auth via query param: `wss://host/ws/arena/{id}?token=<JWT>`
  - **Tanpa token** = penonton publik (scoreboard TV), role `scoreboard`, read-only; hanya bisa terhubung ke arena yang sudah punya pertandingan aktif (selain itu HTTP 404)
  - **Token PIN** (juri/wasit) harus cocok dengan arena yang dituju (selain itu HTTP 403); role `wasit` dan `dewan` dipetakan ke `referee`
  - **Token login email** (tanpa arena) hanya diterima untuk `admin`, `dewan`, `wasit`
- Reconnection + full state snapshot recovery (`sync_request` → `sync_response`)

### 5. Match Orchestrator
Koordinator pusat per match:
```
MatchOrchestrator
  ├─ StateMachine
  ├─ RulesEngine
  ├─ ConsensusEngine
  ├─ Timer (server-side)
  ├─ WebSocket Hub
  └─ Event Store (PostgreSQL)
```
- Handle control (start/pause/resume/finish), score input, penalty, override
- Auto-transition on timer expiry
- Crash recovery: reload state dari DB, replay event log

### 6. Tournament Bracket (Single Elimination)
- Power-of-2 slot calculation dengan BYE handling
- Standard seeding (1 vs 8, 4 vs 5, 2 vs 7, 3 vs 6)
- Auto-advance winner ke round berikutnya
- Walkover/withdrawal support

### 7. Auth & Session (3 Flow Terpadu)
| Flow | Aktor | Method | Scope & Token Lifetime |
|------|-------|--------|------------------------|
| **Officials** | Admin, Dewan, Operator, Wasit | Email & Password (bcrypt) → JWT | 24 jam (`role: admin / dewan / operator / wasit`) |
| **Juri & Wasit Gelanggang** | Tablet Juri / Meja Wasit | PIN saja (juri 6 digit, wasit 8 digit) → JWT | Turnamen, gelanggang, role, dan kursi ditentukan server dari PIN; token scoped ke `competition_id`, `arena_id`, `role: judge/wasit`, `seat_number` |
| **Kontingen** | Manajer Tim / Pelatih | Email & Password (bcrypt) → JWT | 24 jam (`role: contingent`) |

- **Clean Architecture Juri**: Pemisahan tegas antara otorisasi (`role: "judge"`) dan nomor kursi fisik gelanggang (`seat_number`). Mendukung format juri dinamis (1, 3, 5, dst).
- **Login PIN-saja**: setiap PIN unik di seluruh sistem (dicari lewat `judge_pins.pin_lookup`, HMAC-SHA256 terindeks). PIN salah 5x beruntun mengunci IP 15 menit (HTTP 429).
- **Server-Side PIN & Multi-Device Sync**: PIN di-generate otomatis secara kriptografis (`crypto/rand`) dengan expiration time (`expires_at`), disimpan persisten dalam database (`judge_pins.raw_pin`), dan disinkronisasi ke seluruh perangkat admin/operator via `GET /api/v1/auth/pins`.

### 8. Pendaftaran Atlet & Verifikasi Panitia (Contingent Workflow)
- **Portal Kontingen**: Perwakilan kontingen dapat mendaftar mandiri (`/api/v1/auth/contingent/register`) dan mendaftarkan atlet asuhannya.
- **Status Registrasi**: Atlet yang didaftarkan kontingen otomatis berstatus `pending`.
- **Meja Timbang / Verifikasi**: Panitia (Admin / Dewan) memverifikasi berkas & timbang badan peserta (`PUT .../participants/{id}/verify`) menjadi `verified` atau `rejected`.
- **Bracket Safeguard**: Hanya atlet berstatus `verified` yang dapat dimasukkan ke dalam bagan pertandingan single-elimination.

### 9. Database (PostgreSQL 16) - 17 Versioned Migrations
Tabel database terstruktur & termigrasi lengkap:
- `competitions`, `arenas`, `categories`, `rulesets`
- `athletes`, `contingents`, `participants` (dengan `registration_status`), `officials`
- `judge_pins` (dengan `seat_number INT` dan `raw_pin VARCHAR`), `matches`, `match_scores`, `match_events` (audit trail), `bracket_slots`

---

## Quick Start

### Prerequisites
- Go 1.27+
- PostgreSQL 16
- (Optional) Docker for local DB

### 1. Start Database
```bash
docker compose up -d
```
Default: `postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable`

### 2. Configure Environment
```bash
export DB_URL="postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable"
export JWT_SECRET="your-production-secret-here"
export ENV="development"  # atau "production"
```

### 3. Run Migrations (Otomatis)
Server secara otomatis menjalankan migrasi saat bootstrap menggunakan file migration yang di-embed langsung ke binary Go (`//go:embed` + Goose v3). Cukup jalankan server, dan seluruh schema (001 s/d 018) akan otomatis diterapkan:
```bash
go run ./cmd/server
```
*(Opsional: set env `AUTO_MIGRATE=false` jika ingin menonaktifkan auto-migrasi).*

### 4. Build & Run
```bash
go build -o bin/server ./cmd/server
./bin/server
```
Server listen di `:8080`

### 5. API Docs (Swagger UI)
```
GET http://localhost:8080/docs
GET http://localhost:8080/docs/openapi.yaml
```

---

## Key API Endpoints

### 1. Authentication
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/login` | Public | Login akun Official (Admin/Dewan/Operator/Wasit) via email & password |
| POST | `/api/v1/auth/judge-login` | Public | Login juri/wasit hanya dengan `{ "pin" }` (6 digit juri, 8 digit wasit) → JWT (claims: `role`, `arena_id`, `competition_id`, `seat_number`) |
| POST | `/api/v1/auth/contingent/register` | Public | Registrasi akun manajer kontingen baru |
| POST | `/api/v1/auth/contingent/login` | Public | Login akun kontingen via email & password → JWT (`role: contingent`) |
| POST | `/api/v1/auth/create-pin` | Admin, Dewan | Generate PIN acak/custom untuk juri gelanggang tertentu |

### 2. Competition
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/competitions` | JWT | List kompetisi (mendukung pagination `?page=1&limit=20&status=`) |
| POST | `/api/v1/competitions` | Admin, Dewan | Buat kompetisi baru (termasuk binding `ruleset_id`) |
| GET | `/api/v1/competitions/{id}` | JWT | Detail data kompetisi |
| PUT | `/api/v1/competitions/{id}` | Admin, Dewan | Perbarui data kompetisi |
| PUT/POST | `/api/v1/competitions/{id}/activate` | Admin, Dewan | Aktifkan status kompetisi |
| POST | `/api/v1/competitions/{id}/pause` | Admin, Dewan | Jeda kompetisi |
| POST | `/api/v1/competitions/{id}/finish` | Admin, Dewan | Selesaikan kompetisi |
| POST | `/api/v1/competitions/{id}/cancel` | Admin, Dewan | Batalkan kompetisi |

### 3. Categories & Weight Classes
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/competitions/{comp_id}/categories` | JWT | List kategori kelas tanding kompetisi |
| POST | `/api/v1/competitions/{comp_id}/categories` | Admin, Dewan | Tambah kategori baru (nama, gender, min/max weight) |
| GET | `/api/v1/competitions/{comp_id}/categories/{id}` | JWT | Detail kategori |

### 4. Athletes & Participants
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/athletes` | JWT | List master data atlet (`?page=&limit=&gender=`) |
| POST | `/api/v1/athletes` | Admin, Dewan, Contingent | Tambah atlet baru ke master database |
| GET | `/api/v1/athletes/{id}` | JWT | Detail profil atlet |
| GET | `/api/v1/competitions/{comp_id}/athletes` | JWT | List atlet yang terdaftar di kompetisi ini |
| POST | `/api/v1/competitions/{comp_id}/athletes` | Admin, Dewan, Contingent | Daftarkan atlet ke kompetisi (auto status: `pending` jika kontingen) |
| GET | `/api/v1/competitions/{comp_id}/participants` | JWT | List peserta kompetisi (`?status=pending/verified/rejected`) |
| PUT | `/api/v1/competitions/{comp_id}/participants/{id}/verify` | Admin, Dewan | Verifikasi/tolak peserta (meja timbang / verifikasi berkas) |

### 5. Arena
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/competitions/{comp_id}/arenas` | JWT | List gelanggang kompetisi |
| POST | `/api/v1/competitions/{comp_id}/arenas` | Admin, Dewan | Tambah gelanggang baru |
| GET | `/api/v1/competitions/{comp_id}/arenas/{id}` | JWT | Detail gelanggang |
| PUT | `/api/v1/competitions/{comp_id}/arenas/{id}` | Admin, Dewan | Perbarui nama/konfigurasi gelanggang |
| DELETE | `/api/v1/competitions/{comp_id}/arenas/{id}` | Admin, Dewan | Hapus gelanggang |

### 6. Matches & Scoring
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/competitions/{comp_id}/matches` | JWT | List jadwal pertandingan (`?page=&limit=&status=`) |
| POST | `/api/v1/competitions/{comp_id}/matches` | Admin, Dewan | Buat jadwal pertandingan |
| GET | `/api/v1/competitions/{comp_id}/matches/{id}` | JWT | Detail pertandingan |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/load-arena` | Admin, Dewan, Wasit | Muat match ke gelanggang (init orchestrator, cegah tabrakan arena) |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/start` | Admin, Dewan, Wasit | Mulai ronde pertandingan / jalankan timer |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/pause` | Admin, Dewan, Wasit | Jeda pertandingan |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/resume` | Admin, Dewan, Wasit | Lanjutkan ronde yang terjeda |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/finish` | Admin, Dewan, Wasit | Selesaikan match & otomatis majukan pemenang di bagan |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/cancel` | Admin, Dewan, Wasit | Batalkan pertandingan |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/reschedule` | Admin, Dewan, Wasit | Jadwalkan ulang partai yang dibatalkan (kembali `idle`, skor dihapus, diulang dari 0-0) |
| GET | `/api/v1/competitions/{comp_id}/matches/{id}/scores` | JWT | Rincian skor akumulatif & breakdown per-seat juri |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/scores` | JWT (Judge) | Submit skor HTTP fallback (didukung `Idempotency-Key` header) |
| GET | `/api/v1/competitions/{comp_id}/matches/{id}/events` | JWT | Riwayat lengkap event log / audit trail pertandingan |

### 7. Bracket & Tournament
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/competitions/{comp_id}/brackets/generate` | Admin, Dewan | Generate bagan turnamen sistem gugur (hanya peserta `verified`) |
| GET | `/api/v1/competitions/{comp_id}/brackets` | JWT | Ambil struktur pohon bagan (`?weight_class=&gender=`) |

### 8. Contingents & Officials Management
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/contingents` | Admin, Dewan | List semua kontingen yang terdaftar |
| GET | `/api/v1/contingents/{id}` | JWT | Detail profil kontingen |
| GET | `/api/v1/officials` | Admin, Dewan | List akun panitia/wasit/operator |
| POST | `/api/v1/officials` | Admin, Dewan | Buat akun official baru |
| GET | `/api/v1/officials/{id}` | Admin, Dewan | Detail akun official |
| PUT | `/api/v1/officials/{id}` | Admin, Dewan | Perbarui akun official |
| DELETE | `/api/v1/officials/{id}` | Admin, Dewan | Hapus akun official |

### 9. Rulesets
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/rulesets` | JWT | List konfigurasi ruleset pertandingan (IPSI 2024, PERSILAT, Custom DB) |
| POST | `/api/v1/rulesets` | Admin, Dewan | Buat ruleset kustom baru via API (tersimpan langsung di database) |
| GET | `/api/v1/rulesets/{id}` | JWT | Detail konfigurasi ruleset JSON |
| PUT | `/api/v1/rulesets/{id}` | Admin, Dewan | Perbarui konfigurasi ruleset kustom |
| DELETE | `/api/v1/rulesets/{id}` | Admin, Dewan | Hapus ruleset kustom |

### 10. Reports & Analytics
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/competitions/{comp_id}/report/summary` | JWT | Ringkasan metrik statistik kompetisi (peserta, arena, match) |
| GET | `/api/v1/competitions/{comp_id}/report/medals` | JWT | Klasemen perolehan medali (emas, perak, perunggu per kontingen) |
| GET | `/api/v1/competitions/{comp_id}/report/export` | JWT | Ekspor arsip data turnamen lengkap format JSON |

### 11. Live Scoreboard & WebSocket
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/live/arenas` | Public | Daftar gelanggang yang sedang punya pertandingan aktif (nama gelanggang, atlet, ronde, skor) |
| GET | `/api/v1/live/scoreboard/{arena_id}` | Public | Snapshot pertandingan aktif di arena (status, ronde, timer, total skor, nama atlet) atau `{"status":"no active match"}` |
| GET | `/ws/arena/{arena_id}[?token=<JWT>]` | Token opsional | Real-time bi-directional channel gelanggang (Websocket); tanpa token = penonton read-only |

---

## Message Types (WebSocket)

### Client → Server
| Type | From | Payload |
|------|------|---------|
| `judge_input` | Judge | `{event_id, score_type, target, round}` (`event_id` = kunci idempotensi) |
| `state_change` | Referee, Admin | `{action}` dengan `action` = `start` \| `pause` \| `resume` \| `finish` (+`winner` bila seri) \| `cancel` \| `disqualify` (+`target`) \| `declare_outcome` (+`outcome`, `winner`, `note`) \| `adjust_time` (+`delta_ms`) \| `reset_round` |
| `penalty` | Referee, Admin | `{target, type}`; poin potongan ditentukan server dari katalog (`binaan_1/2`, `teguran_1/2`, `peringatan_1/2`) |
| `penalty_revoke` | Referee, Admin | `{target}` membatalkan sanksi terakhir sudut itu |
| `arena_signal` | Referee, Admin | `{signal: "gong"}` atau `{signal: "medic"\|"var", active}` |
| `sync_request` | Semua | `{}` |

### Server → Client
| Type | To | Payload |
|------|----|---------|
| `sync_response` | Pengirim `sync_request` | `{match_id, arena_id, state, round, timer_remaining_ms, timer_running, scores[], red_total, blue_total, server_time, ruleset_name, round_duration_ms, break_duration_ms, total_rounds, penalties[], penalty_catalog[], medic_active, var_active, tie_pending, presence, result}` |
| `state_change` | All | `{state, round, timer_remaining_ms, timer_running, timer_ends_at_ms, server_time_ms, tie_pending}` (saat dijeda `state` tetap `round_N`, `timer_running=false`; `tie_pending=true` bila waktu ronde terakhir habis dengan skor seri) |
| `timer_sync` | All | `{timer_remaining_ms, timer_running, timer_ends_at_ms, server_time_ms}` setiap 5 detik selama timer berjalan |
| `penalties` | All | `{penalties: [{id, round, target, type, label, points, clock_ms}]}` |
| `arena_signal` | All | `{signal, active}` |
| `presence` | Referee, Admin | `{judges: [{seat, connected, inputs, last_input}], referees, admins, scoreboards}` |
| `match_event` | Referee, Admin | log kejadian real-time (sama dengan `GET .../matches/{id}/events`, payload selalu memuat `clock_ms` dan `round`) |
| `score_update` | All | `{round, red_technical, blue_technical, red_penalties, blue_penalties}` |
| `score_confirmed` | All | `{score_type, target, points, count, round}` |
| `consensus_progress` | Hanya tablet juri | `{target, score_type, round, count, needed, window_ms}`: berapa juri (anonim, tanpa identitas kursi) yang sudah menekan teknik dan sudut yang sama dalam window; dikirim saat belum mencapai kesepakatan. Dipakai lampu konsensus di tablet juri |
| `match_result` | All | `{winner, corner, result_type, scores}` |
| `error` | Pengirim | `{code, message}` (mis. `FORBIDDEN`, `SCORING_NOT_ALLOWED`, `INVALID_STATE_TRANSITION`) |

Timer berjalan di server berbasis **tenggat absolut** (`timer_ends_at_ms`), bukan penghitung per detik. Klien menghitung `sisa = tenggat − jam server`
pada tiap penyegaran tampilan (jam server diselaraskan dari `server_time_ms`), dan server menyiarkan `timer_sync` tiap 5 detik. Dengan begitu
wasit, juri, dan scoreboard menampilkan waktu yang sama walau tab diperlambat browser.

**Skor seri:** pertandingan tidak pernah dimenangkan otomatis oleh sudut merah. Server menjalankan aturan seri ruleset (`tiebreak`):
1. **Potongan sanksi paling sedikit** menang (`result_type` = `tiebreak_penalties`).
2. **Teknik terbanyak** menurut urutan prioritas (bawaan dari nilai tertinggi: Tangkapan+Jatuhan, Jatuhan, Tendangan, Pukulan; `tiebreak_technique`). Sudut dengan jumlah lebih banyak pada jenis teknik paling bernilai yang jumlahnya berbeda menang.
3. Bila masih seri, pertandingan menunggu keputusan manual wasit (`tie_pending` pada waktu habis; `TIE_NEEDS_DECISION` lewat REST). Wasit menyertakan `winner` dan dasar keputusan `method`: `extra_round` (babak tambahan), `lighter_weight` (timbang ulang), `coin_toss` (undian koin), atau kosong = keputusan wasit (`tiebreak_extra_round`, `tiebreak_lighter_weight`, `tiebreak_coin_toss`, `referee_decision`).

Langkah 1-2 dihitung server dari data pertandingan; babak tambahan, timbang ulang, dan undian koin dilakukan di luar sistem lalu hasilnya dicatat wasit. Kombinasi teknik 1+2 dan 1+1 belum ada sebagai jenis teknik terpisah (hanya Pukulan, Tendangan, Jatuhan, Tangkapan+Jatuhan). Diskualifikasi (`disqualify`) memenangkan sudut lawan.

---

## Ruleset Format (JSON)

```json
{
  "name": "IPSI 2024",
  "version": "2024.1",
  "scoring": {
    "technique_points": 1,
    "knockdown_points": 2,
    "fall_points": 3,
    "max_penalties": 3,
    "penalty_threshold": 1,
    "round_duration": 120,
    "break_duration": 60,
    "total_rounds": 3
  }
}
```
Simpan di `rulesets/{name}.json` atau buat langsung via REST API `POST /api/v1/rulesets` (tersimpan di database PostgreSQL).

### Mengelola ruleset dari antarmuka
Admin dan dewan membuka **Regulasi & Ruleset** di panel operasional (`/ops?tab=regulasi`): daftar ruleset, **Ubah**, **Duplikat** (mis. salin preset IPSI lalu ubah angkanya), **Hapus**, dan **Ruleset Baru**. Editor memuat waktu dan babak, teknik dan nilainya, sanksi (termasuk centang Diskualifikasi), konsensus juri, aturan seri (urutan prioritas teknik), dan hasil khusus, dengan validasi yang sama seperti server. Ruleset dipilih saat membuat atau mengubah kejuaraan (disimpan sebagai `ruleset_id`); perubahan berlaku untuk partai yang dimuat ke gelanggang setelah disimpan, partai yang sedang berjalan tidak berubah. Ruleset yang masih dipakai kejuaraan tidak dapat dihapus (409 `CONFLICT`), begitu juga nama+versi ganda.

### Sanksi, teknik, dan konsensus per ruleset
Selain poin dan durasi, `scoring` dapat memuat tiga bagian opsional. Bila dikosongkan, server memakai nilai turunan/bawaan sehingga ruleset lama tetap berjalan:

```json
"techniques": [
  {"type": "technique",  "label": "Pukulan",             "points": 1},
  {"type": "catch_fall", "label": "Tangkapan + Jatuhan", "points": 4}
],
"penalties": [
  {"type": "teguran_2",    "label": "Teguran 2",    "points": 2},
  {"type": "peringatan_3", "label": "Peringatan 3", "points": 0, "action": "disqualify"}
],
"consensus": {"judges": 3, "majority": 2, "window_ms": 3000}
```
- **`techniques`**: tombol di tablet juri dan nilainya. `type` adalah `score_type` yang dikirim juri; jenis di luar daftar (selain `undo`) ditolak.
- **`penalties`**: katalog sanksi konsol wasit. `points` = potongan nilai; `action: "disqualify"` membuat sanksi itu langsung mendiskualifikasi sudut yang dikenai. Bawaan: Binaan 1-2 (0), Teguran 1 (-1), Teguran 2 (-2), Peringatan 1 (-5), Peringatan 2 (-10), Peringatan 3 (diskualifikasi). **Cocokkan dengan peraturan event.**
- **`outcomes`**: hasil khusus yang boleh dinyatakan wasit. Tiap item `{type, label, phase}` dengan `type` ∈ `ko`, `tko`, `walkover`, `withdrawal` dan `phase` ∈ `before_start`, `in_match`, `any`. Bawaan: KO dan TKO setelah pertandingan dimulai, WO sebelum dimulai, Undur Diri kapan saja. Dinyatakan lewat WebSocket `declare_outcome` atau `POST .../matches/{id}/outcome` dengan `winner` (sudut yang menang); `result_type` hasil = jenis hasil.
- **`tiebreak`**: `steps` (`fewest_penalties`, `technique_priority`), `technique_priority` (kode teknik dari yang paling bernilai), `manual_methods` (`extra_round`, `lighter_weight`, `coin_toss`). Kosong = bawaan di bagian "Skor seri".
- **`consensus`**: `judges` (1-5, jumlah kursi yang ditampilkan), `majority` (juri yang harus sepakat), `window_ms` (500-10000).
- Preset `IPSI 2024` dan `PERSILAT International` (seeder dan migrasi 019) memuat ketiganya; teknik "Tangkapan + Jatuhan" (1+3) bernilai 4 sebagai tombol terpisah dari Jatuhan (3).
- `POST/PUT /api/v1/rulesets` memvalidasi bagian ini (kode unik huruf kecil/angka/underscore, nilai teknik > 0, potongan >= 0, `majority` <= `judges`).

---

## Pemulihan Setelah Server Restart

Keadaan partai yang sedang berlangsung disimpan di tabel `match_runtime` (satu baris per partai aktif) setiap ada perubahan (dimuat, mulai, jeda, lanjut, skor sah, sanksi, sinyal dokter/VAR, koreksi waktu, pergantian babak) dan sisa waktu dicatat tiap 5 detik selama timer berjalan. Isinya: state mesin, sisa waktu, skor per babak, sanksi aktif, hitungan teknik (untuk aturan seri), keputusan wasit yang sudah ditetapkan, dan **ruleset yang dipakai saat partai dimuat** (mengubah ruleset di editor tidak mengubah partai yang berjalan, juga setelah restart). Baris dihapus saat partai selesai, dibatalkan, atau dijadwalkan ulang.

Saat server start, semua partai tersimpan dipulihkan ke gelanggang yang sama (gelanggang tetap terkunci oleh partai itu):

| Keadaan saat server mati | Setelah restart |
|---|---|
| Ronde berjalan atau dijeda | **Dijeda** dengan sisa waktu catatan terakhir; snapshot `recovered: true`, log mencatat `match_recovered`. Wasit memeriksa waktu (±10 detik) lalu menekan Lanjutkan |
| Istirahat antar babak | Hitung mundur istirahat berlanjut otomatis |
| Dimuat ke gelanggang, belum dimulai | Kembali apa adanya |
| Selesai atau dibatalkan | Tidak dipulihkan |

Tablet juri, konsol wasit, dan scoreboard tersambung ulang sendiri dan meminta sinkronisasi; konsol wasit menampilkan banner "Server dimulai ulang". Yang tidak ikut dipulihkan: input juri yang belum mencapai konsensus (jendela beberapa detik; juri cukup menekan lagi). Catatan sisa waktu berselang 5 detik, sehingga bila server mati saat ronde berjalan, waktu yang tersisa bisa paling banyak 5 detik lebih sedikit dari kenyataan, dan wasit dapat mengoreksinya. Basis data harus tetap tersedia (volume PostgreSQL persisten di Docker).

---

## Deployment (Production)

### VPS Topology
```
Internet → Cloudflare (DNS + DDoS) → Caddy (auto HTTPS, reverse proxy)
  ├─ Go App :8080
  ├─ PostgreSQL :5432 (local or managed)
  └─ Static PWA files (judge, scoreboard, admin)
```

### Caddy Config (example)
```caddyfile
scoring.example.com {
    encode gzip
    
    # API + WebSocket
    handle /api/* /ws/* {
        reverse_proxy localhost:8080
    }
    
    # PWA Frontends
    handle /judge/* {
        root * /var/www/judge
        try_files {path} /index.html
        file_server
    }
    handle /scoreboard/* {
        root * /var/www/scoreboard
        try_files {path} /index.html
        file_server
    }
    handle /admin/* {
        root * /var/www/admin
        try_files {path} /index.html
        file_server
    }
}
```

### Systemd Service
```ini
[Unit]
Description=Digital Scoring Server
After=network.target postgresql.service

[Service]
Type=simple
User=scoring
WorkingDirectory=/opt/digital-scoring
ExecStart=/opt/digital-scoring/bin/server
Environment=DB_URL=postgres://scoring_user@localhost:5432/digital_scoring?sslmode=disable
Environment=JWT_SECRET=your-secret
Environment=ENV=production
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

### Health Check
```bash
curl http://localhost:8080/api/v1/health
```
Returns: DB status, active arenas, active matches, WS connections.

---

## Development

### Database Seeding & Default Credentials
Untuk mengisi database kosong dengan data turnamen realistis (kompetisi, 2 gelanggang, kategori tanding, 4 kontingen, 8 atlet, bagan turnamen, match aktif, serta PIN wasit & juri):
```bash
go run ./cmd/seed/main.go
```
Daftar lengkap akun login official, kontingen, dan PIN wasit/juri didokumentasikan di:  
👉 **[docs/CREDENTIALS.md](docs/CREDENTIALS.md)**

### Run Tests
```bash
go test -v -count=1 ./...
```

Menjalankan seluruh **automated unit tests** (co-located di `internal/`) dan **integration test scenarios** (di `tests/`) dengan hasil 100% PASS:
- **Unit Tests (`internal/`)**:
  - `internal/auth`: Token issuance, verification, expiration, seat number, bcrypt PIN, validasi/generator/lookup PIN, pembatas percobaan gagal (kunci IP)
  - `internal/hub`: Pemetaan role JWT → role hub (wasit/dewan → referee)
  - `internal/bracket`: Single elimination tree generator, BYE handling, auto-advance
  - `internal/consensus`: Majority voting (2/3), sliding window (3s), duplicate suppression
  - `internal/statemachine`: Round lifecycle flow, invalid transitions, early finish
  - `internal/rules`: Dynamic loader & default ruleset configuration (IPSI / PERSILAT)
  - `internal/service`: Category & validation logic
  - `internal/cache`: TTL cache hit/miss, concurrent invalidation
  - `internal/metrics`: Prometheus metric vectors & latency histogram
  - `internal/config`: Environment parser & defaults
  - `internal/api`: Token bucket rate limiting middleware, struct validator, WebSocket melalui `MetricsMiddleware`, akses WebSocket (penonton publik, token PIN, token email)
- **Integration Tests (`tests/`)**:
  - `tests/pin_login_test.go`: Login hanya dengan PIN (database nyata): keunikan global, kedaluwarsa, rotasi, panjang per role, kunci IP, `create-pin` 409/400
  - `tests/contingent_test.go`: End-to-end contingent portal, athlete registration & verification
  - `tests/match_service_test.go`: Full match lifecycle, timer, WebSocket broadcast & scoring
  - `tests/official_and_report_test.go`: Tournament report aggregation, medals table, export
  - `tests/ruleset_crud_test.go`: Dynamic database ruleset CRUD integration
  - `tests/admin_features_test.go`: Arena CRUD & Competition administrative operations
  - `tests/p1_features_test.go` & `tests/p2_features_test.go`: Phase 1 & 2 feature regression tests

Tes login PIN (`tests/pin_login_test.go`) memakai database PostgreSQL nyata dan dilewati bila tidak tersedia
(default `postgres://scoring_user@localhost:5432/digital_scoring`, atau set `TEST_DB_URL`). CI menyediakannya lewat service Postgres.

Simulasi end-to-end (REST + WebSocket + konsensus juri, 20+ pemeriksaan; butuh backend berjalan dan data seed):
```bash
go run ./cmd/seed
go run ./cmd/server &
node apps/web/scripts/e2e-simulation.mjs   # keluar dengan kode 1 bila ada pemeriksaan gagal
```

### Generate Migration
```bash
migrate create -ext sql -dir migrations -seq description
```

### Add New Ruleset
- **Via API (Rekomendasi)**: Kirim request `POST /api/v1/rulesets` dengan payload scoring JSON (otomatis aktif di DB).
- **Via File (Offline)**: Buat file `rulesets/{name}.json` dan sistem akan memuatnya sebagai fallback.

---

## Architecture References

- **Full Backend Architecture**: [`architecture-backend.md`](architecture-backend.md) — comprehensive design doc
- **Development Log**: [`workLog/task.md`](workLog/task.md) — feature history
- **OpenAPI Spec**: [`docs/openapi.yaml`](docs/openapi.yaml)

---

## License

Internal project — Digital Scoring for Pencak Silat.