package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultAddress   = ":8080"
	defaultGCPercent = 100
)

// Config содержит настройки HTTP-сервера и сборщика мусора.
type Config struct {
	Address   string
	GCPercent int
}

// Load читает настройки приложения из переменных окружения.
func Load() (Config, error) {
	cfg := Config{
		Address:   getenv("ADDR", defaultAddress),
		GCPercent: defaultGCPercent,
	}

	if raw := os.Getenv("GC_PERCENT"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse GC_PERCENT: %w", err)
		}

		cfg.GCPercent = value
	}

	return cfg, nil
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
