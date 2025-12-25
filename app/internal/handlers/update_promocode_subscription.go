package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"vpn-app/internal/utils"
)

type tgUpdatePromocodeSubscriptionReq struct {
	TgUserID    int64  `json:"tg_user_id"`
	CountryCode string `json:"country_code"`
}

func (s *Server) handleTelegramUpdatePromocodeSubscription(w http.ResponseWriter, r *http.Request) {
	var req tgUpdatePromocodeSubscriptionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.CountryCode = strings.TrimSpace(strings.ToLower(req.CountryCode))
	if req.CountryCode == "" {
		http.Error(w, "country_code is required", http.StatusBadRequest)
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

	// Обновляем country_code в последней подписке от промокода
	if err := s.subsRepo.UpdateCountryCodeForPromocode(r.Context(), user.ID, req.CountryCode); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	utils.WriteJSON(w, map[string]any{"ok": true})
}
