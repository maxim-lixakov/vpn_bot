package router

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpn-bot/internal/appclient"
)

type Deps struct {
	App *appclient.Client
	Bot *tgbotapi.BotAPI
	Cfg Config
}

type Config struct {
	Payments PaymentsConfig
}

type PaymentsConfig struct {
	ProviderToken string
	Currency      string
	PriceMinor    int64
	Title         string
	Description   string
	Payload       string
}

type StateHandler interface {
	Name() string
	CanHandle(u tgbotapi.Update, s Session) bool
	Handle(ctx context.Context, u tgbotapi.Update, s Session, d Deps) error
}

type Session struct {
	TgUserID        int64
	ChatID          int64
	State           string
	SelectedCountry *string
	SubscriptionOK  bool
}

type Router struct {
	handlers []StateHandler
}

func NewRouter(h ...StateHandler) *Router { return &Router{handlers: h} }

func (r *Router) Dispatch(ctx context.Context, u tgbotapi.Update, s Session, d Deps) error {
	for _, h := range r.handlers {
		if h.CanHandle(u, s) {
			return h.Handle(ctx, u, s, d)
		}
	}
	// no handler -> ignore
	return nil
}
