package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"vpn-app/internal/utils"
)

type tgPromocodeUseReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Code     string `json:"code"`
}

type tgPromocodeUseResp struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	Months  int    `json:"months,omitempty"`
}

func (s *Server) handleTelegramPromocodeUse(w http.ResponseWriter, r *http.Request) {
	var req tgPromocodeUseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		utils.WriteJSON(w, tgPromocodeUseResp{
			Valid:   false,
			Message: "Промокод не может быть пустым",
		})
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

	// Получаем промокод
	promo, found, err := s.promocodesRepo.GetByName(r.Context(), req.Code)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !found {
		utils.WriteJSON(w, tgPromocodeUseResp{
			Valid:   false,
			Message: "Промокод не найден",
		})
		return
	}

	// Проверяем, не является ли промокод созданным самим пользователем
	if promo.PromotedBy.Valid && promo.PromotedBy.Int64 == user.ID {
		utils.WriteJSON(w, tgPromocodeUseResp{
			Valid:   false,
			Message: "Вы не можете использовать промокод, созданный вами",
		})
		return
	}

	// Проверяем, не использовал ли уже этот пользователь этот промокод
	alreadyUsed, err := s.promocodeUsagesRepo.HasUserUsed(r.Context(), promo.ID, user.ID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if alreadyUsed {
		utils.WriteJSON(w, tgPromocodeUseResp{
			Valid:   false,
			Message: "Вы уже использовали этот промокод",
		})
		return
	}

	// Проверяем лимит использований
	if promo.TimesToBeUsed > 0 && promo.TimesUsed >= promo.TimesToBeUsed {
		utils.WriteJSON(w, tgPromocodeUseResp{
			Valid:   false,
			Message: "Промокод использован максимальное количество раз",
		})
		return
	}

	// Проверяем, был ли у пользователя когда-либо подписка (старый пользователь)
	hasEverHadSubscription, err := s.subsRepo.HasEverHadSubscription(r.Context(), user.ID, "vpn")
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Если пользователь старый (имел подписку когда-либо) и промокод не разрешает для старых пользователей
	if hasEverHadSubscription && !promo.AllowForOldUsers {
		utils.WriteJSON(w, tgPromocodeUseResp{
			Valid:   false,
			Message: "Этот промокод работает только для новых пользователей.",
		})
		return
	}

	// Валидация прошла - увеличиваем счётчик использований
	if err := s.promocodesRepo.IncrementUsage(r.Context(), promo.ID); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Создаём запись об использовании
	if err := s.promocodeUsagesRepo.Insert(r.Context(), promo.ID, user.ID); err != nil {
		// Если не удалось создать запись - откатываем инкремент
		_ = s.promocodesRepo.DecrementUsage(r.Context(), promo.ID)
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Проверяем, является ли промокод реферальным
	// Выполняем это асинхронно, чтобы не блокировать ответ
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if strings.HasPrefix(strings.ToLower(promo.PromocodeName), "referral_") && promo.PromotedBy.Valid {
			referrerUserID := promo.PromotedBy.Int64
			// Не даём бонус, если пользователь использует свой собственный промокод
			if referrerUserID != user.ID {
				// Продлеваем активную подписку реферера на +1 месяц
				oldUntil, newUntil, err := s.subsRepo.ExtendActiveSubscriptionByMonth(ctx, referrerUserID, "vpn")
				if err != nil {
					// Логируем ошибку, но не прерываем процесс применения промокода
					// TODO: добавить логирование
				} else {
					// Получаем информацию о пользователе, который использовал промокод, для уведомления
					username := "пользователь"
					if user.Username.Valid && user.Username.String != "" {
						username = "@" + user.Username.String
					} else if user.FirstName.Valid && user.FirstName.String != "" {
						username = user.FirstName.String
					}

					// Получаем tg_user_id реферера для отправки уведомления
					referrerUser, ok, err := s.usersRepo.GetByID(ctx, referrerUserID)
					if err == nil && ok && s.cfg.BotToken != "" {
						// Отправляем уведомление рефереру с информацией о продлении
						message := fmt.Sprintf(
							"Пользователь %s использовал ваш реферальный промокод.\n\nВам добавлен +1 месяц к активной подписке!\n\nБыло активно до: %s\nСтало активно до: %s",
							username,
							oldUntil.Format("2006-01-02 15:04"),
							newUntil.Format("2006-01-02 15:04"),
						)
						_ = utils.SendTelegramMessage(s.cfg.BotToken, referrerUser.TgUserID, message)
					}
				}
			}
		}
	}()

	utils.WriteJSON(w, tgPromocodeUseResp{
		Valid:   true,
		Message: "Промокод успешно применён",
		Months:  promo.PromocodeMonths,
	})
}

type tgPromocodeRollbackReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Code     string `json:"code,omitempty"` // Опционально: если не указан, откатываем последний использованный
}

func (s *Server) handleTelegramPromocodeRollback(w http.ResponseWriter, r *http.Request) {
	var req tgPromocodeRollbackReq
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

	var promocodeID int64
	req.Code = strings.TrimSpace(req.Code)

	if req.Code != "" {
		// Получаем промокод по имени
		promo, found, err := s.promocodesRepo.GetByName(r.Context(), req.Code)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
			return
		}
		if !found {
			http.Error(w, "promocode not found", http.StatusNotFound)
			return
		}
		promocodeID = promo.ID
	} else {
		// Код не указан - находим последний использованный промокод
		var found bool
		promocodeID, found, err = s.promocodeUsagesRepo.GetLastUsedPromocodeID(r.Context(), user.ID)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
			return
		}
		if !found {
			utils.WriteJSON(w, map[string]any{"ok": true, "message": "no promocode to rollback"})
			return
		}
	}

	// Откатываем инкремент использования
	if err := s.promocodesRepo.DecrementUsage(r.Context(), promocodeID); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Удаляем запись об использовании (если есть)
	_ = s.promocodeUsagesRepo.Delete(r.Context(), promocodeID, user.ID)

	// Удаляем подписку, созданную промокодом (если есть)
	_ = s.subsRepo.DeletePromocodeSubscription(r.Context(), user.ID)

	utils.WriteJSON(w, map[string]any{"ok": true})
}
