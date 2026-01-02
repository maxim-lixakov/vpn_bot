package handlers

import (
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/menu"
	"vpn-bot/internal/router"
)

type PaymentFlow struct{}

func (h PaymentFlow) Name() string { return "payment" }

func (h PaymentFlow) CanHandle(u tgbotapi.Update, s router.Session) bool {
	return u.PreCheckoutQuery != nil || (u.Message != nil && u.Message.SuccessfulPayment != nil)
}

func (h PaymentFlow) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	// 1) pre-checkout: must answer OK within ~10 seconds
	if u.PreCheckoutQuery != nil {
		pc := tgbotapi.PreCheckoutConfig{
			PreCheckoutQueryID: u.PreCheckoutQuery.ID,
			OK:                 true,
		}
		_, err := d.Bot.Request(pc)
		return err
	}

	// 2) successful payment
	sp := u.Message.SuccessfulPayment
	payload := sp.InvoicePayload

	// Проверяем, является ли это продлением подписки
	isRenewal := strings.HasPrefix(payload, d.Cfg.Payments.VPNRenewalPayload+":")

	if isRenewal {
		// Продление подписки - извлекаем country_code из payload (формат: "vpn_renewal_v1:subscription_id:country_code")
		parts := strings.Split(payload, ":")
		var countryCode *string
		if len(parts) >= 3 && parts[2] != "" {
			countryCode = &parts[2]
		}

		_, err := d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
			TgUserID:    s.TgUserID,
			Kind:        "vpn",
			CountryCode: countryCode,
			AmountMinor: int64(sp.TotalAmount),
			Currency:    sp.Currency,

			TelegramPaymentChargeID: sp.TelegramPaymentChargeID,
			ProviderPaymentChargeID: payload, // Используем payload для идентификации продления
		})
		if err != nil {
			msg := tgbotapi.NewMessage(s.ChatID, "Оплата получена, но не смог продлить подписку: "+err.Error())
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			return nil
		}

		return nil
	}

	switch payload {
	case d.Cfg.Payments.VPNPayload:
		if s.SelectedCountry == nil {
			msg := tgbotapi.NewMessage(s.ChatID, "Не выбрана страна. Нажми /start")
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			return nil
		}

		_, err := d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
			TgUserID:    s.TgUserID,
			Kind:        "vpn",
			CountryCode: s.SelectedCountry,
			AmountMinor: int64(sp.TotalAmount),
			Currency:    sp.Currency,

			TelegramPaymentChargeID: sp.TelegramPaymentChargeID,
			ProviderPaymentChargeID: sp.ProviderPaymentChargeID,
		})
		if err != nil {
			msg := tgbotapi.NewMessage(s.ChatID, "Оплата получена, но не смог сохранить подписку: "+err.Error())
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			return nil
		}

		// выдаём ключ + инструкцию с картинками
		return IssueKeyNow(ctx, s, d)

	case d.Cfg.Payments.NewCountryPayload:
		_, err := d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
			TgUserID:    s.TgUserID,
			Kind:        "country_request",
			CountryCode: nil,
			AmountMinor: int64(sp.TotalAmount),
			Currency:    sp.Currency,

			TelegramPaymentChargeID: sp.TelegramPaymentChargeID,
			ProviderPaymentChargeID: sp.ProviderPaymentChargeID,
		})
		if err != nil {
			msg := tgbotapi.NewMessage(s.ChatID, "Оплата получена, но не смог сохранить: "+err.Error())
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			return nil
		}

		_ = d.App.TelegramSetState(ctx, s.TgUserID, "AWAIT_COUNTRY_REQUEST_TEXT", nil)

		msg := tgbotapi.NewMessage(s.ChatID, "Какую страну ты бы хотел добавить?")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	msg := tgbotapi.NewMessage(s.ChatID, "Оплата получена, но payload не распознан.")
	msg.ReplyMarkup = menu.Keyboard()
	_, _ = d.Bot.Send(msg)
	return nil
}

func IssueKeyNow(ctx context.Context, s router.Session, d router.Deps) error {
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

	// если app говорит "нужна оплата" — тут можно отправить invoice или dev-bypass (по твоей логике)
	if resp.Status == "payment_required" {
		if d.Cfg.Payments.ProviderToken == "" {
			_, err := d.App.TelegramMarkPaid(ctx, appclient.TelegramMarkPaidReq{
				TgUserID:    s.TgUserID,
				Kind:        "vpn",
				CountryCode: s.SelectedCountry,
				AmountMinor: d.Cfg.Payments.VPNPriceMinor,
				Currency:    d.Cfg.Payments.Currency,

				TelegramPaymentChargeID: "dev-bypass",
				ProviderPaymentChargeID: "dev-bypass",
			})
			if err != nil {
				msg := tgbotapi.NewMessage(s.ChatID, "Не смог сохранить оплату: "+err.Error())
				msg.ReplyMarkup = menu.Keyboard()
				_, _ = d.Bot.Send(msg)
				return nil
			}

			resp2, err := d.App.IssueKey(ctx, s.TgUserID, *s.SelectedCountry)
			if err != nil || resp2.Status != "ok" {
				msg := tgbotapi.NewMessage(s.ChatID, "Оплата сохранена, но ключ пока не выдался. Попробуй ещё раз.")
				msg.ReplyMarkup = menu.Keyboard()
				_, _ = d.Bot.Send(msg)
				return nil
			}
			resp = resp2
		} else {
			msg := tgbotapi.NewMessage(s.ChatID, "Нужна оплата 100р/мес. Нажми кнопку оплаты ещё раз.")
			msg.ReplyMarkup = menu.Keyboard()
			_, _ = d.Bot.Send(msg)
			return nil
		}
	}

	if resp.Status != "ok" {
		msg := tgbotapi.NewMessage(s.ChatID, "Неожиданный ответ от сервера.")
		msg.ReplyMarkup = menu.Keyboard()
		_, _ = d.Bot.Send(msg)
		return nil
	}

	key := html.EscapeString(resp.AccessURL)
	server := html.EscapeString(resp.ServerName)

	msgText := fmt.Sprintf(
		"<b>Сервер:</b> %s\n\n<b>Ключ:</b>\n<pre><code>%s</code></pre>\n<b>\nСкачать Outline Client:</b>\n• <a href=\"%s\">iOS — скачать</a>\n• <a href=\"%s\">Android — скачать</a>\n• <a href=\"%s\">Desktop (Windows/macOS/Linux) — скачать</a>",
		server,
		key,
		"https://apps.apple.com/ru/app/outline-app/id1356177741",
		"https://play.google.com/store/apps/details?id=org.outline.android.client&pcampaignid=web_share",
		"https://getoutline.org/intl/ru/get-started/#step-3",
	)

	m := tgbotapi.NewMessage(s.ChatID, msgText)
	m.ParseMode = "HTML"
	m.DisableWebPagePreview = true
	m.ReplyMarkup = menu.Keyboard()
	if _, err := d.Bot.Send(m); err != nil {
		return err
	}

	baseDir := "internal/images"

	if err := sendTextAndImage(d.Bot, s.ChatID, "После установки приложения, откройте его.", filepath.Join(baseDir, "step1.png")); err != nil {
		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Не смог отправить step1.png: "+err.Error()))
	}
	if err := sendTextAndImage(d.Bot, s.ChatID, "Вставьте сюда скопированный ключ.", filepath.Join(baseDir, "step2.png")); err != nil {
		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Не смог отправить step2.png: "+err.Error()))
	}
	if err := sendTextAndImage(d.Bot, s.ChatID, "Нажмите «Подтвердить», а затем «Подключить».\nVPN должен работать — проверяйте.", filepath.Join(baseDir, "step3.png")); err != nil {
		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Не смог отправить step3.png: "+err.Error()))
	}
	if err := sendTextAndImage(d.Bot, s.ChatID, "Если вы купили больше одного VPN — в правом верхнем углу нажмите плюсик и повторите предыдущие шаги.", filepath.Join(baseDir, "step4.png")); err != nil {
		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Не смог отправить step4.png: "+err.Error()))
	}

	return nil
}

func sendTextAndImage(bot *tgbotapi.BotAPI, chatID int64, text string, imagePath string) error {
	// 1) текст
	if _, err := bot.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		return err
	}

	// 2) картинка
	b, err := os.ReadFile(imagePath)
	if err != nil {
		return err
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
		Name:  filepath.Base(imagePath),
		Bytes: b,
	})
	photo.DisableNotification = true
	_, err = bot.Send(photo)
	return err
}
