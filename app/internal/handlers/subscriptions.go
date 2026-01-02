package handlers

import (
	"log"
	"net/http"
	"time"

	"vpn-app/internal/repo"
	"vpn-app/internal/utils"
)

type tgSubscriptionsResp struct {
	Items []tgSubscriptionDTO `json:"items"`
}

type tgSubscriptionDTO struct {
	Kind         string     `json:"kind"`
	CountryCode  *string    `json:"country_code"`
	PaidAt       time.Time  `json:"paid_at"`
	ActiveUntil  *time.Time `json:"active_until"`
	IsActive     bool       `json:"is_active"`
	TrafficBytes *int64     `json:"traffic_bytes,omitempty"` // Потребленный трафик в байтах
}

func (s *Server) handleTelegramSubscriptions(w http.ResponseWriter, r *http.Request) {
	tgUserID, err := utils.ParseInt64Query(r, "tg_user_id")
	if err != nil {
		http.Error(w, "bad tg_user_id", http.StatusBadRequest)
		return
	}

	user, ok, err := s.usersRepo.GetByTelegramID(r.Context(), tgUserID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		utils.WriteJSON(w, tgSubscriptionsResp{Items: nil})
		return
	}

	items, err := s.subsRepo.ListByUser(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	now := time.Now().UTC()

	// Получаем все активные ключи пользователя для получения трафика
	activeKeys, err := s.keysRepo.GetAllActiveByUser(r.Context(), user.ID)
	if err != nil {
		log.Printf("failed to get active keys for user %d: %v", user.ID, err)
		// Продолжаем без трафика, если не удалось получить ключи
		activeKeys = []repo.AccessKey{}
	}

	// Создаем мапу для быстрого поиска трафика по country_code
	trafficByCountry := make(map[string]int64)
	for _, key := range activeKeys {
		countryCode := key.Country
		if countryCode == "" {
			continue
		}

		// Получаем Outline клиент для этой страны
		client, ok := s.clients[countryCode]
		if !ok {
			continue
		}

		// Получаем метрики трафика
		metrics, err := client.MetricsTransfer(r.Context())
		if err != nil {
			log.Printf("failed to get metrics for country %s: %v", countryCode, err)
			continue
		}

		// Ищем трафик для этого ключа
		if bytes, ok := metrics[key.OutlineKeyID]; ok {
			trafficByCountry[countryCode] = bytes
		}
	}

	out := make([]tgSubscriptionDTO, 0, len(items))
	for _, it := range items {
		var cc *string
		if it.CountryCode.Valid {
			v := it.CountryCode.String
			cc = &v
		}
		u := it.ActiveUntil
		isActive := it.Status == "paid" && u.After(now) && it.Kind == "vpn"

		// Получаем трафик для этой подписки
		var trafficBytes *int64
		if cc != nil && isActive {
			if traffic, ok := trafficByCountry[*cc]; ok {
				trafficBytes = &traffic
			}
		}

		// active_until всегда есть в схеме, но чтобы интерфейс был удобный — отдадим pointer
		out = append(out, tgSubscriptionDTO{
			Kind:         it.Kind,
			CountryCode:  cc,
			PaidAt:       it.PaidAt,
			ActiveUntil:  &u,
			IsActive:     isActive,
			TrafficBytes: trafficBytes,
		})
	}

	utils.WriteJSON(w, tgSubscriptionsResp{Items: out})
}
