package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"vpn-app/internal/utils"
)

type tgSetStateReq struct {
	TgUserID        int64   `json:"tg_user_id"`
	State           string  `json:"router"`
	SelectedCountry *string `json:"selected_country"`
}

type tgSetStateResp struct {
	State           string  `json:"router"`
	SelectedCountry *string `json:"selected_country"`
}

func (s *Server) handleTelegramSetState(w http.ResponseWriter, r *http.Request) {
	var req tgSetStateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 || req.State == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, ok, err := s.users.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil || !ok {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	var sel sql.NullString
	if req.SelectedCountry != nil {
		sel = sql.NullString{String: *req.SelectedCountry, Valid: true}
	}

	st, err := s.states.Set(r.Context(), user.ID, req.State, sel)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	var outSel *string
	if st.SelectedCountry.Valid {
		outSel = &st.SelectedCountry.String
	}

	utils.WriteJSON(w, tgSetStateResp{State: st.State, SelectedCountry: outSel})
}
