package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vpn-periodic-tasks/internal/config"
	"vpn-periodic-tasks/internal/scheduler"
	"vpn-periodic-tasks/tasks/backup"
	"vpn-periodic-tasks/tasks/example"
	"vpn-periodic-tasks/tasks/revoke_expired_keys"
)

func main() {
	log.Println("starting periodic tasks runner...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := cfg.OpenDB()
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("database connection established")

	sched := scheduler.New(cfg)

	sched.RegisterTask(example.New())
	sched.RegisterTask(backup.New())
	sched.RegisterTask(revoke_expired_keys.New())

	schedules := config.GetTaskSchedules()

	if err := sched.Start(schedules); err != nil {
		log.Fatalf("failed to start scheduler: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("periodic tasks runner is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("shutting down...")
	sched.Stop()
	log.Println("shutdown complete")
}
