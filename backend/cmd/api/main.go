package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/greg/darts-league/backend/internal/config"
	"github.com/greg/darts-league/backend/internal/httpapi"
	"github.com/greg/darts-league/backend/internal/league"
	pgstore "github.com/greg/darts-league/backend/internal/store/postgres"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	now := cfg.NowFunc()

	mux := http.NewServeMux()
	store := buildStore(ctx, cfg)
	if _, err := store.EnsureActiveSeason(ctx, league.NewSeason("MVP Season")); err != nil {
		log.Fatal(err)
	}
	authHandler := httpapi.NewAuthHandler(cfg.AdminUser, cfg.AdminPass, cfg.AdminSessionSecret)
	registrationHandler := httpapi.NewRegistrationHandler(league.NewRegistrationServiceWithNow(store, now))
	seasonHandler := httpapi.NewSeasonHandler(league.NewSeasonServiceWithNow(store, now), league.NewFixtureServiceWithNow(store, now))
	resultHandler := httpapi.NewResultHandler(league.NewResultServiceWithNow(store, now))
	authHandler.RegisterRoutes(mux)
	registrationHandler.RegisterRoutes(mux, authHandler.RequireAdmin)
	seasonHandler.RegisterRoutes(mux, authHandler.RequireAdmin)
	resultHandler.RegisterRoutes(mux, authHandler.RequireAdmin)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("darts-league backend listening on %s", cfg.HTTPAddress)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func buildStore(ctx context.Context, cfg config.Config) league.Store {
	if cfg.DatabaseURL != "" {
		store, err := pgstore.Open(ctx, cfg.DatabaseURL)
		if err == nil {
			log.Printf("using postgres store")
			return store
		}
		log.Printf("postgres unavailable, falling back to in-memory store: %v", err)
	}

	log.Printf("using in-memory store")
	return league.NewMemoryStore()
}
