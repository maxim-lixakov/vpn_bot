package handlers

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
	"vpn-bot/internal/utils"
)

type GetReferralCode struct{}

func (h GetReferralCode) Name() string { return "get_referral_code" }

func (h GetReferralCode) CanHandle(u tgbotapi.Update, s router.Session) bool {
	if u.Message == nil {
		return false
	}
	return utils.NormalizeButtonText(strings.TrimSpace(u.Message.Text)) == utils.NormalizeButtonText(menu.BtnReferralCode)
}

func (h GetReferralCode) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	resp, err := d.App.TelegramReferralCode(ctx, s.TgUserID)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не смог получить реферальный код: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	// Если есть ошибка (нет активной подписки)
	if resp.Error != "" {
		msg := tgbotapi.NewMessage(s.ChatID, resp.Error)
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	// Информационное сообщение о реферальной программе
	infoText := `🎁 Ваш реферальный код:
` + fmt.Sprintf("`%s`", resp.Promocode) + `

📋 Как работает реферальная программа:

• Этот промокод предназначен только для новых пользователей

• При использовании вашего промокода новый пользователь получит 1 бесплатный месяц подписки

• За каждого пользователя, который использует ваш реферальный код, вам будет добавлен +1 месяц к текущей активной
подписке. Вы получите уведомление в боте каждый раз, когда кто-то использует ваш промокод`

	msg := tgbotapi.NewMessage(s.ChatID, infoText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = menu.Keyboard()
	_, _ = d.Bot.Send(msg)
	return nil
}
