package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
	"vpn-bot/internal/handlers"
	stateRouter "vpn-bot/internal/router"
	"vpn-bot/internal/utils"
)

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	appBaseURL := os.Getenv("APP_BASE_URL")
	internalToken := os.Getenv("APP_INTERNAL_TOKEN")
	if botToken == "" || appBaseURL == "" || internalToken == "" {
		log.Fatal("BOT_TOKEN, APP_BASE_URL, APP_INTERNAL_TOKEN are required")
	}

	pcfg := stateRouter.PaymentsConfig{
		ProviderToken: strings.TrimSpace(os.Getenv("PAYMENTS_PROVIDER_TOKEN")),
		Currency:      utils.GetEnv("PAYMENTS_CURRENCY", "RUB"),

		VPNPriceMinor:  utils.MustInt64(utils.GetEnv("PAYMENTS_VPN_PRICE_MINOR", "15000")),
		VPNTtitle:      utils.GetEnv("PAYMENTS_VPN_TITLE", "LICKSUCKOFF VPN"),
		VPNDescription: utils.GetEnv("PAYMENTS_VPN_DESCRIPTION", "VPN подписка на 1 месяц"),
		VPNPayload:     utils.GetEnv("PAYMENTS_VPN_PAYLOAD", "vpn_sub_v1"),

		NewCountryPriceMinor:  utils.MustInt64(utils.GetEnv("PAYMENTS_NEWCOUNTRY_PRICE_MINOR", "40000")),
		NewCountryTitle:       utils.GetEnv("PAYMENTS_NEWCOUNTRY_TITLE", "Добавить новую страну"),
		NewCountryDescription: utils.GetEnv("PAYMENTS_NEWCOUNTRY_DESCRIPTION", "Запрос на добавление новой страны"),
		NewCountryPayload:     utils.GetEnv("PAYMENTS_NEWCOUNTRY_PAYLOAD", "new_country_v1"),
	}

	app := appclient.New(appBaseURL, internalToken)

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bot authorized as @%s", bot.Self.UserName)

	router := stateRouter.NewRouter(
		handlers.Start{},
		handlers.Menu{},
		handlers.MySubscriptions{},
		handlers.ChooseVPN{},
		handlers.OrderNewCountry{},
		handlers.CountryChosen{},
		handlers.CountryRequestText{},
		handlers.PaymentFlow{},
	)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	u.AllowedUpdates = []string{"message", "callback_query", "pre_checkout_query"}

	updates := bot.GetUpdatesChan(u)

	deps := stateRouter.Deps{
		App: app,
		Bot: bot,
		Cfg: stateRouter.Config{Payments: pcfg},
	}

	for upd := range updates {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		// всегда снимаем "loading"
		if upd.CallbackQuery != nil {
			_, _ = bot.Request(tgbotapi.NewCallback(upd.CallbackQuery.ID, ""))
		}

		sess, ok := buildSession(ctx, upd, app)
		if !ok {
			cancel()
			continue
		}

		if upd.CallbackQuery != nil {
			log.Printf("callback data=%q state=%q selected=%v", upd.CallbackQuery.Data, sess.State, sess.SelectedCountry)
		}

		if err := router.Dispatch(ctx, upd, sess, deps); err != nil {
			log.Printf("handle error: %v", err)
		}
		cancel()
	}
}

func buildSession(ctx context.Context, upd tgbotapi.Update, app *appclient.Client) (stateRouter.Session, bool) {
	var (
		tgID   int64
		chatID int64
		user   *tgbotapi.User
	)

	switch {
	case upd.Message != nil:
		tgID = upd.Message.From.ID
		chatID = upd.Message.Chat.ID
		user = upd.Message.From
	case upd.CallbackQuery != nil:
		tgID = upd.CallbackQuery.From.ID
		if upd.CallbackQuery.Message != nil {
			chatID = upd.CallbackQuery.Message.Chat.ID
		} else {
			chatID = tgID
		}
		user = upd.CallbackQuery.From
	case upd.PreCheckoutQuery != nil:
		tgID = upd.PreCheckoutQuery.From.ID
		chatID = tgID
		user = upd.PreCheckoutQuery.From
	default:
		return stateRouter.Session{}, false
	}

	req := appclient.TelegramUpsertReq{TgUserID: tgID}
	if user != nil {
		if user.UserName != "" {
			u := user.UserName
			req.Username = &u
		}
		if user.FirstName != "" {
			fn := user.FirstName
			req.FirstName = &fn
		}
		if user.LastName != "" {
			ln := user.LastName
			req.LastName = &ln
		}
		if user.LanguageCode != "" {
			lc := user.LanguageCode
			req.LanguageCode = &lc
		}
	}

	resp, err := app.TelegramUpsert(ctx, req)
	if err != nil {
		log.Printf("TelegramUpsert failed: %v", err)
		return stateRouter.Session{}, false
	}

	st := strings.TrimSpace(resp.Router)
	if st == "" {
		st = strings.TrimSpace(resp.State)
	}

	return stateRouter.Session{
		TgUserID:        tgID,
		ChatID:          chatID,
		State:           st,
		SelectedCountry: resp.SelectedCountry,
		SubscriptionOK:  resp.SubscriptionOK,
	}, true
}
