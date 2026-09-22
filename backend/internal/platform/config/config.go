package config

import (
	"errors"
	"os"
)

type Config struct {
	Environment   string
	HTTPAddress   string
	DatabaseURL   string
	SessionPepper string
	Currency      string
	Timezone      string
	TrustedOrigin string
}

func Load() (Config, error) {
	c := Config{
		Environment:   value("APP_ENV", "development"),
		HTTPAddress:   value("HTTP_ADDRESS", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		SessionPepper: os.Getenv("SESSION_PEPPER"),
		Currency:      value("DEFAULT_CURRENCY", "IDR"),
		Timezone:      value("DEFAULT_TIMEZONE", "Asia/Jakarta"),
		TrustedOrigin: value("TRUSTED_ORIGIN", "http://localhost:3000"),
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(c.SessionPepper) < 32 {
		return Config{}, errors.New("SESSION_PEPPER must contain at least 32 bytes")
	}
	return c, nil
}

func value(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
