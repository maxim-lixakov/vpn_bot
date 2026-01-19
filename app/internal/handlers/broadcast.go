package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"vpn-app/internal/repo"
	"vpn-app/internal/telegram"
	"vpn-app/internal/utils"
)

type tgBroadcastReq struct {
	AdminTgUserID int64  `json:"admin_tg_user_id"`
	Message       string `json:"message"`
	Target        string `json:"target"` // "all", "with_subscription", "without_subscription"
}

type tgBroadcastResp struct {
	SentCount  int      `json:"sent_count"`
	Recipients []string `json:"recipients"` // список пользователей, которым отправлено
}

func (s *Server) handleTelegramBroadcast(w http.ResponseWriter, r *http.Request) {
	var req tgBroadcastReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// Проверяем, что это админ
	if req.AdminTgUserID != s.cfg.BackupAdminTgUserID {
		http.Error(w, "unauthorized: only admin can broadcast", http.StatusUnauthorized)
		return
	}

	// Получаем список пользователей в зависимости от target
	var users []repo.User
	var err error
	now := time.Now().UTC()

	switch req.Target {
	case "all":
		users, err = s.usersRepo.GetAllUsers(r.Context())
	case "with_subscription":
		users, err = s.usersRepo.GetUsersWithActiveSubscriptions(r.Context(), now)
	case "without_subscription":
		users, err = s.usersRepo.GetUsersWithoutSubscriptions(r.Context())
	default:
		http.Error(w, "invalid target: must be 'all', 'with_subscription', or 'without_subscription'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	if len(users) == 0 {
		utils.WriteJSON(w, tgBroadcastResp{
			SentCount:  0,
			Recipients: []string{},
		})
		return
	}

	// Отправляем сообщения асинхронно
	var sentCount int
	recipients := make([]string, 0, len(users))
	var recipientsMutex sync.Mutex
	var wg sync.WaitGroup

	for _, user := range users {
		wg.Add(1)
		go func(u repo.User) {
			defer wg.Done()
			if err := telegram.SendMessage(s.cfg.BotToken, u.TgUserID, req.Message); err != nil {
				log.Printf("failed to send broadcast message to user %d: %v", u.TgUserID, err)
				return
			}
			// Формируем идентификатор пользователя для отчета
			identifier := fmt.Sprintf("%d", u.TgUserID)
			if u.Username.Valid && u.Username.String != "" {
				identifier = "@" + u.Username.String
			}
			recipientsMutex.Lock()
			recipients = append(recipients, identifier)
			sentCount++
			recipientsMutex.Unlock()
		}(user)
	}

	// Ждем завершения всех отправок (с таймаутом)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Все отправки завершены
	case <-time.After(30 * time.Second):
		log.Printf("broadcast timeout: not all messages sent")
	}

	// Формируем отчет для админа
	reportMsg := fmt.Sprintf(
		"📢 Рассылка завершена\n\n"+
			"Целевая группа: %s\n"+
			"Отправлено: %d из %d\n"+
			"Получатели:\n%s",
		req.Target,
		sentCount,
		len(users),
		strings.Join(recipients, "\n"),
	)

	// Отправляем отчет админу
	if s.cfg.BotToken != "" && s.cfg.BackupAdminTgUserID > 0 {
		go func() {
			if err := telegram.SendMessage(s.cfg.BotToken, s.cfg.BackupAdminTgUserID, reportMsg); err != nil {
				log.Printf("failed to send broadcast report to admin: %v", err)
			}
		}()
	}

	utils.WriteJSON(w, tgBroadcastResp{
		SentCount:  sentCount,
		Recipients: recipients,
	})
}
