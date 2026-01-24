package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"vpn-app/internal/utils"
)

type validateRenewalReq struct {
	SubscriptionID int64 `json:"subscription_id"`
}

type validateRenewalResp struct {
	Valid        bool   `json:"valid"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// handleValidateRenewal checks if a subscription renewal is valid
// A renewal is invalid if the subscription's access key has been revoked
func (s *Server) handleValidateRenewal(w http.ResponseWriter, r *http.Request) {
	var req validateRenewalReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Try to parse subscription_id from query string for GET requests
		subIDStr := r.URL.Query().Get("subscription_id")
		if subIDStr == "" {
			http.Error(w, "subscription_id is required", http.StatusBadRequest)
			return
		}
		var err error
		req.SubscriptionID, err = strconv.ParseInt(subIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid subscription_id", http.StatusBadRequest)
			return
		}
	}

	if req.SubscriptionID == 0 {
		http.Error(w, "subscription_id is required", http.StatusBadRequest)
		return
	}

	// Get subscription
	sub, found, err := s.subsRepo.GetByID(r.Context(), req.SubscriptionID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !found {
		utils.WriteJSON(w, validateRenewalResp{
			Valid:        false,
			ErrorMessage: "Подписка не найдена. Пожалуйста, выберите страну заново.",
		})
		return
	}

	// Check if subscription has an access key
	if !sub.AccessKeyID.Valid {
		// No access key linked - this is OK for renewal (key might not have been issued yet)
		utils.WriteJSON(w, validateRenewalResp{Valid: true})
		return
	}

	// Get the access key and check if it's revoked
	countryCode := ""
	if sub.CountryCode.Valid {
		countryCode = strings.TrimSpace(strings.ToLower(sub.CountryCode.String))
	}

	if countryCode == "" {
		utils.WriteJSON(w, validateRenewalResp{
			Valid:        false,
			ErrorMessage: "Страна не указана. Пожалуйста, выберите страну заново.",
		})
		return
	}

	// Check if user has an active (non-revoked) key for this country
	key, hasActiveKey, err := s.keysRepo.GetActive(r.Context(), sub.UserID, countryCode)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// If there's no active key, the renewal is invalid
	if !hasActiveKey {
		utils.WriteJSON(w, validateRenewalResp{
			Valid:        false,
			ErrorMessage: "Ваш ключ был отозван. Пожалуйста, выберите страну заново через меню бота.",
		})
		return
	}

	// Check if the active key matches the subscription's access key
	if key.ID != sub.AccessKeyID.Int64 {
		// Different key - the old one was revoked, user has a new one
		utils.WriteJSON(w, validateRenewalResp{
			Valid:        false,
			ErrorMessage: "Ваш ключ был изменён. Пожалуйста, выберите страну заново через меню бота.",
		})
		return
	}

	// All checks passed - renewal is valid
	utils.WriteJSON(w, validateRenewalResp{Valid: true})
}
