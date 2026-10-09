package api

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/herros27/digital-scoring/internal/api/handler"
	"github.com/herros27/digital-scoring/internal/auth"
	"github.com/herros27/digital-scoring/internal/cache"
	"github.com/herros27/digital-scoring/internal/hub"
	"github.com/herros27/digital-scoring/internal/service"
	"github.com/herros27/digital-scoring/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter merakit seluruh rute. metricsToken (METRICS_TOKEN) kosong = /metrics hanya dari loopback.
func NewRouter(db *sql.DB, hubManager *hub.Manager, metricsToken string) http.Handler {
	// in-memory cache (Otter S3-FIFO)
	appCache := cache.MustNew(10_000)

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
	bracketService := service.NewBracketService(bracketRepo).WithMatchStore(matchRepo).WithArenaStore(arenaRepo)
	categoryService := service.NewCategoryService(categoryRepo).WithRulesetLookup(func(id string) bool {
		rs, err := rulesetRepo.GetByID(id)
		return err == nil && rs != nil
	})
	athleteService := service.NewAthleteService(athleteRepo).WithCategoryRepo(categoryRepo)
	rulesetService := service.NewRulesetService(rulesetRepo)
	matchService.SetRulesetResolver(rulesetService)
	matchService.SetCategoryRulesetLookup(categoryRepo)
	matchService.SetArenaStore(arenaRepo)
	matchService.SetRuntimeStore(&store.MatchRuntimeStore{DB: db})
	if n, err := matchService.RestoreActiveMatches(); err != nil {
		log.Printf("[match] pemulihan partai gagal: %v", err)
	} else if n > 0 {
		log.Printf("[match] %d partai dipulihkan dalam keadaan dijeda setelah restart", n)
	}
	officialService := service.NewOfficialService(officialRepo)
	contingentService := service.NewContingentService(contingentRepo)
	reportService := service.NewReportService(compRepo, arenaRepo, categoryRepo, athleteRepo, matchRepo)

	// handlers
	authH := &handler.AuthHandler{
		Pins: pinRepo,
		// Tebak-PIN dilindungi tiga lapis (perangkat 5x, IP 25x, global 200x per 15 menit): satu tablet yang salah PIN
		// tidak mengunci tablet lain di IP yang sama (Docker Desktop/proxy), tetapi penebakan tetap terbatas.
		Guard:    auth.NewPinGuard(auth.DefaultPinGuardConfig()),
		ClientIP: ClientIP,
	}
	compH := &handler.CompetitionHandler{Service: compService, Cache: appCache}
	arenaH := &handler.ArenaHandler{Service: arenaService, Hubs: hubManager}
	matchH := &handler.MatchHandler{Service: matchService, HubManager: hubManager, Cache: appCache}
	publicBracketH := &handler.PublicBracketHandler{
		Competitions: compService,
		Categories:   categoryService,
		Brackets:     bracketService,
		Participants: athleteService,
	}
	liveH := &handler.LiveHandler{ScoreStore: scoreRepo, MatchStore: matchRepo, Matches: matchService}
	bracketH := &handler.BracketHandler{Service: bracketService, AthleteService: athleteService, CategoryService: categoryService, Cache: appCache}
	categoryH := &handler.CategoryHandler{Service: categoryService}
	athleteH := &handler.AthleteHandler{Service: athleteService, Cache: appCache, Competition: compService}
	rulesetH := &handler.RulesetHandler{Service: rulesetService}
	officialH := &handler.OfficialHandler{Service: officialService}
	contingentH := &handler.ContingentHandler{Service: contingentService}
	reportH := &handler.ReportHandler{Service: reportService}

	mux := http.NewServeMux()

	// === Swagger Documentation (Public) ===
	mux.HandleFunc("GET /docs", SwaggerHandler())
	mux.HandleFunc("GET /docs/openapi.yaml", OpenAPISpecHandler("docs/openapi.yaml"))

	// === Prometheus Metrics (hanya loopback, atau dengan METRICS_TOKEN) ===
	mux.Handle("GET /metrics", MetricsAccess(metricsToken)(promhttp.Handler()))

	// === Health Check (publik, tanpa detail) ===
	mux.HandleFunc("GET /api/v1/health", HealthHandler(db))

	// Rate limiters:
	// Auth endpoints: 10 req/min, burst 10 per IP (brute force mitigation for PIN & password)
	authLimiter := RateLimit(rate.Every(6*time.Second), 10)
	// Login PIN: batas laju hanya untuk banjir permintaan (1/dtk, burst 60 per IP); perlindungan tebak-PIN yang
	// sebenarnya ada di AuthHandler.Guard dan hanya menghitung PIN SALAH. Batas 10/menit sebelumnya menghitung login
	// berhasil juga, sehingga banyak tablet di satu IP saling memblokir saat mulai bersamaan.
	pinLimiter := RateLimit(rate.Every(time.Second), 60)
	// Global: 60 req/sec, burst 100 per IP (DoS/flood protection)
	globalLimiter := RateLimit(rate.Limit(60), 100)

	// === Public endpoints (no auth) ===
	mux.Handle("POST /api/v1/auth/judge-login", pinLimiter(http.HandlerFunc(authH.JudgeLogin)))
	mux.Handle("POST /api/v1/auth/pin", pinLimiter(http.HandlerFunc(authH.JudgeLogin)))
	mux.Handle("POST /api/v1/auth/login", authLimiter(http.HandlerFunc(officialH.Login)))
	mux.Handle("POST /api/v1/auth/contingent/register", authLimiter(http.HandlerFunc(contingentH.Register)))
	mux.Handle("POST /api/v1/auth/contingent/login", authLimiter(http.HandlerFunc(contingentH.Login)))
	mux.HandleFunc("POST /api/v1/auth/logout", authH.Logout)
	mux.HandleFunc("GET /api/v1/live/arenas", liveH.Arenas)
	mux.HandleFunc("GET /api/v1/live/schedule", liveH.Schedule)
	mux.HandleFunc("GET /api/v1/live/scoreboard/{arena_id}", liveH.Scoreboard)

	// === Bagan publik (penonton, tanpa login) ===
	// Hanya kejuaraan berstatus active, hanya nama+kontingen atlet. Respons identik untuk semua pemanggil,
	// jadi aman di-cache; kunci memuat comp_id sehingga ikut dibersihkan InvalidateCompetition.
	mux.HandleFunc("GET /api/v1/public/competitions", appCache.Handler(10*time.Second, publicBracketH.ListCompetitions))
	mux.HandleFunc("GET /api/v1/public/competitions/{comp_id}/brackets", appCache.Handler(10*time.Second, publicBracketH.ListBrackets))
	mux.HandleFunc("GET /api/v1/public/competitions/{comp_id}/brackets/{category_id}", appCache.Handler(10*time.Second, publicBracketH.GetBracket))

	// === Public cached tournament views (Otter in-memory cache) ===
	// ponytail: 10s TTL absorbs high-concurrency bursts from spectators & contingents during tournaments.
	// Auto-invalidated on match finish & bracket generate, fallback to 10s TTL.
	mux.HandleFunc("GET /api/v1/competitions/{comp_id}/report/summary", appCache.Handler(10*time.Second, reportH.Summary))
	mux.HandleFunc("GET /api/v1/competitions/{comp_id}/report/medals", appCache.Handler(10*time.Second, reportH.Medals))
	mux.HandleFunc("GET /api/v1/competitions/{comp_id}/report/export", appCache.Handler(10*time.Second, reportH.Export))
	mux.HandleFunc("GET /api/v1/competitions/{comp_id}/brackets", appCache.Handler(10*time.Second, bracketH.GetTree))

	// === Protected endpoints (JWT required) ===
	protected := http.NewServeMux()

	// RBAC roles
	adminOrDewan := RequireRole("admin", "dewan")
	matchOperators := RequireRole("admin", "dewan", "wasit")
	canRegisterAthlete := RequireRole("admin", "dewan", "contingent")
	athleteViewers := RequireRole("admin", "dewan", "operator", "contingent")

	// Auth & PIN (Admin only)
	protected.Handle("GET /api/v1/auth/pins", adminOrDewan(http.HandlerFunc(authH.ListPINs)))
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
	protected.Handle("DELETE /api/v1/competitions/{comp_id}/categories", adminOrDewan(http.HandlerFunc(categoryH.Reset)))
	protected.Handle("PUT /api/v1/competitions/{comp_id}/categories/ruleset", adminOrDewan(http.HandlerFunc(categoryH.AssignRulesets)))

	// Athletes & Participants
	// Data atlet memuat tanggal lahir & NIK, jadi TIDAK publik dan TIDAK lewat cache bersama (kunci cache hanya
	// method+URL, sedangkan hasilnya berbeda per peran: kontingen hanya melihat atletnya sendiri).
	protected.Handle("GET /api/v1/athletes", adminOrDewan(http.HandlerFunc(athleteH.List)))
	protected.Handle("GET /api/v1/competitions/{comp_id}/athletes", athleteViewers(http.HandlerFunc(athleteH.ListByCompetition)))
	protected.Handle("POST /api/v1/athletes", canRegisterAthlete(http.HandlerFunc(athleteH.Create)))
	protected.HandleFunc("GET /api/v1/athletes/{id}", athleteH.Get)
	protected.Handle("PUT /api/v1/athletes/{id}", canRegisterAthlete(http.HandlerFunc(athleteH.UpdateProfile)))
	protected.Handle("POST /api/v1/competitions/{comp_id}/athletes", canRegisterAthlete(http.HandlerFunc(athleteH.RegisterToCompetition)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/participants", athleteH.ListParticipants)
	protected.Handle("PUT /api/v1/competitions/{comp_id}/participants/{id}/verify", adminOrDewan(http.HandlerFunc(athleteH.VerifyParticipant)))

	// Arenas
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/arenas", arenaH.List)
	protected.Handle("POST /api/v1/competitions/{comp_id}/arenas", adminOrDewan(http.HandlerFunc(arenaH.Create)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/arenas/{id}", arenaH.Get)
	protected.Handle("GET /api/v1/competitions/{comp_id}/arenas/{id}/presence", adminOrDewan(http.HandlerFunc(arenaH.Presence)))
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
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/outcome", matchOperators(http.HandlerFunc(matchH.Outcome)))
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/cancel", matchOperators(http.HandlerFunc(matchH.Cancel)))
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/reschedule", matchOperators(http.HandlerFunc(matchH.Reschedule)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/matches/{id}/scores", matchH.GetScores)
	// Skor REST hanya untuk juri (PIN); identitas diambil dari token (lihat MatchHandler.SubmitScore).
	protected.Handle("POST /api/v1/competitions/{comp_id}/matches/{id}/scores", RequireRole("judge")(http.HandlerFunc(matchH.SubmitScore)))
	protected.HandleFunc("GET /api/v1/competitions/{comp_id}/matches/{id}/events", matchH.GetEvents)

	// Rulesets
	protected.HandleFunc("GET /api/v1/rulesets", rulesetH.List)
	protected.Handle("POST /api/v1/rulesets", adminOrDewan(http.HandlerFunc(rulesetH.Create)))
	protected.HandleFunc("GET /api/v1/rulesets/{id}", rulesetH.Get)
	protected.Handle("PUT /api/v1/rulesets/{id}", adminOrDewan(http.HandlerFunc(rulesetH.Update)))
	protected.Handle("DELETE /api/v1/rulesets/{id}", adminOrDewan(http.HandlerFunc(rulesetH.Delete)))

	// Brackets
	protected.Handle("POST /api/v1/competitions/{comp_id}/brackets/generate", adminOrDewan(http.HandlerFunc(bracketH.Generate)))

	// Officials / Operator Management (Admin & Dewan)
	protected.Handle("GET /api/v1/officials", adminOrDewan(http.HandlerFunc(officialH.List)))
	protected.Handle("POST /api/v1/officials", adminOrDewan(http.HandlerFunc(officialH.Create)))
	protected.Handle("GET /api/v1/officials/{id}", adminOrDewan(http.HandlerFunc(officialH.Get)))
	protected.Handle("PUT /api/v1/officials/{id}", adminOrDewan(http.HandlerFunc(officialH.Update)))
	protected.Handle("DELETE /api/v1/officials/{id}", adminOrDewan(http.HandlerFunc(officialH.Delete)))

	// Contingents Management
	protected.Handle("GET /api/v1/contingents", adminOrDewan(http.HandlerFunc(contingentH.List)))
	protected.HandleFunc("GET /api/v1/contingents/{id}", contingentH.Get)

	// Mount protected routes under AuthMiddleware
	mux.Handle("/api/v1/", Chain(protected, AuthMiddleware))

	// === WebSocket (auth via query param) ===
	mux.HandleFunc("GET /ws/arena/{arena_id}", matchH.HandleWS)

	// Apply CORS, Metrics & Global Rate Limiting ke semua
	return Chain(mux, CORSMiddleware, MetricsMiddleware, globalLimiter)
}
