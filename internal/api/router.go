package api

import (
	"database/sql"
	"net/http"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
)

func NewRouter(db *sql.DB, hubManager *hub.Manager) http.Handler {
	// stores
	compRepo := &store.CompetitionStore{DB: db}
	arenaRepo := &store.ArenaStore{DB: db}
	matchRepo := &store.MatchStore{DB: db}
	scoreRepo := &store.MatchScoreStore{DB: db}
	pinRepo := &auth.PinStore{DB: db}
	bracketRepo := &store.BracketStore{DB: db}
	categoryRepo := &store.CategoryStore{DB: db}
	athleteRepo := &store.AthleteStore{DB: db}
	rulesetRepo := &store.RulesetStore{DB: db}
	officialRepo := &store.OfficialStore{DB: db}
	contingentRepo := &store.ContingentStore{DB: db}

	// services
	compService := service.NewCompetitionService(compRepo)
	arenaService := service.NewArenaService(arenaRepo, compRepo)
	matchService := service.NewMatchService(matchRepo, compRepo, hubManager, &store.MatchEventStore{DB: db}, scoreRepo, matchRepo, bracketRepo)
	bracketService := service.NewBracketService(bracketRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	athleteService := service.NewAthleteService(athleteRepo)
	rulesetService := service.NewRulesetService(rulesetRepo)
	matchService.SetRulesetResolver(rulesetService)
	officialService := service.NewOfficialService(officialRepo)
	contingentService := service.NewContingentService(contingentRepo)
	reportService := service.NewReportService(compRepo, arenaRepo, categoryRepo, athleteRepo, matchRepo)

	// handlers
	authH := &handler.AuthHandler{Pins: pinRepo}
	compH := &handler.CompetitionHandler{Service: compService}
	arenaH := &handler.ArenaHandler{Service: arenaService}
	matchH := &handler.MatchHandler{Service: matchService, HubManager: hubManager}
	liveH := &handler.LiveHandler{ScoreStore: scoreRepo, MatchStore: matchRepo}
	bracketH := &handler.BracketHandler{Service: bracketService, AthleteService: athleteService}
	categoryH := &handler.CategoryHandler{Service: categoryService}
	athleteH := &handler.AthleteHandler{Service: athleteService}
	rulesetH := &handler.RulesetHandler{Service: rulesetService}
	officialH := &handler.OfficialHandler{Service: officialService}
	contingentH := &handler.ContingentHandler{Service: contingentService}
	reportH := &handler.ReportHandler{Service: reportService}

	mux := http.NewServeMux()

	// === Swagger Documentation (Public) ===
	mux.HandleFunc("GET /docs", SwaggerHandler())
	mux.HandleFunc("GET /docs/openapi.yaml", OpenAPISpecHandler("docs/openapi.yaml"))

	// === Public endpoints (no auth) ===
	mux.HandleFunc("POST /api/v1/auth/judge-login", authH.JudgeLogin)
	mux.HandleFunc("POST /api/v1/auth/login", officialH.Login)
	mux.HandleFunc("POST /api/v1/auth/contingent/register", contingentH.Register)
	mux.HandleFunc("POST /api/v1/auth/contingent/login", contingentH.Login)
	mux.HandleFunc("GET /api/v1/live/scoreboard/{arena_id}", liveH.Scoreboard)

	// === Protected endpoints (JWT required) ===
	protected := http.NewServeMux()

	// RBAC roles
	adminOrDewan := RequireRole("admin", "dewan")
	matchOperators := RequireRole("admin", "dewan", "wasit")
	canRegisterAthlete := RequireRole("admin", "dewan", "contingent")

	// Auth & PIN (Admin only)
	protected.Handle("POST /api/v1/auth/create-pin", adminOrDewan(http.HandlerFunc(authH.CreatePIN)))

	// Competitions
	protected.HandleFunc("GET /api/v1/competitions", compH.List)
	protected.Handle("POST /api/v1/competitions", adminOrDewan(http.HandlerFunc(compH.Create)))
	protected.HandleFunc("GET /api/v1/competitions/{id}", compH.Get)
	protected.Handle("PUT /api/v1/competitions/{id}", adminOrDewan(http.HandlerFunc(compH.Update)))
	protected.Handle("PUT /api/v1/competitions/{id}/activate", adminOrDewan(http.HandlerFunc(compH.Activate)))
	protected.Handle("POST /api/v1/competitions/{id}/activate", adminOrDewan(http.HandlerFunc(compH.Activate)))
	protected.Handle("POST /api/v1/competitions/{id}/pause", adminOrDewan(http.HandlerFunc(compH.Pause)))
	protected.Handle("POST /api/v1/competitions/{id}/finish", adminOrDewan(http.HandlerFunc(compH.Finish)))
	protected.Handle("POST /api/v1/competitions/{id}/cancel", adminOrDewan(http.HandlerFunc(compH.Cancel)))

	// Categories
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/categories", categoryH.List)
	protected.Handle("POST /api/v1/competitions/{comp_id}/categories", adminOrDewan(http.HandlerFunc(categoryH.Create)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/categories/{id}", categoryH.Get)

	// Athletes & Participants
	protected.HandleFunc("GET /api/v1/athletes", athleteH.List)
	protected.Handle("POST /api/v1/athletes", canRegisterAthlete(http.HandlerFunc(athleteH.Create)))
	protected.HandleFunc("GET /api/v1/athletes/{id}", athleteH.Get)
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/athletes", athleteH.ListByCompetition)
	protected.Handle("POST /api/v1/competitions/{comp_id}/athletes", canRegisterAthlete(http.HandlerFunc(athleteH.RegisterToCompetition)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/participants", athleteH.ListParticipants)
	protected.Handle("PUT /api/v1/competitions/{comp_id}/participants/{id}/verify", adminOrDewan(http.HandlerFunc(athleteH.VerifyParticipant)))

	// Arenas
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/arenas", arenaH.List)
	protected.Handle("POST /api/v1/competitions/{comp_id}/arenas", adminOrDewan(http.HandlerFunc(arenaH.Create)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/arenas/{id}", arenaH.Get)
	protected.Handle("PUT /api/v1/competitions/{comp_id}/arenas/{id}", adminOrDewan(http.HandlerFunc(arenaH.Update)))
	protected.Handle("DELETE /api/v1/competitions/{comp_id}/arenas/{id}", adminOrDewan(http.HandlerFunc(arenaH.Delete)))

	// Matches
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/matches", matchH.List)
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches", adminOrDewan(http.HandlerFunc(matchH.Create)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/matches/{id}", matchH.Get)
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/load-arena", matchOperators(http.HandlerFunc(matchH.LoadArena)))
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/start", matchOperators(http.HandlerFunc(matchH.Start)))
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/pause", matchOperators(http.HandlerFunc(matchH.Pause)))
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/resume", matchOperators(http.HandlerFunc(matchH.Resume)))
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/finish", matchOperators(http.HandlerFunc(matchH.Finish)))
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/cancel", matchOperators(http.HandlerFunc(matchH.Cancel)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/matches/{id}/scores", matchH.GetScores)
	protected.HandleFunc("POST /api/v1/competitions/{comp_id}/matches/{id}/scores", matchH.SubmitScore)
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/matches/{id}/events", matchH.GetEvents)

	// Rulesets
	protected.HandleFunc("GET /api/v1/rulesets", rulesetH.List)
	protected.Handle("POST /api/v1/rulesets", adminOrDewan(http.HandlerFunc(rulesetH.Create)))
	protected.HandleFunc("GET /api/v1/rulesets/{id}", rulesetH.Get)
	protected.Handle("PUT /api/v1/rulesets/{id}", adminOrDewan(http.HandlerFunc(rulesetH.Update)))
	protected.Handle("DELETE /api/v1/rulesets/{id}", adminOrDewan(http.HandlerFunc(rulesetH.Delete)))

	// Brackets
	protected.Handle("POST /api/v1/competitions/{comp_id}/brackets/generate", adminOrDewan(http.HandlerFunc(bracketH.Generate)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/brackets", bracketH.GetTree)

	// Officials / Operator Management (Admin & Dewan)
	protected.Handle("GET /api/v1/officials", adminOrDewan(http.HandlerFunc(officialH.List)))
	protected.Handle("POST /api/v1/officials", adminOrDewan(http.HandlerFunc(officialH.Create)))
	protected.Handle("GET /api/v1/officials/{id}", adminOrDewan(http.HandlerFunc(officialH.Get)))
	protected.Handle("PUT /api/v1/officials/{id}", adminOrDewan(http.HandlerFunc(officialH.Update)))
	protected.Handle("DELETE /api/v1/officials/{id}", adminOrDewan(http.HandlerFunc(officialH.Delete)))

	// Contingents Management
	protected.Handle("GET /api/v1/contingents", adminOrDewan(http.HandlerFunc(contingentH.List)))
	protected.HandleFunc("GET /api/v1/contingents/{id}", contingentH.Get)

	// Reporting & Statistics & Export
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/report/summary", reportH.Summary)
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/report/medals", reportH.Medals)
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/report/export", reportH.Export)

	// Mount protected routes under AuthMiddleware
	mux.Handle("/api/v1/", Chain(protected, AuthMiddleware))

	// === WebSocket (auth via query param) ===
	mux.HandleFunc("GET /ws/arena/{arena_id}", matchH.HandleWS)

	// Apply CORS ke semua
	return Chain(mux, CORSMiddleware)
}