package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// SendMessage отправляет сообщение пользователю через Telegram Bot API
func SendMessage(botToken string, chatID int64, text string) error {
	if botToken == "" {
		return fmt.Errorf("bot token is empty")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	jsonData, err := json.Marshal(payload)
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
		return fmt.Errorf("telegram api error: %s", resp.Status)
	}

	return nil
}

// SendDocument отправляет документ/файл пользователю через Telegram Bot API
func SendDocument(botToken string, chatID int64, filename string, fileData []byte, caption string) error {
	if botToken == "" {
		return fmt.Errorf("bot token is empty")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", botToken)

	// Create multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add chat_id
	_ = writer.WriteField("chat_id", fmt.Sprintf("%d", chatID))

	// Add caption if provided
	if caption != "" {
		_ = writer.WriteField("caption", caption)
	}

	// Add file
	part, err := writer.CreateFormFile("document", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return fmt.Errorf("write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second} // Longer timeout for file uploads
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
