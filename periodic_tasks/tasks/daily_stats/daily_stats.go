package daily_stats

import (
	"context"
	"fmt"
	"log"

	"vpn-periodic-tasks/internal/appclient"
	"vpn-periodic-tasks/internal/config"
)

// Task implements the scheduler.Task interface for daily statistics
type Task struct {
	client *appclient.Client
}

// New creates a new daily stats task
func New(client *appclient.Client) *Task {
	return &Task{
		client: client,
	}
}

// Name returns the task name
func (t *Task) Name() string {
	return "daily_stats"
}

// Run executes the daily stats task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	result, err := t.client.DailyStats(ctx)
	if err != nil {
		return fmt.Errorf("call daily-stats endpoint: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("daily stats failed: %s", result.Error)
	}

	log.Printf("daily stats completed successfully: %s", result.Message)
	return nil
}
