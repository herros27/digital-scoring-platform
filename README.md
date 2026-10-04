# Pencak Silat Digital Scoring System

Backend system untuk **Pencak Silat Kategori Tanding** dengan real-time scoring, consensus juri, multi-arena support, dan tournament bracket management.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend API + WS | Go 1.27 (stdlib `net/http` + `gorilla/websocket`) |
| Rules Engine | Go struct-based, rule set dari JSON/DB |
| Database | PostgreSQL 16 (`pgx`/`lib/pq` driver) |
| Auth | PIN (bcrypt) + JWT (HS256) |
| Deployment | Single binary + systemd + Caddy reverse proxy |

---

## Project Structure

```
cmd/server/main.go           # Entry point & graceful shutdown
internal/
  api/                       # HTTP handlers, router, middleware (JWT, RBAC, CORS)
    handler/                 # REST handlers (auth, competition, match, arena, bracket, official, contingent, report, live)
    swagger.go               # Swagger UI + OpenAPI spec
  auth/                      # Random PIN generation (crypto/rand), bcrypt, JWT claims (role & seat)
  bracket/                   # Single-elimination bracket generator & tree traversal
  consensus/                 # Dynamic judge consensus engine (sliding window voting)
  hub/                       # WebSocket hub per arena (role-based broadcast & reconnect sync)
  logger/                    # Structured logging (slog)
  orchestrator/              # Match orchestrator, server-side timer, event deduplication
  rules/                     # Rules engine (data-driven dari JSON)
  service/                   # Business logic layer (match, comp, arena, bracket, official, contingent, report)
  statemachine/              # Match state machine (idle → ready → rounds → finished → cancelled)
  store/                     # Repository layer (PostgreSQL connection pool & queries)
rulesets/
  ipsi_2024.json             # IPSI 2024 ruleset
  persilat_v7.json           # PERSILAT ruleset
migrations/                  # SQL migrations (001 s/d 016)
tests/                       # 37 automated unit & QA test suite (100% PASS)
docker-compose.yml           # Local PostgreSQL
docs/openapi.yaml            # OpenAPI 3.0 spec & WebSocket protocol contracts
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
- Reconnection + full state snapshot recovery

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
| **Juri Gelanggang** | Tablet Juri | 6-digit random PIN → JWT | Scoped ke: `competition_id`, `arena_id`, `role: judge`, `seat_number: 1..n` |
| **Kontingen** | Manajer Tim / Pelatih | Email & Password (bcrypt) → JWT | 24 jam (`role: contingent`) |

- **Clean Architecture Juri**: Pemisahan tegas antara otorisasi (`role: "judge"`) dan nomor kursi fisik gelanggang (`seat_number`). Mendukung format juri dinamis (1, 3, 5, dst).
- **Server-Side PIN**: PIN di-generate otomatis secara kriptografis (`crypto/rand`) dengan expiration time (`expires_at`).

### 8. Pendaftaran Atlet & Verifikasi Panitia (Contingent Workflow)
- **Portal Kontingen**: Perwakilan kontingen dapat mendaftar mandiri (`/api/v1/auth/contingent/register`) dan mendaftarkan atlet asuhannya.
- **Status Registrasi**: Atlet yang didaftarkan kontingen otomatis berstatus `pending`.
- **Meja Timbang / Verifikasi**: Panitia (Admin / Dewan) memverifikasi berkas & timbang badan peserta (`PUT .../participants/{id}/verify`) menjadi `verified` atau `rejected`.
- **Bracket Safeguard**: Hanya atlet berstatus `verified` yang dapat dimasukkan ke dalam bagan pertandingan single-elimination.

### 9. Database (PostgreSQL 16) - 16 Versioned Migrations
Tabel database terstruktur & termigrasi lengkap:
- `competitions`, `arenas`, `categories`, `rulesets`
- `athletes`, `contingents`, `participants` (dengan `registration_status`), `officials`
- `judge_pins` (dengan `seat_number INT`), `matches`, `match_scores`, `match_events` (audit trail), `bracket_slots`

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

### 3. Run Migrations
```bash
# Using golang-migrate
migrate -path migrations -database "$DB_URL" up
```

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
| POST | `/api/v1/auth/judge-login` | Public | Login Juri gelanggang via PIN 6-digit → JWT (claims: `role: judge`, `seat_number`) |
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
| GET | `/api/v1/live/scoreboard/{arena_id}` | Public | Live scoreboard feed publik (polling fallback) |
| GET | `/ws/arena/{arena_id}?token=<JWT>` | Query Token | Real-time bi-directional channel gelanggang (Websocket) |

---

## Message Types (WebSocket)

### Client → Server
| Type | From | Payload |
|------|------|---------|
| `score` | Judge | `{technique, target, round}` |
| `penalty_recommend` | Judge | `{type, level, target}` |
| `match_start` | Referee | `{}` |
| `match_pause` | Referee | `{reason}` |
| `match_resume` | Referee | `{}` |
| `penalty_apply` | Referee | `{type, level, target, points}` |
| `override` | Admin | `{ref_event_id, new_points, reason}` |

### Server → Client
| Type | To | Payload |
|------|----|---------|
| `ack` | Sender | `{seq, status}` |
| `state_update` | All | `{state, round, timer}` |
| `score_confirmed` | All except judge | `{technique, points, judges[], target}` |
| `score_unconfirmed` | Judge | `{technique, reason}` |
| `timer_sync` | All | `{remaining_ms, running, round, server_ts}` |
| `penalty_applied` | All | `{type, level, points, target, by}` |
| `match_result` | All | `{winner, result_type, scores}` |
| `error` | Sender | `{code, message, ref_seq}` |

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

### Run Tests
```bash
go test -v -count=1 ./...
```
Menjalankan seluruh **40 automated unit & regression tests** (100% PASS) di folder `tests/` yang mencakup:
- Match State Machine & Server-side Timer
- Match Orchestrator & Crash Recovery
- Consensus Engine (Majority, Deduplication, Disqualification)
- Single Elimination Bracket Generator & Tree Progression
- 3 Auth Flows (Officials bcrypt, Judge PIN/Seat, Contingent Portal)
- Participant Verification Workflow & Bracket Safeguard
- Dynamic Rulesets CRUD & Database-driven Engine Binding
- Idempotent Score Submission (`Idempotency-Key`)
- Admin Reports (Summary, Medals, Full JSON Export)

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