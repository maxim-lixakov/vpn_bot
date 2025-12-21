package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"vpn-app/internal/domain"
	"vpn-app/internal/repo"
	"vpn-app/internal/utils"
)

type tgUpsertReq struct {
	TgUserID     int64   `json:"tg_user_id"`
	Username     *string `json:"username"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	LanguageCode *string `json:"language_code"`
	Phone        *string `json:"phone"`
}

type tgUpsertResp struct {
	UserID          int64      `json:"user_id"`
	State           string     `json:"router"`
	SelectedCountry *string    `json:"selected_country"`
	SubscriptionOK  bool       `json:"subscription_ok"`
	ActiveUntil     *time.Time `json:"active_until"`
}

func (s *Server) handleTelegramUpsert(w http.ResponseWriter, r *http.Request) {
	var req tgUpsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	u := repo.User{
		TgUserID: req.TgUserID,
	}
	if req.Username != nil {
		u.Username = sql.NullString{String: *req.Username, Valid: true}
	}
	if req.FirstName != nil {
		u.FirstName = sql.NullString{String: *req.FirstName, Valid: true}
	}
	if req.LastName != nil {
		u.LastName = sql.NullString{String: *req.LastName, Valid: true}
	}
	if req.LanguageCode != nil {
		u.LanguageCode = sql.NullString{String: *req.LanguageCode, Valid: true}
	}
	if req.Phone != nil {
		u.Phone = sql.NullString{String: *req.Phone, Valid: true}
	}

	user, err := s.users.UpsertByTelegram(r.Context(), u)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	st, err := s.states.EnsureDefault(r.Context(), user.ID, domain.StateMenu)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	now := time.Now().UTC()
	until, ok, err := s.subs.GetActiveUntil(r.Context(), user.ID, now)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	var sel *string
	if st.SelectedCountry.Valid {
		sel = &st.SelectedCountry.String
	}
	var au *time.Time
	if !until.IsZero() {
		au = &until
	}

	utils.WriteJSON(w, tgUpsertResp{
		UserID:          user.ID,
		State:           st.State,
		SelectedCountry: sel,
		SubscriptionOK:  ok,
		ActiveUntil:     au,
	})
}
