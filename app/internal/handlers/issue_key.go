package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"vpn-app/internal/domain"
	"vpn-app/internal/utils"
)

type issueKeyReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Country  string `json:"country"`
}

type issueKeyResp struct {
	ServerName  string `json:"server_name"`
	Country     string `json:"country"`
	AccessKeyID string `json:"access_key_id"`
	AccessURL   string `json:"access_url"`
}

func (s *Server) handleIssueKey(w http.ResponseWriter, r *http.Request) {
	var req issueKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Country = strings.TrimSpace(req.Country)
	if req.TgUserID == 0 || req.Country == "" {
		http.Error(w, "tg_user_id and country are required", http.StatusBadRequest)
		return
	}

	user, ok, err := s.users.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC()

	// Require active vpn subscription for the requested country.
	_, subOK, err := s.subs.GetActiveUntilFor(
		r.Context(),
		user.ID,
		"vpn",
		sql.NullString{String: req.Country, Valid: true},
		now,
	)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !subOK {
		http.Error(w, "subscription inactive", http.StatusPaymentRequired)
		return
	}

	server, exists := s.cfg.Servers[req.Country]
	if !exists {
		http.Error(w, "unknown country", http.StatusBadRequest)
		return
	}

	client, okClient := s.clients[req.Country]
	if !okClient {
		http.Error(w, "outline client not configured", http.StatusBadGateway)
		return
	}

	// Create key in Outline manager.
	keyName := fmt.Sprintf("tg:%d:%s", req.TgUserID, req.Country)
	key, err := client.CreateAccessKey(r.Context(), keyName)
	if err != nil {
		http.Error(w, "outline error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Store mapping for future deactivation.
	if err := s.keys.Insert(r.Context(), user.ID, req.Country, key.ID, key.AccessURL); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	_, _ = s.states.Set(r.Context(), user.ID, domain.StateActive, sql.NullString{String: req.Country, Valid: true})

	utils.WriteJSON(w, issueKeyResp{
		ServerName:  server.Name,
		Country:     req.Country,
		AccessKeyID: key.ID,
		AccessURL:   key.AccessURL,
	})
}
