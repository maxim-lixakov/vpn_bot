package flows

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func CountryKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇰🇿 Kazakhstan", "country:kz"),
			tgbotapi.NewInlineKeyboardButtonData("🇭🇰 Hong Kong", "country:hk"),
		),
	)
}

func Instructions(serverName, accessURL string) string {
	// сухая инструкция
	return fmt.Sprintf(
		"Сервер: %s\n\n"+
			"1) Установи Outline Client (ссылки ниже)\n"+
			"2) Открой Outline Client → нажми “+” / Add server\n"+
			"3) Вставь ключ доступа и добавь сервер\n"+
			"4) Нажми Connect\n\n"+
			"Ключ доступа:\n%s\n",
		serverName, accessURL,
	)
}

func OfficialDownloadLinksText() string {
	// официальные источники, см. страницу Google Developers :contentReference[oaicite:9]{index=9}
	return "Официальные ссылки на Outline:\n" +
		"Client (для пользователя):\n" +
		"- Windows: https://s3.amazonaws.com/outline-releases/client/windows/stable/Outline-Client.exe\n" +
		"- iOS:     https://itunes.apple.com/us/app/outline-app/id1356177741\n" +
		"- Android: https://play.google.com/store/apps/details?id=org.outline.android.client\n" +
		"- Android (APK): https://s3.amazonaws.com/outline-releases/client/android/stable/Outline-Client.apk\n"
}
