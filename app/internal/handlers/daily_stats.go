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

type dailyStatsResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) handleDailyStats(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	last24h := now.Add(-24 * time.Hour)

	// 1. Общее количество пользователей
	totalUsers, err := s.usersRepo.CountAll(r.Context())
	if err != nil {
		log.Printf("failed to count users: %v", err)
		utils.WriteJSON(w, dailyStatsResp{Success: false, Error: err.Error()})
		return
	}

	// 1.1 Новые пользователи за последние 24 часа
	newUsers, err := s.usersRepo.GetUsersCreatedInPeriod(r.Context(), last24h, now)
	if err != nil {
		log.Printf("failed to get new users: %v", err)
		utils.WriteJSON(w, dailyStatsResp{Success: false, Error: err.Error()})
		return
	}

	// 2. Количество активных подписок
	activeSubscriptions, err := s.subsRepo.CountActiveSubscriptions(r.Context(), now)
	if err != nil {
		log.Printf("failed to count active subscriptions: %v", err)
		utils.WriteJSON(w, dailyStatsResp{Success: false, Error: err.Error()})
		return
	}

	// 3. Подписки за последние 24 часа
	recentSubscriptions, err := s.subsRepo.GetSubscriptionsCreatedInPeriod(r.Context(), last24h, now)
	if err != nil {
		log.Printf("failed to get recent subscriptions: %v", err)
		utils.WriteJSON(w, dailyStatsResp{Success: false, Error: err.Error()})
		return
	}

	// 4. Истекшие подписки за последние 24 часа
	expiredSubscriptions, err := s.subsRepo.GetSubscriptionsExpiredInPeriod(r.Context(), last24h, now)
	if err != nil {
		log.Printf("failed to get expired subscriptions: %v", err)
		utils.WriteJSON(w, dailyStatsResp{Success: false, Error: err.Error()})
		return
	}

	// 5. Промокоды — общая статистика использований
	promocodesUsage, err := s.promocodesRepo.GetAllWithUsage(r.Context())
	if err != nil {
		log.Printf("failed to get promocodes usage: %v", err)
		utils.WriteJSON(w, dailyStatsResp{Success: false, Error: err.Error()})
		return
	}

	// 6. Реферальные переходы за последние 24 часа
	referralUsages, err := s.promocodeUsagesRepo.GetReferralUsagesInPeriod(r.Context(), last24h, now)
	if err != nil {
		log.Printf("failed to get referral usages: %v", err)
		utils.WriteJSON(w, dailyStatsResp{Success: false, Error: err.Error()})
		return
	}

	// Формируем сообщение для администратора
	var message strings.Builder
	message.WriteString(fmt.Sprintf("📊 Ежедневная статистика бота\n%s\n\n", now.Format("02.01.2006 15:04 UTC")))

	message.WriteString(fmt.Sprintf("👥 Всего пользователей: %d\n", totalUsers))
	message.WriteString(fmt.Sprintf("✅ Активных подписок: %d\n\n", activeSubscriptions))

	// Новые пользователи за 24 часа
	message.WriteString(fmt.Sprintf("🆕 Новых пользователей за 24 часа: %d\n", len(newUsers)))
	if len(newUsers) > 0 {
		for i, u := range newUsers {
			userDisplay := getUserDisplay(u.Username.String, u.Username.Valid, u.TgUserID)
			message.WriteString(fmt.Sprintf("   %d. %s\n", i+1, userDisplay))
		}
	}
	message.WriteString("\n")

	// Подписки за 24 часа
	message.WriteString(fmt.Sprintf("📝 Новых подписок за 24 часа: %d\n", len(recentSubscriptions)))
	if len(recentSubscriptions) > 0 {
		var paidCount, freeCount int
		for _, sub := range recentSubscriptions {
			if sub.AmountMinor > 0 {
				paidCount++
			} else {
				freeCount++
			}
		}
		message.WriteString(fmt.Sprintf("   💰 Платных: %d\n", paidCount))
		message.WriteString(fmt.Sprintf("   🎁 Бесплатных: %d\n", freeCount))
		message.WriteString("\n")

		for i, sub := range recentSubscriptions {
			userDisplay := getUserDisplay(sub.Username.String, sub.Username.Valid, sub.TgUserID)
			if sub.AmountMinor > 0 {
				price := formatPrice(sub.AmountMinor, sub.Currency)
				message.WriteString(fmt.Sprintf("   %d. %s - %s 💰\n", i+1, userDisplay, price))
			} else {
				message.WriteString(fmt.Sprintf("   %d. %s - бесплатно 🎁\n", i+1, userDisplay))
			}
		}
		message.WriteString("\n")
	}

	// Истекшие подписки за 24 часа
	message.WriteString(fmt.Sprintf("⏰ Истекших подписок за 24 часа: %d\n", len(expiredSubscriptions)))
	if len(expiredSubscriptions) > 0 {
		for i, sub := range expiredSubscriptions {
			userDisplay := getUserDisplay(sub.Username.String, sub.Username.Valid, sub.TgUserID)
			message.WriteString(fmt.Sprintf("   %d. %s\n", i+1, userDisplay))
		}
	}

	// Промокоды
	message.WriteString(fmt.Sprintf("\n🎟 Промокоды (использовано): %d\n", len(promocodesUsage)))
	for i, p := range promocodesUsage {
		if p.IsReferral {
			message.WriteString(fmt.Sprintf("   %d. %s — %d/%d раз 🔗\n", i+1, p.PromocodeName, p.TimesUsed, p.TimesToBeUsed))
		} else {
			message.WriteString(fmt.Sprintf("   %d. %s — %d/%d раз\n", i+1, p.PromocodeName, p.TimesUsed, p.TimesToBeUsed))
		}
	}

	// Реферальные переходы за 24 часа
	message.WriteString(fmt.Sprintf("\n🔗 Реферальных переходов за 24 часа: %d\n", len(referralUsages)))
	for i, ref := range referralUsages {
		referrer := getUserDisplay(ref.ReferrerUsername.String, ref.ReferrerUsername.Valid, ref.ReferrerTgUserID)
		receiver := getUserDisplay(ref.ReceiverUsername.String, ref.ReceiverUsername.Valid, ref.ReceiverTgUserID)
		message.WriteString(fmt.Sprintf("   %d. %s → %s\n", i+1, referrer, receiver))
	}

	// Отправляем сообщение администратору
	if s.cfg.BackupAdminTgUserID > 0 && s.cfg.BotToken != "" {
		if err := telegram.SendMessage(s.cfg.BotToken, s.cfg.BackupAdminTgUserID, message.String()); err != nil {
			log.Printf("failed to send daily stats to admin: %v", err)
			utils.WriteJSON(w, dailyStatsResp{Success: false, Error: fmt.Sprintf("failed to send message: %v", err)})
			return
		}
		log.Printf("sent daily stats to admin (tg_user_id: %d)", s.cfg.BackupAdminTgUserID)
	} else {
		log.Printf("skipping daily stats notification: admin tg user id or bot token not configured")
	}

	utils.WriteJSON(w, dailyStatsResp{
		Success: true,
		Message: fmt.Sprintf("Daily stats sent: %d users, %d active subs, %d new, %d expired",
			totalUsers, activeSubscriptions, len(recentSubscriptions), len(expiredSubscriptions)),
	})
}

func getUserDisplay(username string, usernameValid bool, tgUserID int64) string {
	if usernameValid && username != "" {
		return "@" + username
	}
	return fmt.Sprintf("%d", tgUserID)
}

func formatPrice(amountMinor int64, currency string) string {
	amount := float64(amountMinor) / 100.0
	switch currency {
	case "RUB":
		return fmt.Sprintf("%.2f ₽", amount)
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	case "EUR":
		return fmt.Sprintf("€%.2f", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}
