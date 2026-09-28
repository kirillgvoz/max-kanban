package config

import (
	"testing"
)

func TestValidateProductionSecrets(t *testing.T) {
	cfg := &Config{
		Environment:   "production",
		DatabaseURL:   "postgres://kanban:kanban@localhost:5432/kanban",
		WebhookSecret: "taskflow_secret_change_me",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted default production secrets")
	}

	cfg.MaxBotToken = "token"
	cfg.WebhookSecret = "a-long-random-secret-value"
	cfg.DatabaseURL = "postgres://app:strong-password@db:5432/kanban?sslmode=require"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestParseOrigins(t *testing.T) {
	origins := parseOrigins(" https://example.com,,http://localhost:3002 ")
	if len(origins) != 2 || origins[0] != "https://example.com" || origins[1] != "http://localhost:3002" {
		t.Fatalf("origins = %#v", origins)
	}
}
