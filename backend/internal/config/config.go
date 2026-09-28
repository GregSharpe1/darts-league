package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPAddress         string
	DatabaseURL         string
	InstanceName        string
	PublicBaseURL       string
	Version             string
	Timezone            string
	AdminUser           string
	AdminPass           string
	AdminSessionSecret  string
	SlackBotToken       string
	SlackPublicChannel  string
	SlackAdminChannel   string
	NowOverride         string
	ResultsEndpoint     string
	ResultsPollInterval time.Duration
}

func Load() Config {
	return Config{
		HTTPAddress:         envOrDefault("HTTP_ADDRESS", ":8080"),
		DatabaseURL:         envOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/darts_league?sslmode=disable"),
		InstanceName:        envOrDefault("INSTANCE_NAME", "Darts League"),
		PublicBaseURL:       os.Getenv("PUBLIC_BASE_URL"),
		Version:             envOrDefault("APP_VERSION", "dev"),
		Timezone:            envOrDefault("APP_TIMEZONE", "Europe/London"),
		NowOverride:         os.Getenv("APP_NOW"),
		ResultsEndpoint:     os.Getenv("RESULTS_ENDPOINT"),
		AdminUser:           envOrDefault("ADMIN_USERNAME", "admin"),
		AdminPass:           envOrDefault("ADMIN_PASSWORD", "change-me"),
		AdminSessionSecret:  envOrDefault("ADMIN_SESSION_SECRET", "dev-admin-session-secret"),
		SlackBotToken:       os.Getenv("SLACK_BOT_TOKEN"),
		SlackPublicChannel:  os.Getenv("SLACK_PUBLIC_CHANNEL_ID"),
		SlackAdminChannel:   os.Getenv("SLACK_ADMIN_CHANNEL_ID"),
		ResultsPollInterval: durationOrDefault("RESULTS_POLL_INTERVAL", 15*time.Minute),
	}
}

func (c Config) NowFunc() func() time.Time {
	if c.NowOverride == "" {
		return time.Now
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 -0700",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, c.NowOverride)
		if err == nil {
			return func() time.Time {
				return parsed
			}
		}
	}

	return time.Now
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
