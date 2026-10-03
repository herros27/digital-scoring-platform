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

	// services
	compService := service.NewCompetitionService(compRepo)
	arenaService := service.NewArenaService(arenaRepo, compRepo)
	matchService := service.NewMatchService(matchRepo, compRepo, hubManager, &store.MatchEventStore{DB: db}, scoreRepo, matchRepo)
	bracketService := service.NewBracketService(bracketRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	athleteService := service.NewAthleteService(athleteRepo)

	// handlers
	authH := &handler.AuthHandler{Pins: pinRepo}
	compH := &handler.CompetitionHandler{Service: compService}
	arenaH := &handler.ArenaHandler{Service: arenaService}
	matchH := &handler.MatchHandler{Service: matchService, HubManager: hubManager}
	liveH := &handler.LiveHandler{ScoreStore: scoreRepo, MatchStore: matchRepo}
	bracketH := &handler.BracketHandler{Service: bracketService}
	categoryH := &handler.CategoryHandler{Service: categoryService}
	athleteH := &handler.AthleteHandler{Service: athleteService}

	mux := http.NewServeMux()

	// === Swagger Documentation (Public) ===
	mux.HandleFunc("GET /docs", SwaggerHandler())
	mux.HandleFunc("GET /docs/openapi.yaml", OpenAPISpecHandler("docs/openapi.yaml"))

	// === Public endpoints (no auth) ===
	mux.HandleFunc("POST /api/v1/auth/judge-login", authH.JudgeLogin)
	mux.HandleFunc("GET /api/v1/live/scoreboard/{arena_id}", liveH.Scoreboard)

	// === Protected endpoints (JWT required) ===
	protected := http.NewServeMux()

	// RBAC roles
	adminOrDewan := RequireRole("admin", "dewan")
	matchOperators := RequireRole("admin", "dewan", "wasit")

	// Auth & PIN (Admin only)
	protected.Handle("POST /api/v1/auth/create-pin", adminOrDewan(http.HandlerFunc(authH.CreatePIN)))

	// Competitions
	protected.HandleFunc("GET /api/v1/competitions", compH.List)
	protected.Handle("POST /api/v1/competitions", adminOrDewan(http.HandlerFunc(compH.Create)))
	protected.HandleFunc("GET /api/v1/competitions/{id}", compH.Get)
	protected.Handle("PUT /api/v1/competitions/{id}/activate", adminOrDewan(http.HandlerFunc(compH.Activate)))

	// Categories
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/categories", categoryH.List)
	protected.Handle("POST /api/v1/competitions/{comp_id}/categories", adminOrDewan(http.HandlerFunc(categoryH.Create)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/categories/{id}", categoryH.Get)

	// Athletes & Participants
	protected.HandleFunc("GET /api/v1/athletes", athleteH.List)
	protected.Handle("POST /api/v1/athletes", adminOrDewan(http.HandlerFunc(athleteH.Create)))
	protected.HandleFunc("GET /api/v1/athletes/{id}", athleteH.Get)
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/athletes", athleteH.ListByCompetition)
	protected.Handle("POST /api/v1/competitions/{comp_id}/athletes", adminOrDewan(http.HandlerFunc(athleteH.RegisterToCompetition)))

	// Arenas
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/arenas", arenaH.List)
	protected.Handle("POST /api/v1/competitions/{comp_id}/arenas", adminOrDewan(http.HandlerFunc(arenaH.Create)))

	// Matches
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/matches", matchH.List)
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches", adminOrDewan(http.HandlerFunc(matchH.Create)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/matches/{id}", matchH.Get)
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/load-arena", matchOperators(http.HandlerFunc(matchH.LoadArena)))

	// Brackets
	protected.Handle("POST /api/v1/competitions/{comp_id}/brackets/generate", adminOrDewan(http.HandlerFunc(bracketH.Generate)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/brackets", bracketH.GetTree)

	// Mount protected routes under AuthMiddleware
	mux.Handle("/api/v1/", Chain(protected, AuthMiddleware))

	// === WebSocket (auth via query param) ===
	mux.HandleFunc("GET /ws/arena/{arena_id}", matchH.HandleWS)

	// Apply CORS ke semua
	return Chain(mux, CORSMiddleware)
}