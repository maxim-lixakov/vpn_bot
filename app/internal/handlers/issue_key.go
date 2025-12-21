package handlers

import (
	"strings"
	"time"

	"database/sql"
	"encoding/json"
	"net/http"

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 || req.Country == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Country = strings.ToLower(req.Country)

	server, ok := s.cfg.Servers[req.Country]
	if !ok {
		http.Error(w, "unknown country", http.StatusBadRequest)
		return
	}

	user, okU, err := s.users.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil || !okU {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	until, subOK, err := s.subs.GetActiveUntilFor(
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

	// if already have active key
	if k, ok, err := s.keys.GetActive(r.Context(), user.ID, req.Country); err == nil && ok {
		_ = func() error {
			_, err := s.states.Set(r.Context(), user.ID, domain.StateActive, sql.NullString{String: req.Country, Valid: true})
			return err
		}()
		utils.WriteJSON(w, issueKeyResp{
			ServerName:  server.Name,
			Country:     req.Country,
			AccessKeyID: k.OutlineKeyID,
			AccessURL:   k.AccessURL,
		})
		return
	}

	client := s.clients[req.Country]
	key, err := client.CreateAccessKey(r.Context(), "tg:"+utils.Itoa(req.TgUserID))
	if err != nil {
		http.Error(w, "outline create key failed: "+err.Error(), http.StatusBadGateway)
		return
	}

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
