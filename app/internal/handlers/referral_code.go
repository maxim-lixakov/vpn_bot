package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"vpn-app/internal/utils"
)

type tgReferralCodeReq struct {
	TgUserID int64 `json:"tg_user_id"`
}

type tgReferralCodeResp struct {
	Promocode string `json:"promocode,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (s *Server) handleTelegramReferralCode(w http.ResponseWriter, r *http.Request) {
	var req tgReferralCodeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
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

	// Проверяем, есть ли у пользователя активная подписка
	hasActive, err := s.subsRepo.HasAnyActiveSubscription(r.Context(), user.ID, "vpn", time.Now().UTC())
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !hasActive {
		utils.WriteJSON(w, tgReferralCodeResp{
			Error: "Чтобы получить реферальный промокод, сначала нужно купить подписку.",
		})
		return
	}

	// Получаем или создаём реферальный промокод
	promo, err := s.promocodesRepo.GetOrCreateReferralCode(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	utils.WriteJSON(w, tgReferralCodeResp{
		Promocode: promo.PromocodeName,
	})
}
