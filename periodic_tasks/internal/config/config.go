package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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
	// Database
	PG Postgres

	// Telegram
	BotToken            string
	BackupAdminTgUserID int64 // Telegram user ID to send database backups to

	// Outline servers
	Servers map[string]OutlineServer

	// App API
	AppAddr          string
	AppInternalToken string
}

// TaskSchedule defines the schedule for a periodic task
type TaskSchedule struct {
	TaskName string // Name of the task package
	Schedule string // Cron format schedule (e.g., "*/1 * * * *" for every minute)
}

// Load loads configuration from environment variables
func Load() (Config, error) {
	var cfg Config

	// Database config
	cfg.PG = Postgres{
		Host:     getenv("POSTGRES_HOST", "localhost"),
		Port:     getenv("POSTGRES_PORT", "5432"),
		DB:       getenv("POSTGRES_DB", "vpn"),
		User:     getenv("POSTGRES_USER", "vpn"),
		Password: getenv("POSTGRES_PASSWORD", "vpn"),
		SSLMode:  getenv("POSTGRES_SSLMODE", "disable"),
	}

	// Telegram bot token
	cfg.BotToken = getenv("BOT_TOKEN", "")

	// Backup admin Telegram user ID
	if tgUserIDStr := os.Getenv("BACKUP_ADMIN_TG_USER_ID"); tgUserIDStr != "" {
		var tgUserID int64
		if _, err := fmt.Sscanf(tgUserIDStr, "%d", &tgUserID); err == nil {
			cfg.BackupAdminTgUserID = tgUserID
		}
	}

	// Outline servers
	raw := os.Getenv("OUTLINE_SERVERS_JSON")
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg.Servers); err != nil {
			return cfg, fmt.Errorf("failed to parse OUTLINE_SERVERS_JSON: %w", err)
		}
	}

	// App API config
	cfg.AppAddr = getenv("APP_ADDR", "http://app:8080")
	cfg.AppInternalToken = getenv("APP_INTERNAL_TOKEN", "")

	return cfg, nil
}

// OpenDB opens a database connection
func (c *Config) OpenDB() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.PG.User, c.PG.Password, c.PG.Host, c.PG.Port, c.PG.DB, c.PG.SSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

// GetTaskSchedules returns the list of task schedules
// This can be extended to load from a config file or database
// Note: Schedule format uses 6 fields (with seconds): "second minute hour day month weekday"
// Example: "0 */1 * * * *" means every minute at second 0
// Example: "0 0 0 * * *" means daily at 00:00:00
func GetTaskSchedules() []TaskSchedule {
	return []TaskSchedule{
		{
			TaskName: "example",
			Schedule: "0 0 * * * *", // Every hour at minute 0, second 0
		},
		{
			TaskName: "backup",
			Schedule: "0 0 0 * * *", // Daily at 00:00
		},
		// Add more tasks here as they are created
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
