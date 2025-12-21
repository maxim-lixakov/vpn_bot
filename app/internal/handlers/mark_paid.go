package handlers

import (
	"database/sql"
	"encoding/json"
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

	var cc sql.NullString
	if req.CountryCode != nil {
		v := strings.TrimSpace(strings.ToLower(*req.CountryCode))
		if v != "" {
			cc = sql.NullString{String: v, Valid: true}
		}
	}
	if kind == "vpn" && !cc.Valid {
		http.Error(w, "country_code is required for vpn", http.StatusBadRequest)
		return
	}

	user, ok, err := s.users.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil || !ok {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	until, err := s.subs.MarkPaid(r.Context(), repo.MarkPaidArgs{
		UserID:                  user.ID,
		Kind:                    kind,
		CountryCode:             cc,
		Provider:                "telegram",
		AmountMinor:             req.AmountMinor,
		Currency:                currency,
		TelegramPaymentChargeID: sql.NullString{String: req.TelegramPaymentChargeID, Valid: strings.TrimSpace(req.TelegramPaymentChargeID) != ""},
		ProviderPaymentChargeID: sql.NullString{String: req.ProviderPaymentChargeID, Valid: strings.TrimSpace(req.ProviderPaymentChargeID) != ""},
		PaidAt:                  time.Now().UTC(),
	})
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// state меняем только для vpn (для country_request пусть остаётся как есть)
	if kind == "vpn" {
		_, err := s.states.Get(r.Context(), user.ID)
		if err == nil {
			// проставим выбранную страну явно
			_, _ = s.states.Set(r.Context(), user.ID, domain.StateIssueKey, cc)
		}
	}

	utils.WriteJSON(w, tgMarkPaidResp{ActiveUntil: until})
}
