package payments

import (
	"encoding/json"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendInvoiceRaw sends invoice via low-level MakeRequest to avoid library quirks with tips fields.
// Telegram expects `prices` as JSON-serialized array. :contentReference[oaicite:1]{index=1}
func SendInvoiceRaw(
	bot *tgbotapi.BotAPI,
	chatID int64,
	title string,
	description string,
	payload string,
	providerToken string,
	currency string,
	prices []tgbotapi.LabeledPrice,
) error {
	if providerToken == "" {
		return fmt.Errorf("providerToken is empty")
	}
	if currency == "" {
		return fmt.Errorf("currency is empty")
	}
	if len(prices) == 0 {
		return fmt.Errorf("prices is empty")
	}

	pricesJSON, err := json.Marshal(prices)
	if err != nil {
		return fmt.Errorf("marshal prices: %w", err)
	}

	// Note: we intentionally do NOT send max_tip_amount / suggested_tip_amounts fields at all.
	// This avoids Telegram error and UI tips features entirely.
	params := tgbotapi.Params{
		"chat_id":        strconv.FormatInt(chatID, 10),
		"title":          title,
		"description":    description,
		"payload":        payload,
		"provider_token": providerToken,
		"currency":       currency,
		"prices":         string(pricesJSON),
	}

	_, err = bot.MakeRequest("sendInvoice", params)
	if err != nil {
		return err
	}
	return nil
}

func SendVPNInvoice(
	bot *tgbotapi.BotAPI,
	chatID int64,
	providerToken string,
	currency string,
	title string,
	description string,
	payload string,
	country string,
	amountMinor int64,
) error {
	prices := []tgbotapi.LabeledPrice{
		{Label: "VPN 1 month", Amount: int(amountMinor)},
	}
	desc := description + "\nCountry: " + country
	return SendInvoiceRaw(bot, chatID, title, desc, payload, providerToken, currency, prices)
}

func SendNewCountryInvoice(
	bot *tgbotapi.BotAPI,
	chatID int64,
	providerToken string,
	currency string,
	title string,
	description string,
	payload string,
	amountMinor int64,
) error {
	prices := []tgbotapi.LabeledPrice{
		{Label: "New country request", Amount: int(amountMinor)},
	}
	return SendInvoiceRaw(bot, chatID, title, description, payload, providerToken, currency, prices)
}
