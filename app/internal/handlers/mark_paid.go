package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vpn-app/internal/domain"
	"vpn-app/internal/repo"
	"vpn-app/internal/telegram"
	"vpn-app/internal/utils"
)

type tgMarkPaidReq struct {
	TgUserID    int64   `json:"tg_user_id"`
	Kind        string  `json:"kind"`         // "vpn" | "country_request"
	CountryCode *string `json:"country_code"` // для vpn обязательно

	AmountMinor             int64  `json:"amount_minor"`
	Currency                string `json:"currency"`
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
	Months                  int    `json:"months"` // количество месяцев (0 = использовать дефолт)
}

type tgMarkPaidResp struct {
	ActiveUntil time.Time `json:"active_until"`
}

func (s *Server) handleTelegramMarkPaid(w http.ResponseWriter, r *http.Request) {
	var req tgMarkPaidReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	kind := strings.TrimSpace(strings.ToLower(req.Kind))
	if kind == "" {
		http.Error(w, "kind is required", http.StatusBadRequest)
		return
	}

	currency := strings.TrimSpace(strings.ToUpper(req.Currency))
	if currency == "" {
		http.Error(w, "currency is required", http.StatusBadRequest)
		return
	}

	// Проверяем, является ли это промокодом (до обработки country_code)
	// Промокод определяется по ProviderPaymentChargeID == "promocode" или TelegramPaymentChargeID == "promocode"
	isPromocode := strings.TrimSpace(req.ProviderPaymentChargeID) == "promocode" ||
		strings.TrimSpace(req.TelegramPaymentChargeID) == "promocode"

	var cc sql.NullString
	if req.CountryCode != nil {
		v := strings.TrimSpace(strings.ToLower(*req.CountryCode))
		if v != "" {
			cc = sql.NullString{String: v, Valid: true}
		}
	}

	// Для VPN подписки country_code обязателен, кроме случаев когда это промокод
	// В этом случае country_code может быть пустым (NULL) - подписка будет без привязки к стране
	if kind == "vpn" {
		if !cc.Valid && !isPromocode {
			log.Printf("mark_paid: kind=vpn, country_code invalid, isPromocode=%v, ProviderPaymentChargeID=%q, TelegramPaymentChargeID=%q",
				isPromocode, req.ProviderPaymentChargeID, req.TelegramPaymentChargeID)
			http.Error(w, "country_code is required for vpn", http.StatusBadRequest)
			return
		}
	}

	user, ok, err := s.usersRepo.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil || !ok {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	// Проверяем, является ли это продлением подписки
	// Payload приходит в ProviderPaymentChargeID или TelegramPaymentChargeID
	isRenewal := (req.ProviderPaymentChargeID != "" && strings.HasPrefix(req.ProviderPaymentChargeID, s.cfg.PaymentsVPNRenewalPayload+":")) ||
		(req.TelegramPaymentChargeID != "" && strings.HasPrefix(req.TelegramPaymentChargeID, s.cfg.PaymentsVPNRenewalPayload+":"))

	var until time.Time
	if isRenewal && kind == "vpn" {
		// Продление существующей подписки
		// Извлекаем subscription_id из payload (формат: "vpn_renewal_v1:subscription_id:country_code")
		var subscriptionID int64
		payload := req.ProviderPaymentChargeID
		if payload == "" {
			payload = req.TelegramPaymentChargeID
		}

		parts := strings.Split(payload, ":")
		if len(parts) >= 2 {
			if id, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				subscriptionID = id
			}
		}

		if subscriptionID == 0 {
			http.Error(w, "invalid renewal payload: subscription_id not found", http.StatusBadRequest)
			return
		}

		// Получаем существующую подписку
		sub, found, err := s.subsRepo.GetByID(r.Context(), subscriptionID)
		if err != nil {
			http.Error(w, "db error: failed to get subscription: "+err.Error(), http.StatusBadGateway)
			return
		}
		if !found {
			http.Error(w, "subscription not found", http.StatusNotFound)
			return
		}

		// Проверяем, что подписка принадлежит пользователю
		if sub.UserID != user.ID {
			http.Error(w, "subscription does not belong to user", http.StatusForbidden)
			return
		}

		oldUntil := sub.ActiveUntil
		newUntil := oldUntil.AddDate(0, 1, 0)

		if err := s.subsRepo.UpdateActiveUntil(r.Context(), subscriptionID, newUntil); err != nil {
			http.Error(w, "db error: failed to update subscription: "+err.Error(), http.StatusBadGateway)
			return
		}

		_, err = s.paymentsRepo.Insert(r.Context(), repo.InsertPaymentArgs{
			SubscriptionID:          subscriptionID,
			UserID:                  user.ID,
			Provider:                "telegram",
			AmountMinor:             req.AmountMinor,
			Currency:                currency,
			PaidAt:                  time.Now().UTC(),
			TelegramPaymentChargeID: sql.NullString{String: req.TelegramPaymentChargeID, Valid: strings.TrimSpace(req.TelegramPaymentChargeID) != ""},
			ProviderPaymentChargeID: sql.NullString{String: req.ProviderPaymentChargeID, Valid: strings.TrimSpace(req.ProviderPaymentChargeID) != ""},
			Months:                  1,
		})
		if err != nil {
			http.Error(w, "db error: failed to record payment: "+err.Error(), http.StatusBadGateway)
			return
		}

		// Получаем название страны
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

		// Отправляем уведомление пользователю о продлении
		message := fmt.Sprintf(
			"✅ Ваша VPN подписка для страны %s успешно продлена на +1 месяц!\n\nБыло активно до: %s\nСтало активно до: %s",
			countryName,
			oldUntil.Format("2006-01-02 15:04"),
			newUntil.Format("2006-01-02 15:04"),
		)

		go func() {
			if err := telegram.SendMessage(s.cfg.BotToken, user.TgUserID, message); err != nil {
				log.Printf("failed to send renewal confirmation to user %d: %v", user.TgUserID, err)
			}
		}()

		until = newUntil
	} else {
		// Обычная оплата - создаем новую подписку
		// После миграции access_key_id будет привязан через issue-key после выдачи ключа
		// Не ищем ключ автоматически здесь - issue-key должен привязать его
		var subscriptionID int64
		subscriptionID, until, err = s.subsRepo.MarkPaid(r.Context(), repo.MarkPaidArgs{
			UserID:                  user.ID,
			Kind:                    kind,
			CountryCode:             cc,
			AccessKeyID:             sql.NullInt64{}, // Будет привязан через issue-key
			Provider:                "telegram",
			AmountMinor:             req.AmountMinor,
			Currency:                currency,
			TelegramPaymentChargeID: sql.NullString{String: req.TelegramPaymentChargeID, Valid: strings.TrimSpace(req.TelegramPaymentChargeID) != ""},
			ProviderPaymentChargeID: sql.NullString{String: req.ProviderPaymentChargeID, Valid: strings.TrimSpace(req.ProviderPaymentChargeID) != ""},
			PaidAt:                  time.Now().UTC(),
			Months:                  req.Months,
		})
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
			return
		}

		// Создаем запись о платеже в таблице payments
		_, err = s.paymentsRepo.Insert(r.Context(), repo.InsertPaymentArgs{
			SubscriptionID:          subscriptionID,
			UserID:                  user.ID,
			Provider:                "telegram",
			AmountMinor:             req.AmountMinor,
			Currency:                currency,
			PaidAt:                  time.Now().UTC(),
			TelegramPaymentChargeID: sql.NullString{String: req.TelegramPaymentChargeID, Valid: strings.TrimSpace(req.TelegramPaymentChargeID) != ""},
			ProviderPaymentChargeID: sql.NullString{String: req.ProviderPaymentChargeID, Valid: strings.TrimSpace(req.ProviderPaymentChargeID) != ""},
			Months:                  req.Months,
		})
		if err != nil {
			log.Printf("failed to insert payment record for subscription %d: %v", subscriptionID, err)
			// Не возвращаем ошибку, так как подписка уже создана
		}

		// state меняем только для vpn
		if kind == "vpn" {
			if _, err := s.statesRepo.Get(r.Context(), user.ID); err == nil {
				_, _ = s.statesRepo.Set(r.Context(), user.ID, domain.StateIssueKey, cc)
			}
		}
	}

	utils.WriteJSON(w, tgMarkPaidResp{ActiveUntil: until})
}
