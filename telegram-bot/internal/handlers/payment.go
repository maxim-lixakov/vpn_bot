package handlers

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/router"
)

type PaymentFlow struct{}

func (h PaymentFlow) Name() string { return "payment" }

func (h PaymentFlow) CanHandle(u tgbotapi.Update, s router.Session) bool {
	// 1) pre-checkout query
	if u.PreCheckoutQuery != nil {
		return true
	}
	// 2) successful payment appears in Message.SuccessfulPayment
	if u.Message != nil && u.Message.SuccessfulPayment != nil {
		return true
	}
	// 3) we can trigger invoice when user is in AWAIT_PAYMENT and writes anything like "pay"
	if u.Message != nil && s.State == "AWAIT_PAYMENT" && !u.Message.IsCommand() {
		return true
	}
	return false
}

func (h PaymentFlow) Handle(ctx context.Context, u tgbotapi.Update, s router.Session, d router.Deps) error {
	// pre-checkout: must answer ok
	if u.PreCheckoutQuery != nil {
		pc := tgbotapi.PreCheckoutConfig{
			PreCheckoutQueryID: u.PreCheckoutQuery.ID,
			OK:                 true,
		}
		_, err := d.Bot.Request(pc)
		return err
	}

	// successful payment
	if u.Message != nil && u.Message.SuccessfulPayment != nil {
		sp := u.Message.SuccessfulPayment

		req := appclient.TelegramMarkPaidReq{
			TgUserID:                s.TgUserID,
			AmountMinor:             int64(sp.TotalAmount),
			Currency:                sp.Currency,
			TelegramPaymentChargeID: sp.TelegramPaymentChargeID,
			ProviderPaymentChargeID: sp.ProviderPaymentChargeID,
		}

		_, err := d.App.TelegramMarkPaid(ctx, req)
		if err != nil {
			_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Оплата получена, но не смог сохранить подписку: "+err.Error()))
			return nil
		}

		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Оплата успешна. Сейчас выдам ключ."))
		if s.SelectedCountry == nil {
			_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Не выбрана страна. Нажми /start"))
			return nil
		}
		return issueKeyNow(ctx, s, d)
	}

	// send invoice
	if d.Cfg.Payments.ProviderToken == "" {
		_, err := d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Оплата не настроена (PAYMENTS_PROVIDER_TOKEN пустой)."))
		return err
	}
	if s.SelectedCountry == nil {
		_, err := d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Сначала выбери страну через /start"))
		return err
	}

	price := tgbotapi.LabeledPrice{
		Label:  "VPN 1 month",
		Amount: int(d.Cfg.Payments.PriceMinor),
	}

	inv := tgbotapi.NewInvoice(
		s.ChatID,
		d.Cfg.Payments.Title,
		d.Cfg.Payments.Description+"\nCountry: "+*s.SelectedCountry,
		d.Cfg.Payments.Payload,
		d.Cfg.Payments.ProviderToken,
		"",
		d.Cfg.Payments.Currency,
		[]tgbotapi.LabeledPrice{price},
	)

	_, err := d.Bot.Send(inv)
	if err != nil {
		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Не смог отправить invoice: "+err.Error()+" (price="+strconv.FormatInt(d.Cfg.Payments.PriceMinor, 10)+")"))
	}
	return nil
}

func issueKeyNow(ctx context.Context, s router.Session, d router.Deps) error {
	resp, err := d.App.IssueKey(ctx, s.TgUserID, *s.SelectedCountry)
	if err != nil {
		_, _ = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, "Ошибка выдачи ключа: "+err.Error()))
		return nil
	}

	text := fmt.Sprintf("Сервер: %s\n\nКлюч:\n%s\n\n%s",
		resp.ServerName, resp.AccessURL, "Скачать Outline Client:\n"+officialLinks(),
	)
	_, err = d.Bot.Send(tgbotapi.NewMessage(s.ChatID, text))
	return err
}

func officialLinks() string {
	return "" +
		"- Windows: https://s3.amazonaws.com/outline-releases/client/windows/stable/Outline-Client.exe\n" +
		"- iOS: https://itunes.apple.com/us/app/outline-app/id1356177741\n" +
		"- Android: https://play.google.com/store/apps/details?id=org.outline.android.client\n" +
		"- Android (APK): https://s3.amazonaws.com/outline-releases/client/android/stable/Outline-Client.apk\n"
}
