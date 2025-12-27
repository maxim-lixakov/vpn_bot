package menu

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BtnMySubs       = "ℹ️ Моя подписка"
	BtnChooseVPN    = "🇺🇳 Выбрать страну впн"
	BtnOrderCountry = "➡️ Заказать новую страну "
	BtnUsePromocode = "🎫️ Использовать промокод"
	BtnReferralCode = "🎁 Получить код для реферальной программы"
	BtnFeedback     = "💬 Оставить отзыв"
)

func Keyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnMySubs)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnChooseVPN)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnOrderCountry)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnUsePromocode)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnFeedback)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnReferralCode)),
	)
	kb.ResizeKeyboard = true
	kb.OneTimeKeyboard = false
	kb.Selective = false
	return kb
}
