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
	"github.com/greg/darts-league/backend/internal/resultsrelay"
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
	seasonHandler := httpapi.NewSeasonHandler(league.NewSeasonServiceWithNow(store, now), league.NewFixtureServiceWithNow(store, now), cfg.InstanceName)
	resultService := league.NewResultServiceWithNow(store, now)
	resultHandler := httpapi.NewResultHandler(resultService)
	pendingResultHandler := httpapi.NewPendingResultHandler(league.NewPendingResultServiceWithNow(store, resultService, now))
	versionHandler := httpapi.NewVersionHandler(cfg.Version)
	authHandler.RegisterRoutes(mux)
	registrationHandler.RegisterRoutes(mux, authHandler.RequireAdmin)
	seasonHandler.RegisterRoutes(mux, authHandler.RequireAdmin)
	resultHandler.RegisterRoutes(mux, authHandler.RequireAdmin)
	pendingResultHandler.RegisterRoutes(mux, authHandler.RequireAdmin)
	versionHandler.RegisterRoutes(mux)
	httpapi.NewMatchDetailHandler(league.NewFixtureServiceWithNow(store, now)).RegisterRoutes(mux, authHandler.RequireAdmin)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	stopPoller, pollNow := startResultPoller(cfg, store, resultService, now)
	defer stopPoller()
	mux.HandleFunc("POST /api/admin/results/poll", authHandler.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if pollNow == nil {
			http.Error(w, "Results relay polling is not configured.", http.StatusServiceUnavailable)
			return
		}
		if err := pollNow(r.Context()); err != nil {
			http.Error(w, "Results relay poll failed.", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

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
	weeklyService := notifications.NewWeeklyService(store, now, client, cfg.SlackPublicChannel, cfg.PublicBaseURL)

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

func startResultPoller(cfg config.Config, store league.Store, resultService league.ResultService, now func() time.Time) (func(), func(context.Context) error) {
	if cfg.ResultsEndpoint == "" {
		return func() {}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	client, err := resultsrelay.NewClient(cfg.ResultsEndpoint)
	if err != nil {
		log.Printf("results relay: disabled, failed to build client: %v", err)
		cancel()
		return func() {}, nil
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		loc = time.UTC
	}

	pendingResults := league.NewPendingResultServiceWithNow(store, resultService, now).WithNotifier(
		notifications.NewPendingResultNotifier(slack.NewClient(cfg.SlackBotToken), cfg.SlackAdminChannel, cfg.PublicBaseURL),
	)
	poller := resultsrelay.NewPoller(client, pendingResults, loc, cfg.ResultsPollInterval, log.Default())
	if _, ok := store.(*pgstore.Store); ok {
		poller = poller.WithDurableIngest(func(ctx context.Context, message resultsrelay.Message) error {
			_, err := pendingResults.IngestDurablePayload(ctx, []byte(message.Body))
			return err
		})
	}
	go poller.Run(ctx)

	return cancel, poller.PollNow
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
