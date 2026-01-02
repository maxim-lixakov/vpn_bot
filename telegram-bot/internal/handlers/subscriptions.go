package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
	"vpn-bot/internal/utils"
)

type MySubscriptions struct{}

func (h MySubscriptions) Name() string { return "my_subs" }

func (h MySubscriptions) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.Message == nil {
		return false
	}
	text := strings.TrimSpace(u.Message.Text)
	return utils.NormalizeButtonText(text) == utils.NormalizeButtonText(menu.BtnMySubs)
}

func (h MySubscriptions) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	resp, err := d.App.TelegramSubscriptions(ctx, s.TgUserID)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог получить подписки: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	if len(resp.Items) == 0 {
		msg := tgbotapi.NewMessage(s.ChatID, "Подписок нет.")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	// маппинг country_code -> красивое имя сервера
	// (подправь под свои коды/названия)
	serverNameByCode := map[string]string{
		"kz": "Kazakhstan",
		"hk": "HongKong",
	}

	now := time.Now()
	lines := make([]string, 0, len(resp.Items))

	for _, it := range resp.Items {
		// показываем только vpn
		if it.Kind != "vpn" {
			continue
		}
		// и только активные
		if !it.IsActive {
			continue
		}
		if it.ActiveUntil != nil && !it.ActiveUntil.After(now) {
			continue
		}

		code := ""
		if it.CountryCode != nil {
			code = strings.ToLower(strings.TrimSpace(*it.CountryCode))
		}

		serverName := code
		if code == "" {
			serverName = "—"
		} else if v, ok := serverNameByCode[code]; ok {
			serverName = v
		} else {
			// если нет в маппинге — покажем код как есть
			serverName = strings.ToUpper(code)
		}

		until := "—"
		if it.ActiveUntil != nil {
			until = it.ActiveUntil.Format("2006-01-02 15:04")
		}

		// Форматируем трафик
		trafficStr := "—"
		if it.TrafficBytes != nil {
			trafficStr = utils.FormatBytes(*it.TrafficBytes)
		}

		line := fmt.Sprintf("Сервер: %s — активна до *%s*\nТрафик: *%s*",
			utils.Mdv2Escape(serverName),
			utils.Mdv2Escape(until),
			utils.Mdv2Escape(trafficStr))
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		msg := tgbotapi.NewMessage(s.ChatID, "Активных VPN-подписок нет.")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	text := strings.Join(lines, "\n")

	msg := tgbotapi.NewMessage(s.ChatID, text)
	msg.ParseMode = "MarkdownV2"
	msg.ReplyMarkup = menu.Keyboard()
	_, _ = d.Bot.Send(msg)
	return nil
}
