package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"vpn-app/internal/domain"
	"vpn-app/internal/repo"
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

	// После миграции access_key_id будет привязан через issue-key после выдачи ключа
	// Не ищем ключ автоматически здесь - issue-key должен привязать его
	until, err := s.subsRepo.MarkPaid(r.Context(), repo.MarkPaidArgs{
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

	// state меняем только для vpn
	if kind == "vpn" {
		if _, err := s.statesRepo.Get(r.Context(), user.ID); err == nil {
			_, _ = s.statesRepo.Set(r.Context(), user.ID, domain.StateIssueKey, cc)
		}
	}

	utils.WriteJSON(w, tgMarkPaidResp{ActiveUntil: until})
}
