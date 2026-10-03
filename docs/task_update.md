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

## Pending
- Write unit tests for `MatchService` lifecycle methods and consensus engine.
- Verify status codes and payloads for all REST endpoints against `docs/openapi.yaml`.
- Ensure `FailMsg()` helper is consistently used where appropriate.
- Optimize OpenAPI specification with newly added category and athlete endpoints.
- Deploy changes to staging and run integration tests.
