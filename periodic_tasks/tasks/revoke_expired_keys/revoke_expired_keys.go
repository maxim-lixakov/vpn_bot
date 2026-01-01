package revoke_expired_keys

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"vpn-periodic-tasks/internal/config"
	"vpn-periodic-tasks/internal/telegram"
)

// Task implements the scheduler.Task interface for revoking expired access keys
type Task struct{}

// New creates a new revoke expired keys task
func New() *Task {
	return &Task{}
}

// Name returns the task name
func (t *Task) Name() string {
	return "revoke_expired_keys"
}

// Run executes the task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	if cfg.AppAddr == "" {
		return fmt.Errorf("APP_ADDR is not set")
	}
	if cfg.AppInternalToken == "" {
		return fmt.Errorf("APP_INTERNAL_TOKEN is not set")
	}

	// Build the API endpoint URL
	// APP_ADDR might be just ":8080" or "http://app:8080", so we need to handle both cases
	appAddr := cfg.AppAddr
	if !strings.HasPrefix(appAddr, "http://") && !strings.HasPrefix(appAddr, "https://") {
		// If it's just a port or host:port, prepend http://
		if strings.HasPrefix(appAddr, ":") {
			appAddr = "http://app" + appAddr
		} else {
			appAddr = "http://" + appAddr
		}
	}
	url := fmt.Sprintf("%s/v1/revoke-expired-keys", appAddr)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Internal-Token", cfg.AppInternalToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var body bytes.Buffer
		body.ReadFrom(resp.Body)
		return fmt.Errorf("app api error: %s, body: %s", resp.Status, body.String())
	}

	// Parse response
	var result struct {
		RevokedCount int `json:"revoked_count"`
		Revoked      []struct {
			SubscriptionID int64  `json:"subscription_id"`
			TgUserID       int64  `json:"tg_user_id"`
			CountryCode    string `json:"country_code"`
		} `json:"revoked,omitempty"`
		Errors []string `json:"errors,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	log.Printf("revoked %d expired access keys", result.RevokedCount)
	if len(result.Errors) > 0 {
		log.Printf("encountered %d errors during revocation:", len(result.Errors))
		for _, errMsg := range result.Errors {
			log.Printf("  - %s", errMsg)
		}
	}

	// Send notification to admin if there are revoked subscriptions
	if result.RevokedCount > 0 && cfg.BackupAdminTgUserID > 0 && cfg.BotToken != "" {
		var message strings.Builder
		message.WriteString(fmt.Sprintf("🔒 Отозвано %d истекших VPN ключей:\n\n", result.RevokedCount))

		for i, revoked := range result.Revoked {
			if revoked.TgUserID > 0 {
				message.WriteString(fmt.Sprintf("%d. Подписка #%d\n   Пользователь: %d\n   Страна: %s\n\n",
					i+1, revoked.SubscriptionID, revoked.TgUserID, revoked.CountryCode))
			} else {
				message.WriteString(fmt.Sprintf("%d. Подписка #%d\n   Пользователь: не найден\n   Страна: %s\n\n",
					i+1, revoked.SubscriptionID, revoked.CountryCode))
			}
		}

		if len(result.Errors) > 0 {
			message.WriteString(fmt.Sprintf("\n⚠️ Ошибки (%d):\n", len(result.Errors)))
			for _, errMsg := range result.Errors {
				message.WriteString(fmt.Sprintf("  • %s\n", errMsg))
			}
		}

		// Send message to admin
		if err := telegram.SendTelegramMessage(cfg.BotToken, cfg.BackupAdminTgUserID, message.String()); err != nil {
			log.Printf("failed to send telegram notification to admin: %v", err)
		} else {
			log.Printf("sent revocation report to admin (tg_user_id: %d)", cfg.BackupAdminTgUserID)
		}
	}

	return nil
}
