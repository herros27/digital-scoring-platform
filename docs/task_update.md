# Task Update

## Completed
1. **Auth Handler** – Added missing `nil` argument to all `Fail()` calls.
2. **Bracket Handler** – Added missing `nil` argument to `Fail()` calls.
3. **Match Handler** –
   - Removed unused `GetOrchestrator()` call.
   - Updated lifecycle routes (`Start`, `Pause`, `Resume`, `Finish`) to directly use service methods and handle errors correctly.
4. **Match Service** –
   - Implemented `LoadMatchToArena()`, `GetOrchestrator()`, `StartMatch()`, `PauseMatch()`, `ResumeMatch()`, `FinishMatch()`.
5. **Orchestrator** – Added `GetMatchID()` getter.
6. **Architecture & Clean Code Refactoring** –
   - Created `internal/domain` containing core models and domain sentinel errors.
   - Centralized environment & app settings in `internal/config`.
   - Tuned DB connection pool in `internal/store/db.go`.
   - Added graceful shutdown with signal handling and server timeout settings in `cmd/server/main.go`.
7. **Category Management** –
   - Implemented `CategoryService` (`internal/service/category_service.go`) and `CategoryHandler` (`internal/api/handler/category.go`).
   - Added CRUD endpoints for competition categories under `/api/v1/competitions/{comp_id}/categories`.
8. **Athlete & Participant Management** –
   - Implemented `AthleteService` (`internal/service/athlete_service.go`) and `AthleteHandler` (`internal/api/handler/athlete.go`).
   - Extended `AthleteStore` to support master athletes and competition registration.
   - Added endpoints under `/api/v1/athletes` and `/api/v1/competitions/{comp_id}/athletes`.
9. **Role-Based Access Control (RBAC)** –
   - Implemented `RequireRole` middleware in `internal/api/middleware.go`.
   - Protected administrative endpoints (`admin`, `dewan`) against unauthorized access from `juri` or other scoped tokens.
10. **Match Lifecycle Routing & Arena Concurrency (P0)** –
    - Mounted `start`, `pause`, `resume`, `finish`, `cancel` in `internal/api/router.go`.
    - Added `ARENA_BUSY` validation (HTTP 409 Conflict) in `MatchService.LoadMatchToArena`.
    - Implemented `CancelMatch` and `StateCancelled` in statemachine, orchestrator, service, and handler.
    - Fixed timer double unlock bug in `Timer.Pause()`.
    - Implemented unit tests in `tests/match_service_test.go` covering full lifecycle and concurrency.
11. **Match Score Breakdown API (P0)** –
    - Implemented `GetMatchScores()` in `MatchService` and `GetScores` handler in `MatchHandler`.
    - Mounted `GET /api/v1/competitions/{comp_id}/matches/{id}/scores` in `internal/api/router.go`.
    - Added automated unit test in `tests/match_service_test.go`.
12. **OpenAPI Spec & WebSocket Contract Documentation (P0)** –
    - Updated `docs/openapi.yaml` with complete two-way WebSocket protocol schemas (klien: `judge_input`, `state_change` + `action`, `penalty`, `sync_request`; server: `sync_response`, `state_change`, `score_update`, `score_confirmed`, `match_result`, `error`; dikoreksi mengikuti kode backend).
    - Documented Category, Athlete, Match Lifecycle, and Score endpoints.
13. **Automatic Bracket Advancement (P1)** –
    - Added `AdvanceWinner()` in `BracketStore` (`internal/store/bracket.go`).
    - Integrated with `Orchestrator.finalize()`: upon match completion, the winner participant is resolved and auto-advanced to `next_slot_id` in the single-elimination tree.
    - Added automated unit test in `tests/p1_features_test.go`.
14. **Ruleset Management API (P1)** –
    - Implemented `RulesetService` (`internal/service/ruleset_service.go`) and `RulesetHandler` (`internal/api/handler/ruleset.go`).
    - Handled JSONB serialization in `RulesetStore` and added fallback to builtin rulesets (IPSI 2024, PERSILAT).
    - Mounted routes: `GET /api/v1/rulesets` and `GET /api/v1/rulesets/{id}`.
    - Added automated unit test in `tests/p1_features_test.go`.
15. **Match Events Audit Trail API (P1)** –
    - Implemented `GetMatchEvents()` in `MatchService` and `GetEvents` in `MatchHandler`.
    - Mounted route: `GET /api/v1/competitions/{comp_id}/matches/{id}/events`.
    - Documented schema `MatchEvent` in `docs/openapi.yaml` and added automated unit test in `tests/p1_features_test.go`.
16. **WebSocket Reconnect & State Synchronization (P1)** –
    - Added message types `sync_request` and `sync_response` in `internal/hub/message.go`.
    - Handled `sync_request` in `Orchestrator.handleSyncRequest()`, delivering immediate snapshot state of round, timer, and scores to reconnected tablets/screens.
    - Documented protocol contract in `docs/openapi.yaml` and added automated unit test in `tests/p1_features_test.go`.
17. **Server-Side Random PIN Generation (P2)** –
    - Added cryptographic numeric PIN generator `auth.GenerateRandomPIN()` in [`internal/auth/pin.go`](file:///d:/Freelance/DigitalScoring/internal/auth/pin.go) using `crypto/rand`.
    - Updated `AuthHandler.CreatePIN` in [`internal/api/handler/auth.go`](file:///d:/Freelance/DigitalScoring/internal/api/handler/auth.go) to auto-generate a 6-digit PIN if empty and return expiration details (`expires_at`).
    - Documented request and response schema in [`docs/openapi.yaml`](file:///d:/Freelance/DigitalScoring/docs/openapi.yaml).
    - Added unit test in [`tests/p2_features_test.go`](file:///d:/Freelance/DigitalScoring/tests/p2_features_test.go).
18. **Pagination & Filtering (P2)** –
    - Added `Pagination` struct, `ParsePagination()`, and `OkPaged()` in [`internal/api/handler/respond.go`](file:///d:/Freelance/DigitalScoring/internal/api/handler/respond.go).
    - Added `ListPaged` methods in `CompetitionStore`, `MatchStore`, and `AthleteStore` with SQL pagination and row counting.
    - Updated `GET /competitions` (status filter), `GET .../matches` (status filter), and `GET /athletes` (gender filter) with `?page=1&limit=20`.
    - Documented query parameters and `Pagination` schema in [`docs/openapi.yaml`](file:///d:/Freelance/DigitalScoring/docs/openapi.yaml).
    - Added unit test in [`tests/p2_features_test.go`](file:///d:/Freelance/DigitalScoring/tests/p2_features_test.go).
19. **Idempotency Key pada Scoring Event (P2)** –
    - Added `seenEvents` registry with TTL and mutex protection in `Orchestrator` ([`internal/orchestrator/orchestrator.go`](file:///d:/Freelance/DigitalScoring/internal/orchestrator/orchestrator.go)).
    - Supported `event_id` and `idempotency_key` deduplication on WebSocket judge scoring inputs.
    - Added `POST /api/v1/competitions/{comp_id}/matches/{id}/scores` with `Idempotency-Key` header and payload fallback to prevent accidental double-tap scoring.
    - Documented endpoint and schema in [`docs/openapi.yaml`](file:///d:/Freelance/DigitalScoring/docs/openapi.yaml).
20. **Comprehensive Unit Test Suite & QA (Ops)** –
    - Created isolated unit test suites in [`tests/`](file:///d:/Freelance/DigitalScoring/tests):
      - [`tests/match_service_test.go`](file:///d:/Freelance/DigitalScoring/tests/match_service_test.go): Full lifecycle flow and concurrency (`ARENA_BUSY`).
      - [`tests/consensus_engine_test.go`](file:///d:/Freelance/DigitalScoring/tests/consensus_engine_test.go): Majority voting (2/3), sliding window (3s), duplicate suppression, mismatch filters, and concurrency.
      - [`tests/bracket_generator_test.go`](file:///d:/Freelance/DigitalScoring/tests/bracket_generator_test.go): Single-elimination tree generation, BYE handling, auto-advancing BYE opponents, and next slot linking.
      - [`tests/statemachine_and_rules_test.go`](file:///d:/Freelance/DigitalScoring/tests/statemachine_and_rules_test.go): Normal round flow, invalid transitions, early finish, cancel, and ruleset loader (IPSI 2024 / PERSILAT).
      - [`tests/auth_jwt_test.go`](file:///d:/Freelance/DigitalScoring/tests/auth_jwt_test.go): JWT token issuance/verification, expiration, signature validation, and bcrypt PIN hashing/verification.
      - [`tests/category_service_test.go`](file:///d:/Freelance/DigitalScoring/tests/category_service_test.go): Input validation, category creation, retrieval, and listing.
      - [`tests/p1_features_test.go`](file:///d:/Freelance/DigitalScoring/tests/p1_features_test.go): Bracket advancement, rulesets, match events, and WebSocket sync.
      - [`tests/p2_features_test.go`](file:///d:/Freelance/DigitalScoring/tests/p2_features_test.go): Server PIN generator, pagination & filtering, and scoring idempotency.
      - [`tests/admin_features_test.go`](file:///d:/Freelance/DigitalScoring/tests/admin_features_test.go): Arena CRUD and Competition lifecycle.
    - Verified all 32 test cases passing (`go test -count=1 ./...` exits 0, `go vet ./...` clean).
21. **Arena CRUD Lanjutan & Competition Lifecycle (P2)** –
    - Implemented `GetArenaByID`, `UpdateArena`, and `DeleteArena` in [`ArenaStore`](file:///d:/Freelance/DigitalScoring/internal/store/arena.go), [`ArenaService`](file:///d:/Freelance/DigitalScoring/internal/service/arena_service.go), and [`ArenaHandler`](file:///d:/Freelance/DigitalScoring/internal/api/handler/arena.go).
    - Implemented `UpdateCompetition`, `PauseCompetition`, `FinishCompetition`, and `CancelCompetition` in [`CompetitionStore`](file:///d:/Freelance/DigitalScoring/internal/store/competition.go), [`CompetitionService`](file:///d:/Freelance/DigitalScoring/internal/service/competition_service.go), and [`CompetitionHandler`](file:///d:/Freelance/DigitalScoring/internal/api/handler/competition.go).
    - Mounted new endpoints in [`internal/api/router.go`](file:///d:/Freelance/DigitalScoring/internal/api/router.go) protected with RBAC (`admin`, `dewan`).
    - Documented all endpoints in [`docs/openapi.yaml`](file:///d:/Freelance/DigitalScoring/docs/openapi.yaml) and verified via automated test in [`tests/admin_features_test.go`](file:///d:/Freelance/DigitalScoring/tests/admin_features_test.go).
22. **Admin/Operator Management & Reporting, Statistics, Export (P2)** –
    - Implemented `OfficialStore` ([`internal/store/official.go`](file:///d:/Freelance/DigitalScoring/internal/store/official.go)), `OfficialService` ([`internal/service/official_service.go`](file:///d:/Freelance/DigitalScoring/internal/service/official_service.go)), and `OfficialHandler` ([`internal/api/handler/official.go`](file:///d:/Freelance/DigitalScoring/internal/api/handler/official.go)) with password hashing via bcrypt, 24h JWT issuance, duplicate email checks, and full CRUD (`GET`, `POST`, `PUT`, `DELETE /api/v1/officials`).
    - Implemented `ReportService` ([`internal/service/report_service.go`](file:///d:/Freelance/DigitalScoring/internal/service/report_service.go)) and `ReportHandler` ([`internal/api/handler/report.go`](file:///d:/Freelance/DigitalScoring/internal/api/handler/report.go)) with `GET .../report/summary`, `GET .../report/medals`, and `GET .../report/export`.
    - Mounted public login endpoint `POST /api/v1/auth/login` and protected routes in [`internal/api/router.go`](file:///d:/Freelance/DigitalScoring/internal/api/router.go).
    - Documented paths and schemas (`Official`, `CompetitionSummary`, `MedalStanding`, `CompetitionExport`) in [`docs/openapi.yaml`](file:///d:/Freelance/DigitalScoring/docs/openapi.yaml).
    - Added automated unit & HTTP integration tests in [`tests/official_and_report_test.go`](file:///d:/Freelance/DigitalScoring/tests/official_and_report_test.go).
23. **Option A: Clean Architecture — Pemisahan Role & Seat Juri** –
    - Decoupled judge position from authorization role: `role: "judge"` (RBAC) dan `seat_number: 1, 2, ...` (penempatan gelanggang).
    - Added database migration `014_judge_pins_seat_number.sql` (`ALTER TABLE judge_pins ADD COLUMN seat_number INT DEFAULT 0`).
    - Added backward-compatible auto-normalization in `internal/auth/pin.go` (`NormalizeRoleAndSeat`), mapping legacy `"judge_1"` into `("judge", 1)`.
    - Updated `MatchScoreDetail` response with `consensus` and `judges` breakdown per seat.
    - Added unit test coverage in `tests/auth_jwt_test.go`.
24. **Database Reset & Migrasi Penuh (001 - 016)** –
    - Added `015_officials.sql` for management & official accounts.
    - Refactored `011_participants.sql` to be safe & idempotent.
    - Executed clean schema reset and sequential execution of all 16 migration files against local PostgreSQL container.
25. **Option 2: Manajemen Kontingen & Alur Verifikasi Pendaftaran Atlet (Weigh-in / Document Approval)** –
    - Created dedicated table `contingents` (`migrations/016_contingents.sql`) linked to `athletes` and `participants`.
    - Implemented `ContingentStore`, `ContingentService`, and `ContingentHandler` with self-registration (`POST /api/v1/auth/contingent/register`), login (`POST /api/v1/auth/contingent/login`), and listing (`GET /api/v1/contingents`).
    - Supported athlete registration by contingent with automatic `pending` status (`POST /api/v1/competitions/{comp_id}/athletes`).
    - Added verification endpoints for Admin & Dewan (`PUT /api/v1/competitions/{comp_id}/participants/{id}/verify` and `GET .../participants?status=`).
    - Enforced bracket generation safeguard in `BracketHandler`: only `verified` participants enter tournament brackets.
26. **Dynamic Rulesets API & Database-driven Engine Integration** –
    - Implemented `POST /api/v1/rulesets`, `PUT /api/v1/rulesets/{id}`, and `DELETE /api/v1/rulesets/{id}` in [`RulesetHandler`](file:///d:/Freelance/DigitalScoring/internal/api/handler/ruleset.go), protected with RBAC (`admin`, `dewan`).
    - Added `CreateRuleset`, `UpdateRuleset`, `DeleteRuleset`, and `ResolveRuleset` in [`RulesetService`](file:///d:/Freelance/DigitalScoring/internal/service/ruleset_service.go).
    - Added `Update` and `Delete` methods in [`RulesetStore`](file:///d:/Freelance/DigitalScoring/internal/store/rulseset.go).
    - Integrated dynamic ruleset resolution into `MatchService.LoadMatchToArena`: matches automatically load their ruleset directly from the PostgreSQL database (via competition `ruleset_id` or explicit ruleset identifier) without requiring manual JSON file creation or project rebuilds.
    - Added comprehensive unit and HTTP integration tests in [`tests/ruleset_crud_test.go`](file:///d:/Freelance/DigitalScoring/tests/ruleset_crud_test.go), bringing total test suite to **40 automated unit tests (100% PASS)**.
    - Updated documentation in [`docs/openapi.yaml`](file:///d:/Freelance/DigitalScoring/docs/openapi.yaml) and [`README.md`](file:///d:/Freelance/DigitalScoring/README.md).
27. **Rate Limiting Middleware (`golang.org/x/time/rate`)** –
    - Implemented in-memory token bucket rate limiter in [`internal/api/middleware.go`](file:///d:/Freelance/DigitalScoring/internal/api/middleware.go) with background garbage collection (3-minute ticker).
    - Extracted client IP from `X-Forwarded-For`, `X-Real-IP`, and `RemoteAddr` with port stripping.
    - Applied strict auth limiter (10 req/min, burst 10) to `/auth/judge-login`, `/auth/pin`, `/auth/login`, `/auth/contingent/register`, `/auth/contingent/login`.
    - Applied global limiter (60 req/s, burst 100) to all HTTP endpoints via `Chain()`.
28. **Upgrade Driver Database ke `jackc/pgx/v5`** –
    - Completely replaced deprecated `lib/pq` with `github.com/jackc/pgx/v5/stdlib`.
    - Updated `internal/store/db.go` with `sql.Open("pgx", dsn)` while preserving production connection pool tuning.
    - Cleaned transitives with `go mod tidy`.
29. **Struct Validator (`go-playground/validator/v10`)** –
    - Integrated struct validation into [`internal/api/handler/respond.go`](file:///d:/Freelance/DigitalScoring/internal/api/handler/respond.go) with `Decode()` and `Bind()`.
    - Mapped validation error names to `json` tags for client-friendly error reporting.
30. **In-Memory Caching & Prometheus Observability** –
    - Added `internal/cache/` TTL cache for tournament reports (`/report/summary`, `/report/medals`) with concurrent invalidation on match completion.
    - Added `internal/metrics/` exposing standard Prometheus metrics (`/metrics`) for HTTP latency, WebSocket connection gauges, and consensus duration histograms.
31. **Refactoring Struktur Pengujian (Clean Unit Test Co-location)** –
    - Relocated pure unit tests directly into `internal/` alongside their source code (`internal/auth`, `internal/bracket`, `internal/consensus`, `internal/statemachine`, `internal/rules`, `internal/service`, `internal/cache`, `internal/metrics`, `internal/api`).
    - Dedicated `tests/` folder exclusively for end-to-end tournament scenario integration tests (`contingent_test.go`, `match_service_test.go`, `official_and_report_test.go`, `ruleset_crud_test.go`, `admin_features_test.go`, `p1_features_test.go`, `p2_features_test.go`).
32. **Public Health Check Endpoint (`GET /api/v1/health`)** –
    - Mounted `GET /api/v1/health` in `internal/api/router.go` checking database ping and reporting service status.
33. **Sinkronisasi Multi-Device & Penyimpanan Persisten PIN Gelanggang (`GET /api/v1/auth/pins` & Migrasi 017)** –
    - Menambahkan migrasi database [`migrations/017_judge_pins_raw_pin.sql`](file:///d:/Freelance/DigitalScoring/migrations/017_judge_pins_raw_pin.sql) untuk kolom `raw_pin VARCHAR(20)` pada tabel `judge_pins`.
    - Mengimplementasikan arsitektur penyimpanan ganda: `pin_hash` (bcrypt) untuk verifikasi login tablet juri/wasit (`POST /api/v1/auth/judge-login`), dan `raw_pin` untuk inspeksi authorized admin serta sinkronisasi multi-device panitia.
    - Menambahkan endpoint `GET /api/v1/auth/pins?competition_id={compId}&arena_id={arenaId}` di [`internal/api/handler/auth.go`](file:///d:/Freelance/DigitalScoring/internal/api/handler/auth.go) yang dilindungi RBAC (`admin`, `dewan`).
    - Mekanisme server-side auto-provisioning: jika kursi gelanggang (Juri 1–5 atau Wasit) belum memiliki PIN aktif atau sudah kadaluarsa, backend otomatis mem-generate PIN numerik kriptografis 6-digit baru (TTL 12 jam) dan menyimpannya secara persisten ke database PostgreSQL.
    - Pembersihan duplikasi (stale cleanup): `POST /api/v1/auth/create-pin` secara otomatis menghapus rekaman PIN lama pada kursi/role yang sama sebelum menyimpan PIN baru.
    - Integrasi frontend: [`apps/web/src/views/admin/DevicePinManager.vue`](file:///d:/Freelance/DigitalScoring/apps/web/src/views/admin/DevicePinManager.vue) beralih sepenuhnya ke sinkronisasi database server via `listPins`, mengeliminasi dependensi lokal `localStorage` dan string PIN hardcoded/palsu.
    - Dokumentasi OpenAPI diperbarui di [`docs/openapi.yaml`](file:///d:/Freelance/DigitalScoring/docs/openapi.yaml) dan kredensial di [`docs/CREDENTIALS.md`](file:///d:/Freelance/DigitalScoring/docs/CREDENTIALS.md).
34. **Validasi Keunikan Nama Gelanggang & Safe Deletion (`ArenaService` & `ArenaStore`)** –
    - Menambahkan validasi `strings.EqualFold` pada [`ArenaService.CreateArena`](file:///d:/Freelance/DigitalScoring/internal/service/arena_service.go) dan `UpdateArena` untuk mencegah duplikasi nama gelanggang kembar dalam kompetisi yang sama.
    - Menambahkan mekanisme safe deletion pada [`ArenaStore.Delete`](file:///d:/Freelance/DigitalScoring/internal/store/arena.go) untuk membersihkan relasi foreign key (`judge_pins` dan `matches.arena_id = NULL`) secara aman sebelum baris arena dihapus.
35. **Validasi Eksistensi Gelanggang pada Pembuatan PIN & Defensive Guard Frontend** –
    - Backend: Menambahkan `ErrArenaNotFound` pada [`internal/auth/pin.go`](file:///d:/Freelance/DigitalScoring/internal/auth/pin.go) yang memverifikasi eksistensi gelanggang di database sebelum melakukan insert ke tabel `judge_pins`. Di handler [`internal/api/handler/auth.go`](file:///d:/Freelance/DigitalScoring/internal/api/handler/auth.go), mengembalikan status HTTP 404 (`NOT_FOUND`) dengan pesan ramah jika ID gelanggang tidak valid atau sudah dihapus.
    - Frontend: Memperbaiki [`apps/web/src/views/admin/DevicePinManager.vue`](file:///d:/Freelance/DigitalScoring/apps/web/src/views/admin/DevicePinManager.vue) dengan menghapus fallback ID statis palsu, menambahkan hook `onMounted` dan sinkronisasi otomatis `refetchArenas()`, serta menyematkan `getValidArenaId()` guard sebelum proses generate PIN agar tidak pernah mengirimkan ID gelanggang basi (*stale*) ke server.


36. **Perbaikan WebSocket, Akses Penonton Publik, dan Siaran Event** –
    - `MetricsMiddleware` (`statusRecorder`) kini meneruskan `http.Hijacker`/`Flusher`; sebelumnya seluruh upgrade WebSocket gagal dengan "response does not implement http.Hijacker".
    - `GET /ws/arena/{arena_id}` tanpa token diterima sebagai penonton publik (role `scoreboard`, read-only) hanya untuk arena yang sudah punya hub; selain itu 404. Token PIN harus cocok dengan arena; token login email (tanpa arena) hanya untuk `admin`, `dewan`, `wasit`.
    - Role JWT `wasit`/`dewan` dipetakan ke `hub.RoleReferee` (`hub.RoleFromClaim`) sehingga wasit dapat start/pause/penalty.
    - Orchestrator menyiarkan `score_confirmed`; `state_change` membawa `timer_running` (jeda tidak lagi tampak sebagai `round_N` yang berjalan).
    - `GET /api/v1/live/scoreboard/{arena_id}` mengembalikan snapshot pertandingan aktif (status, ronde, timer, total skor, nama atlet) lewat `MatchService.GetLiveSnapshot`.
    - Dokumentasi: `docs/openapi.yaml` dan README disamakan dengan protokol sebenarnya.
37. **Login Juri/Wasit Hanya dengan PIN (Migrasi 018)** –
    - `POST /api/v1/auth/judge-login` menerima `{ "pin" }` saja; turnamen, gelanggang, role, dan kursi ditentukan server dari PIN.
    - Kolom `judge_pins.pin_lookup` (HMAC-SHA256 dari PIN dengan kunci turunan JWT secret, indeks unik) untuk pencarian O(1); bcrypt diverifikasi sekali setelah ketemu. Migrasi menghapus PIN lama (tahap development).
    - PIN unik di seluruh sistem (`ErrPinTaken` -> HTTP 409); PIN acak dibuat ulang otomatis bila bentrok (`PinStore.CreateWithRandomPIN`). PIN juri 6 digit, PIN wasit 8 digit; PIN kustom divalidasi.
    - `auth.AttemptLimiter`: 5 PIN salah beruntun dalam 15 menit mengunci IP 15 menit (HTTP 429 + `Retry-After`).
    - Seeder: PIN wasit `11110001` (Gelanggang A) dan `22220001` (Gelanggang B).
    - Tes database nyata di `tests/pin_login_test.go`; CI menyediakan service Postgres (`TEST_DB_URL`). Simulasi `apps/web/scripts/e2e-simulation.mjs` kini memverifikasi (keluar kode 1 bila gagal).

## Status Backend Development & UAT Readiness
- **Development Backend**: **100% SELESAI** untuk seluruh kebutuhan fitur core, business logic, WebSocket, scoring, audit trail, security hardening, dan database migration.
- **Test Suite**: **100% PASS** (`go test -race ./...` termasuk tes database nyata untuk login PIN; simulasi e2e REST+WebSocket lolos).
- **UAT / Requirement Readiness**: Memenuhi 100% spesifikasi fungsional backend sesuai `architecture-backend.md` dan `workLog/api-missing-requirements.md`.

## Next Steps (Production Deployment & Operations)
- UAT pengujian antarmuka frontend (Tablet Juri, Scoreboard, dan Admin UI) terhadap backend.
- Setup Reverse Proxy TLS/HTTPS (Caddy / Nginx) di server venue / VPS.
