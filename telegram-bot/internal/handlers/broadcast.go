package handlers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/router"
)

type Broadcast struct{}

func (h Broadcast) Name() string { return "broadcast" }

func (h Broadcast) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.Message == nil {
		return false
	}
	text := strings.TrimSpace(u.Message.Text)
	return strings.HasPrefix(text, "/broadcast_all") ||
		strings.HasPrefix(text, "/broadcast_with_subscription") ||
		strings.HasPrefix(text, "/broadcast_without_subscription")
}

func (h Broadcast) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	// Проверяем, что это админ
	adminTgUserIDStr := os.Getenv("BACKUP_ADMIN_TG_USER_ID")
	if adminTgUserIDStr == "" {
		msg := tgbotapi.NewMessage(s.ChatID, "Админ не настроен")
		_, _ = d.Bot.Send(msg)
		return nil
	}

	adminTgUserID, err := strconv.ParseInt(adminTgUserIDStr, 10, 64)
	if err != nil || s.TgUserID != adminTgUserID {
		msg := tgbotapi.NewMessage(s.ChatID, "У вас нет прав для выполнения этой команды")
		_, _ = d.Bot.Send(msg)
		return nil
	}

	text := strings.TrimSpace(u.Message.Text)
	var target string
	var command string

	if strings.HasPrefix(text, "/broadcast_all") {
		target = "all"
		command = "/broadcast_all"
	} else if strings.HasPrefix(text, "/broadcast_with_subscription") {
		target = "with_subscription"
		command = "/broadcast_with_subscription"
	} else if strings.HasPrefix(text, "/broadcast_without_subscription") {
		target = "without_subscription"
		command = "/broadcast_without_subscription"
	} else {
		return nil
	}

	// Извлекаем сообщение после команды
	message := strings.TrimSpace(strings.TrimPrefix(text, command))
	if message == "" {
		msg := tgbotapi.NewMessage(s.ChatID, "Пожалуйста, укажите сообщение для рассылки после команды.\n\nПример:\n"+command+" Ваше сообщение здесь")
		_, _ = d.Bot.Send(msg)
		return nil
	}

	// Отправляем запрос на рассылку
	resp, err := d.App.TelegramBroadcast(ctx, appclient.TelegramBroadcastReq{
		AdminTgUserID: s.TgUserID,
		Message:       message,
		Target:        target,
	})
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Ошибка при рассылке: "+err.Error())
		_, _ = d.Bot.Send(msg)
		return nil
	}

	// Отправляем подтверждение админу
	confirmMsg := tgbotapi.NewMessage(s.ChatID, fmt.Sprintf("Рассылка запущена. Отправлено: %d сообщений", resp.SentCount))
	_, _ = d.Bot.Send(confirmMsg)

	return nil
}
