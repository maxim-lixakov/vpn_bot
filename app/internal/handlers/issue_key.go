package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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

// Единый ответ: либо ключ, либо "нужна оплата".
type issueKeyResp struct {
	Status string `json:"status"` // "ok" | "payment_required"

	Country    string `json:"country"`
	ServerName string `json:"server_name,omitempty"`

	AccessKeyID string `json:"access_key_id,omitempty"`
	AccessURL   string `json:"access_url,omitempty"`

	Payment *paymentHint `json:"payment,omitempty"`
}

type paymentHint struct {
	Kind        string `json:"kind"`         // "vpn"
	CountryCode string `json:"country_code"` // "hk"/"kz"
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
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

	user, ok, err := s.usersRepo.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	server, exists := s.cfg.Servers[req.Country]
	if !exists {
		http.Error(w, "unknown country", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()

	_, subOK, err := s.subsRepo.GetActiveUntilFor(
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
		utils.WriteJSON(w, issueKeyResp{
			Status:     "payment_required",
			Country:    req.Country,
			ServerName: server.Name,
			Payment: &paymentHint{
				Kind:        "vpn",
				CountryCode: req.Country,
			},
		})
		return
	}

	existingKey, hasKey, err := s.keysRepo.GetActive(r.Context(), user.ID, req.Country)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	var keyID string
	var accessURL string
	var accessKeyDBID int64

	if hasKey {
		keyID = existingKey.OutlineKeyID
		accessURL = existingKey.AccessURL
		accessKeyDBID = existingKey.ID
	} else {
		client, okClient := s.clients[req.Country]
		if !okClient {
			log.Printf("ERROR: outline client not configured for country %s, user %d (tg:%d)", req.Country, user.ID, req.TgUserID)
			http.Error(w, "outline client not configured", http.StatusBadGateway)
			return
		}

		keyName := fmt.Sprintf("tg:%d:%s", req.TgUserID, req.Country)
		log.Printf("Creating new Outline access key for user %d (tg:%d) country %s with name %s", user.ID, req.TgUserID, req.Country, keyName)

		key, err := client.CreateAccessKey(r.Context(), keyName)
		if err != nil {
			log.Printf("ERROR: failed to create Outline key for user %d (tg:%d) country %s: %v", user.ID, req.TgUserID, req.Country, err)
			http.Error(w, "outline error: "+err.Error(), http.StatusBadGateway)
			return
		}

		log.Printf("Successfully created Outline key %s for user %d (tg:%d) country %s", key.ID, user.ID, req.TgUserID, req.Country)

		insertedID, err := s.keysRepo.Insert(r.Context(), user.ID, req.Country, key.ID, key.AccessURL)
		if err != nil {
			log.Printf("ERROR: failed to insert access key into DB for user %d (tg:%d): %v", user.ID, req.TgUserID, err)
			http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
			return
		}

		log.Printf("Inserted access key %s into DB with ID %d for user %d (tg:%d)", key.ID, insertedID, user.ID, req.TgUserID)

		keyID = key.ID
		accessURL = key.AccessURL
		accessKeyDBID = insertedID
	}

	if err := s.subsRepo.AttachAccessKeyToLatestPaid(
		r.Context(),
		user.ID,
		"vpn",
		sql.NullString{String: req.Country, Valid: true},
		accessKeyDBID,
	); err != nil {
		log.Printf("ERROR: failed to attach access key %d to subscription for user %d (tg:%d) country %s: %v",
			accessKeyDBID, user.ID, req.TgUserID, req.Country, err)
		// This is critical - subscription exists but key not attached
		// User will get key but subscription might not be properly linked
	}

	if _, err := s.statesRepo.Set(r.Context(), user.ID, domain.StateActive, sql.NullString{String: req.Country, Valid: true}); err != nil {
		log.Printf("WARNING: failed to set user state to Active for user %d (tg:%d): %v",
			user.ID, req.TgUserID, err)
		// Non-critical, just log it
	}

	utils.WriteJSON(w, issueKeyResp{
		Status:      "ok",
		Country:     req.Country,
		ServerName:  server.Name,
		AccessKeyID: keyID,
		AccessURL:   accessURL,
	})
}
