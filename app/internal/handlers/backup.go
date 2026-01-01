package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"vpn-app/internal/telegram"
	"vpn-app/internal/utils"
)

type backupResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	if s.cfg.BackupAdminTgUserID == 0 {
		utils.WriteJSON(w, backupResp{
			Success: false,
			Error:   "BACKUP_ADMIN_TG_USER_ID is not set",
		})
		return
	}

	if s.cfg.BotToken == "" {
		utils.WriteJSON(w, backupResp{
			Success: false,
			Error:   "BOT_TOKEN is not set",
		})
		return
	}

	ctx := r.Context()

	// Create temporary directory for backup
	tmpDir, err := os.MkdirTemp("", "db_backup_*")
	if err != nil {
		http.Error(w, "failed to create temp dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("vpn_backup_%s.sql", timestamp)
	backupPath := filepath.Join(tmpDir, backupFilename)

	log.Printf("creating database backup: %s", backupPath)

	// Build pg_dump command with PGPASSWORD from config
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", s.cfg.PG.Password))

	pgDumpCmd := exec.CommandContext(ctx, "pg_dump",
		"-h", s.cfg.PG.Host,
		"-p", s.cfg.PG.Port,
		"-U", s.cfg.PG.User,
		"-d", s.cfg.PG.DB,
		"-F", "p", // Plain SQL format
		"-f", backupPath,
	)
	pgDumpCmd.Env = env

	// Execute pg_dump
	output, err := pgDumpCmd.CombinedOutput()
	if err != nil {
		log.Printf("pg_dump failed: %v, output: %s", err, string(output))
		utils.WriteJSON(w, backupResp{
			Success: false,
			Error:   fmt.Sprintf("pg_dump failed: %v", err),
		})
		return
	}

	log.Printf("backup created successfully: %s", backupPath)

	// Read backup file
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		http.Error(w, "failed to read backup file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare caption with backup info
	caption := fmt.Sprintf(
		"📦 Database Backup\n\n"+
			"Date: %s\n"+
			"Database: %s\n"+
			"Size: %.2f MB",
		time.Now().Format("2006-01-02 15:04:05"),
		s.cfg.PG.DB,
		float64(len(backupData))/(1024*1024),
	)

	// Send backup via Telegram
	log.Printf("sending backup to Telegram user ID: %d", s.cfg.BackupAdminTgUserID)
	if err := telegram.SendDocument(s.cfg.BotToken, s.cfg.BackupAdminTgUserID, backupFilename, backupData, caption); err != nil {
		log.Printf("failed to send backup to Telegram: %v", err)
		utils.WriteJSON(w, backupResp{
			Success: false,
			Error:   fmt.Sprintf("failed to send telegram document: %v", err),
		})
		return
	}

	log.Printf("backup sent successfully to Telegram")
	utils.WriteJSON(w, backupResp{
		Success: true,
		Message: fmt.Sprintf("Backup created and sent successfully. Size: %.2f MB", float64(len(backupData))/(1024*1024)),
	})
}
