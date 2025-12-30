package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"

	"vpn-periodic-tasks/internal/config"
)

// Task defines the interface for periodic tasks
type Task interface {
	Name() string
	Run(ctx context.Context, cfg config.Config) error
}

// Scheduler manages and runs periodic tasks
type Scheduler struct {
	cron    *cron.Cron
	tasks   map[string]Task
	cfg     config.Config
	running sync.Map // Track running tasks to prevent overlapping executions
}

// New creates a new scheduler
func New(cfg config.Config) *Scheduler {
	return &Scheduler{
		cron:  cron.New(cron.WithSeconds()), // Use seconds precision for cron
		tasks: make(map[string]Task),
		cfg:   cfg,
	}
}

// RegisterTask registers a task with the scheduler
func (s *Scheduler) RegisterTask(task Task) {
	s.tasks[task.Name()] = task
}

// Start starts the scheduler with the given schedules
func (s *Scheduler) Start(schedules []config.TaskSchedule) error {
	for _, schedule := range schedules {
		task, ok := s.tasks[schedule.TaskName]
		if !ok {
			return fmt.Errorf("task %s not registered", schedule.TaskName)
		}

		// Create a closure to capture the task
		taskName := task.Name()
		_, err := s.cron.AddFunc(schedule.Schedule, func() {
			// Check if task is already running
			if _, running := s.running.LoadOrStore(taskName, true); running {
				log.Printf("task %s is already running, skipping this execution", taskName)
				return
			}
			defer s.running.Delete(taskName)

			ctx := context.Background()
			log.Printf("starting task: %s", taskName)
			if err := task.Run(ctx, s.cfg); err != nil {
				log.Printf("task %s failed: %v", taskName, err)
			} else {
				log.Printf("task %s completed successfully", taskName)
			}
		})
		if err != nil {
			return fmt.Errorf("failed to schedule task %s: %w", schedule.TaskName, err)
		}

		log.Printf("scheduled task %s with schedule: %s", schedule.TaskName, schedule.Schedule)
	}

	s.cron.Start()
	log.Println("scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("stopping scheduler...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("scheduler stopped")
}
