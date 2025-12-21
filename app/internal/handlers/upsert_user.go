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
	TgUserID  int64   `json:"tg_user_id"`
	Username  *string `json:"username"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Language  *string `json:"language"`
	// optional fields - ignored by app for now
	IsPremium *bool `json:"is_premium"`
}

type tgUpsertResp struct {
	UserID          int64      `json:"user_id"`
	State           string     `json:"state"`
	SelectedCountry *string    `json:"selected_country"`
	SubscriptionOK  bool       `json:"subscription_ok"`
	ActiveUntil     *time.Time `json:"active_until"`
}

func (s *Server) handleTelegramUpsert(w http.ResponseWriter, r *http.Request) {
	var req tgUpsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.TgUserID == 0 {
		http.Error(w, "tg_user_id is required", http.StatusBadRequest)
		return
	}

	user, err := s.users.UpsertByTelegram(r.Context(), repo.User{
		TgUserID:     req.TgUserID,
		Username:     toNullString(req.Username),
		FirstName:    toNullString(req.FirstName),
		LastName:     toNullString(req.LastName),
		LanguageCode: toNullString(req.Language),
		Phone:        sql.NullString{}, // phone can be filled later if you implement contact sharing
	})
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Ensure we have a state row (default MENU).
	st, err := s.states.EnsureDefault(r.Context(), user.ID, domain.StateMenu)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	var sel *string
	if st.SelectedCountry.Valid {
		v := st.SelectedCountry.String
		sel = &v
	}

	now := time.Now().UTC()
	okSub := false
	var until time.Time

	// subscription_ok here means: active vpn subscription for currently selected country (if selected).
	if sel != nil {
		until, okSub, err = s.subs.GetActiveUntilFor(
			r.Context(),
			user.ID,
			"vpn",
			sql.NullString{String: *sel, Valid: true},
			now,
		)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
			return
		}
	}

	var au *time.Time
	if !until.IsZero() {
		u := until
		au = &u
	}

	utils.WriteJSON(w, tgUpsertResp{
		UserID:          user.ID,
		State:           st.State,
		SelectedCountry: sel,
		SubscriptionOK:  okSub,
		ActiveUntil:     au,
	})
}

func toNullString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
