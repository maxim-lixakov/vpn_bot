package menu

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BtnMySubs       = "Моя подписка ℹ️"
	BtnChooseVPN    = "Выбрать страну впн 🇺🇳"
	BtnOrderCountry = "Заказать новую страну ➡️"
	BtnUsePromocode = "Использовать промокод 🎫️"
)

func Keyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnMySubs)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnChooseVPN)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnOrderCountry)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnUsePromocode)),
	)
	kb.ResizeKeyboard = true
	kb.OneTimeKeyboard = false
	kb.Selective = false
	return kb
}
