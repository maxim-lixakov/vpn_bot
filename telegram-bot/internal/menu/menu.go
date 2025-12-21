package menu

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BtnMySubs       = "моя подписка ℹ️"
	BtnChooseVPN    = "выбрать страну впн 📰"
	BtnOrderCountry = "заказать новую страну 🔎"
	BtnMenu         = "меню 📱"
)

func Keyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnMySubs)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnChooseVPN)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(BtnOrderCountry)),
	)
	kb.ResizeKeyboard = true
	kb.OneTimeKeyboard = false
	kb.Selective = false
	return kb
}

func IsMenuText(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == BtnMenu || s == "/menu"
}
