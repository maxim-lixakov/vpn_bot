package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"vpn-app/internal/utils"
)

type tgCountryToAddReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Text     string `json:"text"`
}

func (s *Server) handleTelegramCountriesToAdd(w http.ResponseWriter, r *http.Request) {
	var req tgCountryToAddReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.TgUserID == 0 || req.Text == "" {
		http.Error(w, "tg_user_id and text are required", http.StatusBadRequest)
		return
	}

	user, ok, err := s.usersRepo.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Находим последнюю подписку kind='country_request' для привязки
	var subscriptionID sql.NullInt64
	subID, found, err := s.subsRepo.GetLatestPaidByKind(r.Context(), user.ID, "country_request")
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if found {
		subscriptionID = sql.NullInt64{Int64: subID, Valid: true}
	}
	// Если подписки нет - subscriptionID останется Invalid (NULL), это допустимо

	if err := s.countriesAddRepo.Insert(r.Context(), user.ID, subscriptionID, req.Text); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	utils.WriteJSON(w, map[string]any{"ok": true})
}
