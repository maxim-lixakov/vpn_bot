package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"vpn-app/internal/utils"
)

type tgCountryStatusResp struct {
	Active      bool       `json:"active"`
	ActiveUntil *time.Time `json:"active_until"`
}

func (s *Server) handleTelegramCountryStatus(w http.ResponseWriter, r *http.Request) {
	tgUserID, err := utils.ParseInt64Query(r, "tg_user_id")
	if err != nil {
		http.Error(w, "bad tg_user_id", http.StatusBadRequest)
		return
	}
	country := r.URL.Query().Get("country")
	if country == "" {
		http.Error(w, "country is required", http.StatusBadRequest)
		return
	}

	user, ok, err := s.users.GetByTelegramID(r.Context(), tgUserID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		utils.WriteJSON(w, tgCountryStatusResp{Active: false, ActiveUntil: nil})
		return
	}

	until, active, err := s.subs.GetActiveUntilFor(
		r.Context(),
		user.ID,
		"vpn",
		sql.NullString{String: country, Valid: true},
		time.Now().UTC(),
	)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	var p *time.Time
	if !until.IsZero() {
		u := until
		p = &u
	}

	utils.WriteJSON(w, tgCountryStatusResp{Active: active, ActiveUntil: p})
}
