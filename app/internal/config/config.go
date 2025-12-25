package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type OutlineServer struct {
	Name        string `json:"name"`
	APIURL      string `json:"api_url"`
	TLSInsecure bool   `json:"tls_insecure"`
}

type Postgres struct {
	Host     string
	Port     string
	DB       string
	User     string
	Password string
	SSLMode  string
}

type Config struct {
	Addr          string
	InternalToken string
	Servers       map[string]OutlineServer

	PG Postgres

	// PromocodesOnlyForNewUsers - если true, промокоды работают только для пользователей без активных подписок
	PromocodesOnlyForNewUsers bool
}

func Load() (Config, error) {
	var cfg Config

	cfg.Addr = getenv("APP_ADDR", ":8080")
	cfg.InternalToken = os.Getenv("APP_INTERNAL_TOKEN")
	if cfg.InternalToken == "" {
		return cfg, fmt.Errorf("APP_INTERNAL_TOKEN is required")
	}

	raw := os.Getenv("OUTLINE_SERVERS_JSON")
	if raw == "" {
		return cfg, fmt.Errorf("OUTLINE_SERVERS_JSON is required")
	}
	if err := json.Unmarshal([]byte(raw), &cfg.Servers); err != nil {
		return cfg, fmt.Errorf("failed to parse OUTLINE_SERVERS_JSON: %w", err)
	}

	cfg.PG = Postgres{
		Host:     getenv("POSTGRES_HOST", "localhost"),
		Port:     getenv("POSTGRES_PORT", "5432"),
		DB:       getenv("POSTGRES_DB", "vpn"),
		User:     getenv("POSTGRES_USER", "vpn"),
		Password: getenv("POSTGRES_PASSWORD", "vpn"),
		SSLMode:  getenv("POSTGRES_SSLMODE", "disable"),
	}

	cfg.PromocodesOnlyForNewUsers = getenv("PROMOCODES_ONLY_FOR_NEW_USERS", "true") == "true"

	return cfg, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
