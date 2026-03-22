package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/greg/darts-league/backend/internal/config"
	"github.com/greg/darts-league/backend/internal/httpapi"
	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/notifications"
	"github.com/greg/darts-league/backend/internal/slack"
	pgstore "github.com/greg/darts-league/backend/internal/store/postgres"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	now := cfg.NowFunc()
	store := buildStore(ctx, cfg)
	defer closeStore(store)

	if len(os.Args) > 1 && os.Args[1] == "notify" {
		runNotificationCommand(ctx, cfg, store, now, os.Args[2:])
		return
	}

	mux := http.NewServeMux()
	if _, err := store.EnsureActiveSeason(ctx, league.NewSeason("MVP Season")); err != nil {
		log.Fatal(err)
	}
	registrationNotifier := buildRegistrationNotifier(cfg)
	authHandler := httpapi.NewAuthHandler(cfg.AdminUser, cfg.AdminPass, cfg.AdminSessionSecret)
	registrationHandler := httpapi.NewRegistrationHandler(league.NewRegistrationServiceWithNowAndNotifier(store, now, registrationNotifier))
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

func runNotificationCommand(ctx context.Context, cfg config.Config, store league.Store, now func() time.Time, args []string) {
	if _, err := store.EnsureActiveSeason(ctx, league.NewSeason("MVP Season")); err != nil {
		log.Fatal(err)
	}

	client := slack.NewClient(cfg.SlackBotToken)
	weeklyService := notifications.NewWeeklyService(store, now, client, cfg.SlackPublicChannel)

	var (
		posted bool
		err    error
	)

	switch notifications.WeeklyCommandName(args) {
	case "weekly-fixtures":
		posted, err = weeklyService.PostWeeklyFixtures(ctx)
	case "weekly-summary":
		posted, err = weeklyService.PostWeeklySummary(ctx)
	default:
		log.Fatalf("unknown notification command %q", notifications.WeeklyCommandName(args))
	}

	if err != nil {
		log.Fatal(err)
	}
	if !posted {
		log.Printf("notification command completed without sending a message")
		return
	}

	log.Printf("notification command completed successfully")
}

func buildRegistrationNotifier(cfg config.Config) league.RegistrationNotifier {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		loc = time.UTC
	}

	return notifications.NewRegistrationNotifier(slack.NewClient(cfg.SlackBotToken), cfg.SlackAdminChannel, loc, log.Default())
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

func closeStore(store league.Store) {
	closer, ok := store.(interface{ Close() })
	if ok {
		closer.Close()
	}
}
