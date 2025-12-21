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
	UserID int64 `json:"user_id"`

	State string `json:"state"`

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

	u := repo.User{TgUserID: req.TgUserID}

	if req.Username != nil && *req.Username != "" {
		u.Username = sql.NullString{String: *req.Username, Valid: true}
	}
	if req.FirstName != nil && *req.FirstName != "" {
		u.FirstName = sql.NullString{String: *req.FirstName, Valid: true}
	}
	if req.LastName != nil && *req.LastName != "" {
		u.LastName = sql.NullString{String: *req.LastName, Valid: true}
	}
	if req.LanguageCode != nil && *req.LanguageCode != "" {
		u.LanguageCode = sql.NullString{String: *req.LanguageCode, Valid: true}
	}
	if req.Phone != nil && *req.Phone != "" {
		u.Phone = sql.NullString{String: *req.Phone, Valid: true}
	}

	user, err := s.usersRepo.UpsertByTelegram(r.Context(), u)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	st, err := s.statesRepo.EnsureDefault(r.Context(), user.ID, domain.StateMenu)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// гарантия
	if st.State == "" {
		st, _ = s.statesRepo.Set(r.Context(), user.ID, domain.StateMenu, st.SelectedCountry)
	}

	var sel *string
	if st.SelectedCountry.Valid {
		v := st.SelectedCountry.String
		sel = &v
	}

	// subscription_ok только если selected_country выбран
	now := time.Now().UTC()
	subOK := false
	var until time.Time

	if sel != nil {
		until, subOK, err = s.subsRepo.GetActiveUntilFor(
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
		SubscriptionOK:  subOK,
		ActiveUntil:     au,
	})
}
