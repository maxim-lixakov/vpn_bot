package send_logs

import (
	"context"
	"fmt"
	"log"

	"vpn-periodic-tasks/internal/appclient"
	"vpn-periodic-tasks/internal/config"
)

// Task implements the scheduler.Task interface for sending logs
type Task struct {
	appClient *appclient.Client
}

// New creates a new send logs task
func New(appClient *appclient.Client) *Task {
	return &Task{appClient: appClient}
}

// Name returns the task name
func (t *Task) Name() string {
	return "send_logs"
}

// Run executes the task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	log.Println("starting send logs task...")

	result, err := t.appClient.SendLogs(ctx)
	if err != nil {
		log.Printf("Error calling app API for send logs: %v", err)
		return fmt.Errorf("app API call failed: %w", err)
	}

	if !result.Success {
		log.Printf("App API reported error for send logs: %s - %s", result.Message, result.Error)
		return fmt.Errorf("app API reported error: %s", result.Message)
	}

	log.Printf("Send logs task completed: %s", result.Message)
	return nil
}
