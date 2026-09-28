package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Environment           string
	Version               string
	Revision              string
	Port                  string
	DatabaseURL           string
	MaxBotToken           string
	WebhookSecret         string
	FrontendURL           string
	BotName               string
	MaxWebhookURL         string
	MaxWebhookUpdateTypes []string
	MaxCAFile             string
	AllowedOrigins        []string
}

func Load() *Config {
	return &Config{
		Environment:           getEnv("APP_ENV", "development"),
		Version:               getEnv("APP_VERSION", "dev"),
		Revision:              getEnv("GIT_SHA", "unknown"),
		Port:                  getEnv("PORT", "9300"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://kanban:kanban@localhost:5432/kanban?sslmode=disable"),
		MaxBotToken:           os.Getenv("MAX_BOT_TOKEN"),
		WebhookSecret:         getEnv("WEBHOOK_SECRET", "taskflow_secret_change_me"),
		FrontendURL:           getEnv("FRONTEND_URL", "https://kirillgvoz.ru/max-kanban/"),
		BotName:               getEnv("BOT_NAME", "se14445725_bot"),
		MaxWebhookURL:         getEnv("MAX_WEBHOOK_URL", ""),
		MaxWebhookUpdateTypes: parseList(getEnv("MAX_WEBHOOK_UPDATE_TYPES", "message_created,message_callback,bot_started,bot_added")),
		MaxCAFile:             getEnv("MAX_CA_FILE", ""),
		AllowedOrigins:        parseOrigins(getEnv("ALLOWED_ORIGINS", "https://kirillgvoz.ru,https://www.kirillgvoz.ru,http://localhost:3002,http://127.0.0.1:3002,http://localhost:9300,http://127.0.0.1:9300")),
	}
}

func (c *Config) Validate() error {
	if c.Environment != "production" {
		return nil
	}
	if strings.TrimSpace(c.MaxBotToken) == "" {
		return fmt.Errorf("MAX_BOT_TOKEN is required in production")
	}
	secret := strings.TrimSpace(c.WebhookSecret)
	if secret == "" || secret == "taskflow_secret_change_me" {
		return fmt.Errorf("WEBHOOK_SECRET must be changed in production")
	}
	if strings.Contains(c.DatabaseURL, "kanban:kanban@") {
		return fmt.Errorf("DATABASE_URL must not use development credentials in production")
	}
	if strings.TrimSpace(c.MaxWebhookURL) != "" && !strings.HasPrefix(strings.TrimSpace(c.MaxWebhookURL), "https://") {
		return fmt.Errorf("MAX_WEBHOOK_URL must use HTTPS in production")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func parseOrigins(raw string) []string {
	origins := make([]string, 0)
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func parseList(raw string) []string {
	return parseOrigins(raw)
}
