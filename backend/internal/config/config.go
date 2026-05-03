package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	HTTPAddr           string
	DatabaseURL        string
	DataDir            string
	PublicBaseURL      string
	WidgetFetchTimeout time.Duration
	WidgetCacheTTL     time.Duration
	MaxIconUploadBytes int64
	AllowedOrigins     []string
	MigrationsDir      string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:             env("APP_ENV", "development"),
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		DatabaseURL:        env("DATABASE_URL", ""),
		DataDir:            env("DATA_DIR", "./data"),
		PublicBaseURL:      env("PUBLIC_BASE_URL", "http://localhost:8080"),
		WidgetFetchTimeout: 5 * time.Second,
		WidgetCacheTTL:     60 * time.Second,
		MaxIconUploadBytes: 524288,
		AllowedOrigins:     splitCSV(env("ALLOWED_ORIGINS", "http://localhost:3000")),
		MigrationsDir:      env("MIGRATIONS_DIR", "../db/migrations"),
	}

	var err error
	if raw := os.Getenv("WIDGET_FETCH_TIMEOUT"); raw != "" {
		cfg.WidgetFetchTimeout, err = time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("WIDGET_FETCH_TIMEOUT: %w", err)
		}
	}
	if raw := os.Getenv("WIDGET_CACHE_TTL"); raw != "" {
		cfg.WidgetCacheTTL, err = time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("WIDGET_CACHE_TTL: %w", err)
		}
	}
	if raw := os.Getenv("MAX_ICON_UPLOAD_BYTES"); raw != "" {
		cfg.MaxIconUploadBytes, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("MAX_ICON_UPLOAD_BYTES: %w", err)
		}
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.MaxIconUploadBytes <= 0 {
		return Config{}, errors.New("MAX_ICON_UPLOAD_BYTES must be positive")
	}
	if cfg.WidgetFetchTimeout <= 0 {
		return Config{}, errors.New("WIDGET_FETCH_TIMEOUT must be positive")
	}
	if cfg.WidgetCacheTTL <= 0 {
		return Config{}, errors.New("WIDGET_CACHE_TTL must be positive")
	}
	if _, err := url.ParseRequestURI(cfg.PublicBaseURL); err != nil {
		return Config{}, fmt.Errorf("PUBLIC_BASE_URL must be absolute URL: %w", err)
	}
	return cfg, nil
}

func (c Config) IconDir() string {
	return strings.TrimRight(c.DataDir, "/") + "/icons"
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
