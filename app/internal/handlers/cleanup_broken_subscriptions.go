package handlers

import (
	"log"
	"net/http"
	"time"

	"vpn-app/internal/utils"
)

type cleanupBrokenSubscriptionsResp struct {
	TotalFound   int                      `json:"total_found"`
	Cleaned      int                      `json:"cleaned"`
	Failed       int                      `json:"failed"`
	Subscription []brokenSubscriptionInfo `json:"subscriptions"`
}

type brokenSubscriptionInfo struct {
	SubscriptionID int64     `json:"subscription_id"`
	UserID         int64     `json:"user_id"`
	CountryCode    string    `json:"country_code,omitempty"`
	PaidAt         time.Time `json:"paid_at"`
	ActiveUntil    time.Time `json:"active_until"`
	Action         string    `json:"action"` // "deleted", "failed"
	Error          string    `json:"error,omitempty"`
	IsPromocode    bool      `json:"is_promocode"`
}

// handleCleanupBrokenSubscriptions finds and cleans up active subscriptions with NULL access_key_id
// These are subscriptions where key creation or attachment failed, leaving the subscription in an inconsistent state
func (s *Server) handleCleanupBrokenSubscriptions(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	// Find all active subscriptions without access keys
	subs, err := s.subsRepo.GetActiveSubscriptionsWithoutAccessKey(r.Context(), now)
	if err != nil {
		log.Printf("ERROR: failed to get broken subscriptions: %v", err)
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	resp := cleanupBrokenSubscriptionsResp{
		TotalFound:   len(subs),
		Subscription: make([]brokenSubscriptionInfo, 0, len(subs)),
	}

	log.Printf("Found %d active subscriptions with NULL access_key_id", len(subs))

	for _, sub := range subs {
		info := brokenSubscriptionInfo{
			SubscriptionID: sub.ID,
			UserID:         sub.UserID,
			PaidAt:         sub.PaidAt,
			ActiveUntil:    sub.ActiveUntil,
		}

		if sub.CountryCode.Valid {
			info.CountryCode = sub.CountryCode.String
		}

		// Check if this is a promocode subscription
		isPromocode := sub.ProviderPaymentChargeID.Valid && sub.ProviderPaymentChargeID.String == "promocode"
		info.IsPromocode = isPromocode

		// Try to delete the subscription
		if err := s.subsRepo.DeleteSubscription(r.Context(), sub.ID); err != nil {
			log.Printf("ERROR: failed to delete broken subscription %d for user %d: %v", sub.ID, sub.UserID, err)
			info.Action = "failed"
			info.Error = err.Error()
			resp.Failed++
		} else {
			log.Printf("Deleted broken subscription %d for user %d (paid_at=%s, active_until=%s, country=%s, is_promocode=%v)",
				sub.ID, sub.UserID, sub.PaidAt.Format("2006-01-02 15:04"), sub.ActiveUntil.Format("2006-01-02 15:04"), info.CountryCode, isPromocode)
			info.Action = "deleted"
			resp.Cleaned++

			// If it was a promocode subscription, also rollback the promocode usage
			if isPromocode {
				// Try to find and rollback promocode usage
				promocodeID, found, err := s.promocodeUsagesRepo.GetLastUsedPromocodeID(r.Context(), sub.UserID)
				if err == nil && found {
					// Decrement usage count
					if err := s.promocodesRepo.DecrementUsage(r.Context(), promocodeID); err != nil {
						log.Printf("WARNING: failed to decrement promocode usage for promocode %d after cleaning subscription %d: %v", promocodeID, sub.ID, err)
					} else {
						// Delete usage record
						if err := s.promocodeUsagesRepo.Delete(r.Context(), promocodeID, sub.UserID); err != nil {
							log.Printf("WARNING: failed to delete promocode usage record for promocode %d user %d: %v", promocodeID, sub.UserID, err)
						} else {
							log.Printf("Rolled back promocode usage for promocode %d after cleaning subscription %d", promocodeID, sub.ID)
						}
					}
				}
			}
		}

		resp.Subscription = append(resp.Subscription, info)
	}

	log.Printf("Cleanup completed: found=%d, cleaned=%d, failed=%d", resp.TotalFound, resp.Cleaned, resp.Failed)
	utils.WriteJSON(w, resp)
}
