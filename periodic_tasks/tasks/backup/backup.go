package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"vpn-periodic-tasks/internal/config"
	"vpn-periodic-tasks/internal/utils"
)

// Task implements the scheduler.Task interface for database backups
type Task struct{}

// New creates a new backup task
func New() *Task {
	return &Task{}
}

// Name returns the task name
func (t *Task) Name() string {
	return "backup"
}

// Run executes the backup task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	if cfg.BackupAdminTgUserID == 0 {
		return fmt.Errorf("BACKUP_ADMIN_TG_USER_ID is not set")
	}

	if cfg.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is not set")
	}

	// Create temporary directory for backup
	tmpDir, err := os.MkdirTemp("", "db_backup_*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("vpn_backup_%s.sql", timestamp)
	backupPath := filepath.Join(tmpDir, backupFilename)

	log.Printf("creating database backup: %s", backupPath)

	// Build pg_dump command
	// Set PGPASSWORD environment variable for pg_dump
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", cfg.PG.Password))

	pgDumpCmd := exec.CommandContext(ctx, "pg_dump",
		"-h", cfg.PG.Host,
		"-p", cfg.PG.Port,
		"-U", cfg.PG.User,
		"-d", cfg.PG.DB,
		"-F", "p", // Plain SQL format
		"-f", backupPath,
	)
	pgDumpCmd.Env = env

	// Execute pg_dump
	output, err := pgDumpCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump failed: %w, output: %s", err, string(output))
	}

	log.Printf("backup created successfully: %s", backupPath)

	// Read backup file
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup file: %w", err)
	}

	// Prepare caption with backup info
	caption := fmt.Sprintf(
		"📦 Database Backup\n\n"+
			"Date: %s\n"+
			"Database: %s\n"+
			"Size: %.2f MB",
		time.Now().Format("2006-01-02 15:04:05"),
		cfg.PG.DB,
		float64(len(backupData))/(1024*1024),
	)

	// Send backup via Telegram
	log.Printf("sending backup to Telegram user ID: %d", cfg.BackupAdminTgUserID)
	if err := utils.SendTelegramDocument(
		cfg.BotToken,
		cfg.BackupAdminTgUserID,
		backupFilename,
		backupData,
		caption,
	); err != nil {
		return fmt.Errorf("send telegram document: %w", err)
	}

	log.Printf("backup sent successfully to Telegram")
	return nil
}
