package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"vpn-app/internal/telegram"
	"vpn-app/internal/utils"
)

type revokedSubscriptionInfo struct {
	SubscriptionID int64  `json:"subscription_id"`
	TgUserID       int64  `json:"tg_user_id"`
	CountryCode    string `json:"country_code"`
}

type revokeExpiredKeysResp struct {
	RevokedCount int                       `json:"revoked_count"`
	Revoked      []revokedSubscriptionInfo `json:"revoked,omitempty"`
	Errors       []string                  `json:"errors,omitempty"`
}

func (s *Server) handleRevokeExpiredKeys(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	// Получаем список истекших подписок с активными ключами
	expiredSubs, err := s.subsRepo.GetExpiredSubscriptionsWithActiveKeys(r.Context(), now)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	if len(expiredSubs) == 0 {
		utils.WriteJSON(w, revokeExpiredKeysResp{
			RevokedCount: 0,
		})
		return
	}

	var revokedCount int
	var revoked []revokedSubscriptionInfo
	var errors []string

	for _, sub := range expiredSubs {
		// Получаем country_code для определения правильного outline client
		if !sub.CountryCode.Valid || sub.CountryCode.String == "" {
			errors = append(errors, fmt.Sprintf("subscription %d: country_code is missing", sub.SubscriptionID))
			continue
		}

		countryCode := strings.TrimSpace(strings.ToLower(sub.CountryCode.String))
		client, ok := s.clients[countryCode]
		if !ok {
			errors = append(errors, fmt.Sprintf("subscription %d: outline client not found for country %s", sub.SubscriptionID, countryCode))
			continue
		}

		// Отзываем ключ в Outline API
		if err := client.DeleteAccessKey(r.Context(), sub.OutlineKeyID); err != nil {
			log.Printf("failed to revoke outline key %s for subscription %d: %v", sub.OutlineKeyID, sub.SubscriptionID, err)
			errors = append(errors, fmt.Sprintf("subscription %d: failed to revoke outline key: %v", sub.SubscriptionID, err))
			continue
		}

		// Помечаем ключ как отозванный в базе данных
		if err := s.keysRepo.Revoke(r.Context(), sub.AccessKeyID, now); err != nil {
			log.Printf("failed to mark access key %d as revoked for subscription %d: %v", sub.AccessKeyID, sub.SubscriptionID, err)
			errors = append(errors, fmt.Sprintf("subscription %d: failed to mark key as revoked: %v", sub.SubscriptionID, err))
			continue
		}

		// Получаем информацию о пользователе для отправки уведомления
		user, ok, err := s.usersRepo.GetByID(r.Context(), sub.UserID)
		if err == nil && ok && s.cfg.BotToken != "" {
			serverName := ""
			if server, ok := s.cfg.Servers[countryCode]; ok {
				serverName = server.Name
			}
			countryName := utils.GetCountryName(countryCode, serverName)

			// Отправляем уведомление пользователю о том, что его ключ истек
			message := fmt.Sprintf(
				"🔒 Ваш VPN ключ для страны %s был отозван, так как срок действия подписки истек.\n\nДля продолжения использования VPN необходимо продлить подписку.",
				countryName,
			)
			// Отправляем асинхронно, чтобы не блокировать процесс
			go func() {
				if err := telegram.SendMessage(s.cfg.BotToken, user.TgUserID, message); err != nil {
					log.Printf("failed to send telegram message to user %d (tg_user_id %d): %v", sub.UserID, user.TgUserID, err)
				}
			}()

			// Сохраняем информацию об отозванной подписке для отчета администратору
			revoked = append(revoked, revokedSubscriptionInfo{
				SubscriptionID: sub.SubscriptionID,
				TgUserID:       user.TgUserID,
				CountryCode:    strings.ToUpper(countryCode),
			})
		} else {
			// Если не удалось получить пользователя, все равно добавляем в список (без tg_user_id)
			revoked = append(revoked, revokedSubscriptionInfo{
				SubscriptionID: sub.SubscriptionID,
				TgUserID:       0, // 0 означает, что tg_user_id не найден
				CountryCode:    strings.ToUpper(countryCode),
			})
		}

		revokedCount++
		log.Printf("revoked access key %d (outline key %s) for subscription %d (user %d, country %s)",
			sub.AccessKeyID, sub.OutlineKeyID, sub.SubscriptionID, sub.UserID, countryCode)
	}

	// Send notification to admin if there are revoked subscriptions
	if revokedCount > 0 && s.cfg.BackupAdminTgUserID > 0 && s.cfg.BotToken != "" {
		var message strings.Builder
		message.WriteString(fmt.Sprintf("🔒 Отозвано %d истекших VPN ключей:\n\n", revokedCount))

		for i, rev := range revoked {
			if rev.TgUserID > 0 {
				message.WriteString(fmt.Sprintf("%d. Подписка #%d\n   Пользователь: %d\n   Страна: %s\n\n",
					i+1, rev.SubscriptionID, rev.TgUserID, rev.CountryCode))
			} else {
				message.WriteString(fmt.Sprintf("%d. Подписка #%d\n   Пользователь: не найден\n   Страна: %s\n\n",
					i+1, rev.SubscriptionID, rev.CountryCode))
			}
		}

		if len(errors) > 0 {
			message.WriteString(fmt.Sprintf("\n⚠️ Ошибки (%d):\n", len(errors)))
			for _, errMsg := range errors {
				message.WriteString(fmt.Sprintf("  • %s\n", errMsg))
			}
		}

		// Send message to admin asynchronously
		go func() {
			if err := telegram.SendMessage(s.cfg.BotToken, s.cfg.BackupAdminTgUserID, message.String()); err != nil {
				log.Printf("failed to send telegram notification to admin: %v", err)
			} else {
				log.Printf("sent revocation report to admin (tg_user_id: %d)", s.cfg.BackupAdminTgUserID)
			}
		}()
	}

	utils.WriteJSON(w, revokeExpiredKeysResp{
		RevokedCount: revokedCount,
		Revoked:      revoked,
		Errors:       errors,
	})
}
