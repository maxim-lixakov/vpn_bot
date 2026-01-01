package backup

import (
	"context"
	"fmt"
	"log"

	"vpn-periodic-tasks/internal/appclient"
	"vpn-periodic-tasks/internal/config"
)

// Task implements the scheduler.Task interface for database backups
type Task struct {
	client *appclient.Client
}

// New creates a new backup task
func New(client *appclient.Client) *Task {
	return &Task{
		client: client,
	}
}

// Name returns the task name
func (t *Task) Name() string {
	return "backup"
}

// Run executes the backup task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	result, err := t.client.Backup(ctx)
	if err != nil {
		return fmt.Errorf("call backup endpoint: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("backup failed: %s", result.Error)
	}

	log.Printf("backup completed successfully: %s", result.Message)
	return nil
}
