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
    compRepo  := &store.CompetitionStore{DB: db}
    arenaRepo := &store.ArenaStore{DB: db}
    matchRepo := &store.MatchStore{DB: db}
    scoreRepo := &store.MatchScoreStore{DB: db}
    pinRepo  := &auth.PinStore{DB: db}
    bracketRepo := &store.BracketStore{DB: db}

    // services
    compService := service.NewCompetitionService(compRepo)
    arenaService := service.NewArenaService(arenaRepo, compRepo)
    matchService := service.NewMatchService(matchRepo, compRepo, hubManager, &store.MatchEventStore{DB: db}, scoreRepo, matchRepo)
    bracketService := service.NewBracketService(bracketRepo)

    // handlers
    authH  := &handler.AuthHandler{Pins: pinRepo}
    compH  := &handler.CompetitionHandler{Service: compService}
    arenaH := &handler.ArenaHandler{Service: arenaService}
    matchH := &handler.MatchHandler{Service: matchService, HubManager: hubManager}
    liveH  := &handler.LiveHandler{ScoreStore: scoreRepo, MatchStore: matchRepo}
    bracketH := &handler.BracketHandler{Service: bracketService}

    mux := http.NewServeMux()

    // === Swagger Documentation (Public) ===
    mux.HandleFunc("GET /docs", SwaggerHandler())
    mux.HandleFunc("GET /docs/openapi.yaml", OpenAPISpecHandler("docs/openapi.yaml"))

    // === Public endpoints (no auth) ===
    mux.HandleFunc("POST /api/v1/auth/judge-login", authH.JudgeLogin)
    mux.HandleFunc("GET /api/v1/live/scoreboard/{arena_id}", liveH.Scoreboard)

    // === Protected endpoints (JWT required) ===
    protected := http.NewServeMux()
    mux.HandleFunc("POST /api/v1/auth/create-pin",                          authH.CreatePIN)
    mux.HandleFunc("GET /api/v1/competitions",                               compH.List)
    mux.HandleFunc("POST /api/v1/competitions",                              compH.Create)
    mux.HandleFunc("GET /api/v1/competitions/{id}",                          compH.Get)
    mux.HandleFunc("PUT /api/v1/competitions/{id}/activate",                 compH.Activate)
    
    mux.HandleFunc("GET /api/v1/competitions/{comp_id}/arenas",              arenaH.List)
    mux.HandleFunc("POST /api/v1/competitions/{comp_id}/arenas",             arenaH.Create)
    
    mux.HandleFunc("GET /api/v1/competitions/{comp_id}/matches",             matchH.List)
    mux.HandleFunc("POST /api/v1/competitions/{comp_id}/matches",            matchH.Create)
    mux.HandleFunc("GET /api/v1/competitions/{comp_id}/matches/{id}",        matchH.Get)
    mux.HandleFunc("POST /api/v1/competitions/{comp_id}/matches/{id}/load-arena", matchH.LoadArena)

    mux.HandleFunc("POST /api/v1/competitions/{comp_id}/brackets/generate", bracketH.Generate)
    mux.HandleFunc("GET /api/v1/competitions/{comp_id}/brackets", bracketH.GetTree)

    mux.Handle("/api/v1/", Chain(protected, AuthMiddleware))

    // === WebSocket (auth via query param) ===
    mux.HandleFunc("GET /ws/arena/{arena_id}", matchH.HandleWS)

    // Apply CORS ke semua
    return Chain(mux, CORSMiddleware)
}