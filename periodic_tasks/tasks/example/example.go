package example

import (
	"context"
	"log"

	"vpn-periodic-tasks/internal/config"
)

// Task implements the scheduler.Task interface
// This is an example task - you can use it as a template for creating new tasks
type Task struct{}

// New creates a new example task
func New() *Task {
	return &Task{}
}

// Name returns the task name
func (t *Task) Name() string {
	return "example"
}

// Run executes the task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	log.Println("Example task executed - this is an example task that runs once per hour")
	// Add your task logic here
	return nil
}
