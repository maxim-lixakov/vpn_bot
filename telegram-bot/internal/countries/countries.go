package countries

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func CountryKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇰🇿 Kazakhstan", "country:kz"),
			tgbotapi.NewInlineKeyboardButtonData("🇭🇰 Hong Kong", "country:hk"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇺🇸 USA", "country:us"),
			tgbotapi.NewInlineKeyboardButtonData("🇩🇪 Germany", "country:de"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇷🇺 Russia", "country:rus"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Меню", "menu"),
		),
	)
}
