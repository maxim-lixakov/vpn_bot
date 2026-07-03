package countries

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func CountryKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇰🇿 Kazakhstan", "country:kz"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Меню", "menu"),
		),
	)
}
