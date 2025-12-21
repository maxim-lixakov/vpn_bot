package handlers

import (
	"net/http"
	"time"

	"vpn-app/internal/utils"
)

type tgSubscriptionsResp struct {
	Items []tgSubscriptionDTO `json:"items"`
}

type tgSubscriptionDTO struct {
	Kind        string     `json:"kind"`
	CountryCode *string    `json:"country_code"`
	PaidAt      time.Time  `json:"paid_at"`
	ActiveUntil *time.Time `json:"active_until"`
	IsActive    bool       `json:"is_active"`
}

func (s *Server) handleTelegramSubscriptions(w http.ResponseWriter, r *http.Request) {
	tgUserID, err := utils.ParseInt64Query(r, "tg_user_id")
	if err != nil {
		http.Error(w, "bad tg_user_id", http.StatusBadRequest)
		return
	}

	user, ok, err := s.users.GetByTelegramID(r.Context(), tgUserID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		utils.WriteJSON(w, tgSubscriptionsResp{Items: nil})
		return
	}

	items, err := s.subs.ListByUser(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	now := time.Now().UTC()
	out := make([]tgSubscriptionDTO, 0, len(items))
	for _, it := range items {
		var cc *string
		if it.CountryCode.Valid {
			v := it.CountryCode.String
			cc = &v
		}
		u := it.ActiveUntil
		isActive := it.Status == "paid" && u.After(now) && it.Kind == "vpn"

		// active_until всегда есть в схеме, но чтобы интерфейс был удобный — отдадим pointer
		out = append(out, tgSubscriptionDTO{
			Kind:        it.Kind,
			CountryCode: cc,
			PaidAt:      it.PaidAt,
			ActiveUntil: &u,
			IsActive:    isActive,
		})
	}

	utils.WriteJSON(w, tgSubscriptionsResp{Items: out})
}
