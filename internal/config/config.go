// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds every environment-driven setting the application needs.
type Config struct {
	AppEnv string `env:"APP_ENV" envDefault:"development"`
	Port   int    `env:"PORT" envDefault:"8080"`

	DatabaseDSN        string        `env:"DATABASE_DSN,required"`
	DatabaseMaxOpen    int           `env:"DATABASE_MAX_OPEN_CONNS" envDefault:"25"`
	DatabaseMaxIdle    int           `env:"DATABASE_MAX_IDLE_CONNS" envDefault:"25"`
	DatabaseConnMaxAge time.Duration `env:"DATABASE_CONN_MAX_LIFETIME" envDefault:"5m"`

	JWTSecret       string        `env:"JWT_SECRET,required"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`
	TrustProxy         bool     `env:"TRUST_PROXY" envDefault:"false"`
	LogLevel           string   `env:"LOG_LEVEL" envDefault:"info"`
}

// IsProduction reports whether the app is running with APP_ENV=production.
func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load parses the environment into a Config and validates invariants that
// the env tags alone cannot express, such as the minimum JWT secret length.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 bytes long, got %d", len(cfg.JWTSecret))
	}

	return cfg, nil
}
