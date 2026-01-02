package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
}

func SendInvoice(botToken string, chatID int64, title, description, payload, providerToken, currency string, prices []LabeledPrice) error {
	if botToken == "" {
		return fmt.Errorf("bot token is empty")
	}
	if providerToken == "" {
		return fmt.Errorf("provider token is empty")
	}
	if currency == "" {
		return fmt.Errorf("currency is empty")
	}
	if len(prices) == 0 {
		return fmt.Errorf("prices is empty")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendInvoice", botToken)

	pricesJSON, err := json.Marshal(prices)
	if err != nil {
		return fmt.Errorf("marshal prices: %w", err)
	}

	payloadMap := map[string]interface{}{
		"chat_id":        chatID,
		"title":          title,
		"description":    description,
		"payload":        payload,
		"provider_token": providerToken,
		"currency":       currency,
		"prices":         string(pricesJSON),
	}

	jsonData, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram api error: %s, body: %s", resp.Status, string(body))
	}

	return nil
}
