package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"vpn-app/internal/telegram"
	"vpn-app/internal/utils"
)

type subscriptionRenewalReminderResp struct {
	NotifiedCount int      `json:"notified_count"`
	Errors        []string `json:"errors,omitempty"`
}

func (s *Server) handleSubscriptionRenewalReminder(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	tomorrowStart := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	tomorrowEnd := tomorrowStart.AddDate(0, 0, 1).Add(-time.Second)

	// Получаем подписки, истекающие завтра
	expiringSubs, err := s.subsRepo.GetSubscriptionsExpiringTomorrow(r.Context(), tomorrowStart, tomorrowEnd)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	if len(expiringSubs) == 0 {
		utils.WriteJSON(w, subscriptionRenewalReminderResp{
			NotifiedCount: 0,
		})
		return
	}

	if s.cfg.BotToken == "" {
		http.Error(w, "BOT_TOKEN is not set", http.StatusBadRequest)
		return
	}

	if s.cfg.PaymentsProviderToken == "" {
		http.Error(w, "PAYMENTS_PROVIDER_TOKEN is not set", http.StatusBadRequest)
		return
	}

	if s.cfg.PaymentsVPNPriceMinor <= 0 {
		http.Error(w, "PAYMENTS_VPN_PRICE_MINOR is not set or invalid", http.StatusBadRequest)
		return
	}

	var notifiedCount int
	errorsChan := make(chan string, len(expiringSubs))

	for _, sub := range expiringSubs {
		countryCode := ""
		if sub.CountryCode.Valid && sub.CountryCode.String != "" {
			countryCode = sub.CountryCode.String
		}
		serverName := ""
		if countryCode != "" {
			if server, ok := s.cfg.Servers[countryCode]; ok {
				serverName = server.Name
			}
		}
		countryName := utils.GetCountryName(countryCode, serverName)

		notificationMsg := fmt.Sprintf(
			"⏰ Напоминание: завтра (%s) истекает срок действия вашей VPN подписки для страны %s.\n\nДля продолжения использования VPN необходимо продлить подписку.",
			sub.ActiveUntil.Format("2006-01-02 15:04"),
			countryName,
		)

		// Отправляем уведомление асинхронно
		go func(tgUserID int64, msg string) {
			if err := telegram.SendMessage(s.cfg.BotToken, tgUserID, msg); err != nil {
				log.Printf("failed to send renewal notification to user %d: %v", tgUserID, err)
			}
		}(sub.TgUserID, notificationMsg)

		// Формируем payload с информацией о подписке для продления
		renewalPayload := fmt.Sprintf("%s:%d:%s", s.cfg.PaymentsVPNRenewalPayload, sub.SubscriptionID, countryCode)

		prices := []telegram.LabeledPrice{
			{Label: "VPN 1 month prolongation", Amount: int(s.cfg.PaymentsVPNPriceMinor)},
		}

		// Отправляем инвойс асинхронно
		go func(tgUserID int64, payload string, subID int64) {
			if err := telegram.SendInvoice(
				s.cfg.BotToken,
				tgUserID,
				s.cfg.PaymentsVPNTitle,
				s.cfg.PaymentsVPNDescription,
				payload,
				s.cfg.PaymentsProviderToken,
				s.cfg.PaymentsCurrency,
				prices,
			); err != nil {
				log.Printf("failed to send renewal invoice to user %d: %v", tgUserID, err)
				errorsChan <- fmt.Sprintf("user %d (subscription %d): failed to send invoice: %v", tgUserID, subID, err)
			}
		}(sub.TgUserID, renewalPayload, sub.SubscriptionID)

		notifiedCount++
		log.Printf("sent renewal reminder and invoice to user %d (subscription %d, country %s)",
			sub.TgUserID, sub.SubscriptionID, countryCode)
	}

	// Собираем ошибки (с таймаутом)
	var errors []string
	done := make(chan bool)
	go func() {
		time.Sleep(5 * time.Second) // Даем время на отправку
		done <- true
	}()

	for {
		select {
		case err := <-errorsChan:
			errors = append(errors, err)
		case <-done:
			goto done
		}
	}
done:

	utils.WriteJSON(w, subscriptionRenewalReminderResp{
		NotifiedCount: notifiedCount,
		Errors:        errors,
	})
}
