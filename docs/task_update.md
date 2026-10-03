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
    - Updated `docs/openapi.yaml` with complete two-way WebSocket protocol schemas (`score_event`, `referee_command`, `penalty_event`, `state_change`, `score_update`, `score_confirmed`, `match_result`, `error`).
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

## Pending
- Write unit tests for consensus engine edge cases.
- Verify status codes and payloads for all REST endpoints against `docs/openapi.yaml`.
- Ensure `FailMsg()` helper is consistently used where appropriate.
- Deploy changes to staging and run integration tests.
