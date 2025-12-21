package handlers

import (
	"time"

	"database/sql"
	"encoding/json"
	"net/http"

	"vpn-app/internal/domain"
	"vpn-app/internal/repo"
	"vpn-app/internal/utils"
)

type tgMarkPaidReq struct {
	TgUserID    int64   `json:"tg_user_id"`
	Kind        string  `json:"kind"`         // "vpn" | "country_request"
	CountryCode *string `json:"country_code"` // для vpn

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 || req.Currency == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, ok, err := s.users.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil || !ok {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	until, err := s.subs.MarkPaid(r.Context(), repo.MarkPaidArgs{
		UserID:                  user.ID,
		Provider:                "telegram",
		AmountMinor:             req.AmountMinor,
		Currency:                req.Currency,
		TelegramPaymentChargeID: sql.NullString{String: req.TelegramPaymentChargeID, Valid: req.TelegramPaymentChargeID != ""},
		ProviderPaymentChargeID: sql.NullString{String: req.ProviderPaymentChargeID, Valid: req.ProviderPaymentChargeID != ""},
		PaidAt:                  time.Now().UTC(),
	})
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// move router to ISSUE_KEY (selected_country remains as-is)
	st, err := s.states.Get(r.Context(), user.ID)
	if err == nil {
		_ = func() error {
			_, err := s.states.Set(r.Context(), user.ID, domain.StateIssueKey, st.SelectedCountry)
			return err
		}()
	}

	utils.WriteJSON(w, tgMarkPaidResp{ActiveUntil: until})
}
