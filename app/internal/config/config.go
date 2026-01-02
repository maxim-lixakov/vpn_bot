package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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

	BotToken string

	BackupAdminTgUserID int64

	PaymentsProviderToken     string
	PaymentsCurrency          string
	PaymentsVPNPriceMinor     int64
	PaymentsVPNTitle          string
	PaymentsVPNDescription    string
	PaymentsVPNPayload        string
	PaymentsVPNRenewalPayload string
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

	cfg.BotToken = getenv("BOT_TOKEN", "token")

	// Backup admin Telegram user ID
	if tgUserIDStr := os.Getenv("BACKUP_ADMIN_TG_USER_ID"); tgUserIDStr != "" {
		if tgUserID, err := strconv.ParseInt(tgUserIDStr, 10, 64); err == nil {
			cfg.BackupAdminTgUserID = tgUserID
		}
	}

	// Payments config
	cfg.PaymentsProviderToken = getenv("PAYMENTS_PROVIDER_TOKEN", "")
	cfg.PaymentsCurrency = getenv("PAYMENTS_CURRENCY", "RUB")
	cfg.PaymentsVPNPriceMinor, _ = strconv.ParseInt(getenv("PAYMENTS_VPN_PRICE_MINOR", "15000"), 10, 64)
	cfg.PaymentsVPNTitle = getenv("PAYMENTS_VPN_TITLE", "VPN подписка на 1 месяц")
	cfg.PaymentsVPNDescription = getenv("PAYMENTS_VPN_DESCRIPTION", "VPN подписка на 1 месяц")
	cfg.PaymentsVPNPayload = getenv("PAYMENTS_VPN_PAYLOAD", "vpn_sub_v1")
	cfg.PaymentsVPNRenewalPayload = getenv("PAYMENTS_VPN_RENEWAL_PAYLOAD", "vpn_renewal_v1")

	return cfg, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
