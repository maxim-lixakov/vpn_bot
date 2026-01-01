package appclient

import (
	"context"
	"net/http"
)

type BackupResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Backup triggers database backup and sends it to admin via Telegram
func (c *Client) Backup(ctx context.Context) (BackupResp, error) {
	var out BackupResp
	err := c.doJSON(ctx, http.MethodPost, "/v1/backup", nil, &out)
	return out, err
}
