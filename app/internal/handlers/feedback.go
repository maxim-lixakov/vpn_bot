package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"vpn-app/internal/utils"
)

type tgFeedbackReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Text     string `json:"text"`
}

func (s *Server) handleTelegramFeedback(w http.ResponseWriter, r *http.Request) {
	var req tgFeedbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
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

	// Сохраняем feedback
	if err := s.feedbackRepo.Insert(r.Context(), user.ID, req.Text); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	utils.WriteJSON(w, map[string]any{"ok": true})
}
