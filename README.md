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
cmd/server/main.go           # Entry point
internal/
  api/                       # HTTP handlers, router, middleware
    handler/                 # REST handlers (auth, competition, match, arena, bracket, live)
    swagger.go               # Swagger UI + OpenAPI spec
  auth/                      # PIN auth + JWT tokens
  bracket/                   # Single-elimination bracket generator
  consensus/                 # Judge consensus engine (sliding window voting)
  hub/                       # WebSocket hub per arena (role-based broadcast)
  logger/                    # Structured logging (slog)
  orchestrator/              # Match orchestrator + server-side timer
  rules/                     # Rules engine (data-driven dari JSON)
  service/                   # Business logic layer
  statemachine/              # Match state machine (idle → ready → rounds → finished)
  store/                     # Repository layer (PostgreSQL)
rulesets/
  ipsi_2024.json             # IPSI 2024 ruleset
  persilat_v7.json           # PERSILAT ruleset
migrations/                  # golang-migrate compatible SQL files
docker-compose.yml           # Local PostgreSQL
docs/openapi.yaml            # OpenAPI 3.0 spec
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

### 7. Auth & Session
| Flow | User | Method | Scope |
|------|------|--------|-------|
| Judge/Referee | Tablet | 6-digit PIN → JWT | `competition_id` + `arena_id` + `role` |
| Admin | Dashboard | Email/password → JWT | Full access |

- PIN: bcrypt hash, per-competition, TTL default 12 jam
- JWT: HS256, scoped claims (`comp_id`, `arena_id`, `role`)
- Multi-device prevention: login baru revoke session lama

### 8. Database (PostgreSQL) - Event Sourcing
Key tables:
- `competitions`, `rule_sets`, `arenas`
- `athletes`, `officials`, `judge_sessions`
- `matches`, `match_events` (immutable audit trail), `match_scores` (materialized view)
- `brackets`, `bracket_slots`, `match_schedule`
- `match_results` (final certified result)

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

### Authentication
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/judge-login` | Public | PIN login → JWT |
| POST | `/api/v1/auth/create-pin` | Admin | Generate PIN untuk juri |

### Competition
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/competitions` | Admin | List kompetisi |
| POST | `/api/v1/competitions` | Admin | Buat kompetisi |
| GET | `/api/v1/competitions/{id}` | Admin | Detail kompetisi |
| PUT | `/api/v1/competitions/{id}/activate` | Admin | Activate kompetisi |

### Arena
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/competitions/{comp_id}/arenas` | Admin | List arena |
| POST | `/api/v1/competitions/{comp_id}/arenas` | Admin | Tambah arena |

### Match
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/competitions/{comp_id}/matches` | Admin | Buat match |
| GET | `/api/v1/competitions/{comp_id}/matches/{id}` | Admin | Detail match |
| POST | `/api/v1/competitions/{comp_id}/matches/{id}/load-arena` | Admin | Load match ke arena (init orchestrator) |

### Bracket
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/competitions/{comp_id}/brackets/generate` | Admin | Generate single-elimination bracket |
| GET | `/api/v1/competitions/{comp_id}/brackets` | Admin | Get bracket tree |

### Live Scoreboard (Public)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/live/scoreboard/{arena_id}` | None | Live scoreboard feed |

### WebSocket
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/ws/arena/{arena_id}?token=<JWT>` | Query param | Real-time arena channel |

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
Simpan di `rulesets/{name}.json` — auto-loaded by `rules.LoadRuleset(name)`.

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
go test ./...
```

### Generate Migration
```bash
migrate create -ext sql -dir migrations -seq description
```

### Add New Ruleset
1. Create `rulesets/myruleset.json`
2. Call `rules.LoadRuleset("myruleset")` — auto works

---

## Architecture References

- **Full Backend Architecture**: [`architecture-backend.md`](architecture-backend.md) — comprehensive design doc
- **Development Log**: [`workLog/task.md`](workLog/task.md) — feature history
- **OpenAPI Spec**: [`docs/openapi.yaml`](docs/openapi.yaml)

---

## License

Internal project — Digital Scoring for Pencak Silat.