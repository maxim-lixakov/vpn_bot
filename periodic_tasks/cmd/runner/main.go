package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"vpn-periodic-tasks/internal/appclient"
	"vpn-periodic-tasks/internal/config"
	"vpn-periodic-tasks/internal/scheduler"
	"vpn-periodic-tasks/tasks/backup"
	"vpn-periodic-tasks/tasks/revoke_expired_keys"
)

func main() {
	log.Println("starting periodic tasks runner...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create app API client
	appClient := appclient.New(cfg.AppAddr, cfg.AppInternalToken)
	log.Printf("app client initialized: %s", cfg.AppAddr)

	sched := scheduler.New(cfg)

	sched.RegisterTask(backup.New(appClient))
	sched.RegisterTask(revoke_expired_keys.New(appClient))

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
