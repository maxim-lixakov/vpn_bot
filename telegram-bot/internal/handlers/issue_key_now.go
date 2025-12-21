package handlers

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

// issueKeyNow issues Outline key via app and sends instructions to the user.
// It expects s.SelectedCountry != nil.
func issueKeyNow(ctx context.Context, s router.Session, d router.Deps) error {
	if s.SelectedCountry == nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Не выбрана страна. Нажми /start")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	resp, err := d.App.IssueKey(ctx, s.TgUserID, *s.SelectedCountry)
	if err != nil {
		msg := tgbotapi.NewMessage(s.ChatID, "Ошибка выдачи ключа: "+err.Error())
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	text := fmt.Sprintf(
		"Сервер: %s\nСтрана: %s\n\nКлюч (скопируй и вставь в Outline):\n%s\n\nСкачать Outline Client:\n%s",
		resp.ServerName,
		resp.Country,
		resp.AccessURL,
		officialLinks(),
	)

	msg := tgbotapi.NewMessage(s.ChatID, text)
	msg.ReplyMarkup = menu.Keyboard()
	_, err = d.Bot.Send(msg)
	return err
}

func officialLinks() string {
	return "" +
		"- Windows: https://s3.amazonaws.com/outline-releases/client/windows/stable/Outline-Client.exe\n" +
		"- macOS: https://s3.amazonaws.com/outline-releases/client/macos/stable/Outline-Client.dmg\n" +
		"- iOS: https://itunes.apple.com/us/app/outline-app/id1356177741\n" +
		"- Android: https://play.google.com/store/apps/details?id=org.outline.android.client\n" +
		"- Android (APK): https://s3.amazonaws.com/outline-releases/client/android/stable/Outline-Client.apk\n"
}
