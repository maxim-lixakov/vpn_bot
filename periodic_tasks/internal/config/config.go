package config

import (
	"fmt"
	"os"
)

// TaskSchedule defines the schedule for a periodic task
type TaskSchedule struct {
	TaskName string // Name of the task package
	Schedule string // Cron format schedule (e.g., "*/1 * * * *" for every minute)
}

type Config struct {
	// App API
	AppAddr          string
	AppInternalToken string
}

// Load loads configuration from environment variables
func Load() (Config, error) {
	var cfg Config

	// App API config
	cfg.AppAddr = getenv("APP_ADDR", "http://app:8080")
	cfg.AppInternalToken = getenv("APP_INTERNAL_TOKEN", "")
	if cfg.AppInternalToken == "" {
		return cfg, fmt.Errorf("APP_INTERNAL_TOKEN is required")
	}

	return cfg, nil
}

// GetTaskSchedules returns the list of task schedules
// This can be extended to load from a config file or database
// Note: Schedule format uses 6 fields (with seconds): "second minute hour day month weekday"
// Example: "0 */1 * * * *" means every minute at second 0
// Example: "0 0 0 * * *" means daily at 00:00:00
func GetTaskSchedules() []TaskSchedule {
	return []TaskSchedule{
		{
			TaskName: "backup",
			Schedule: "0 0 0 * * *", // Daily at 00:00
		},
		{
			TaskName: "revoke_expired_keys",
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
