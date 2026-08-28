package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address         string
	DatabasePath    string
	ShutdownTimeout time.Duration
	DefaultLine     string
	LogFormat       string
	MaxBodyBytes    int64
}

func Load() Config {
	return Config{Address: env("ALERT_ADDR", ":8080"), DatabasePath: env("ALERT_DB", "./data/alerts.db"), ShutdownTimeout: durationEnv("ALERT_SHUTDOWN", "5s"), DefaultLine: env("ALERT_DEFAULT_LINE", "production-43"), LogFormat: env("ALERT_LOG_FORMAT", "text"), MaxBodyBytes: int64Env("ALERT_MAX_BODY", 1048576)}
}
func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
func durationEnv(key, fallback string) time.Duration {
	value := env(key, fallback)
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		parsed, _ = time.ParseDuration(fallback)
	}
	return parsed
}
func int64Env(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
func (c Config) Validate() error {
	if c.Address == "" || c.DatabasePath == "" || c.ShutdownTimeout <= 0 || c.MaxBodyBytes <= 0 {
		return strconv.ErrSyntax
	}
	return nil
}
