package outline

import (
	"context"
	"net/http"
)

type AccessKey struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Password  string `json:"password,omitempty"`
	Port      int    `json:"port,omitempty"`
	Method    string `json:"method,omitempty"`
	AccessURL string `json:"accessUrl,omitempty"`
	// dataLimit optional in list, see api.yml :contentReference[oaicite:6]{index=6}
	DataLimit *struct {
		Bytes int64 `json:"bytes"`
	} `json:"dataLimit,omitempty"`
}

type createAccessKeyReq struct {
	Name string `json:"name,omitempty"`
	// Можно задать limit на создание (пример в api.yml) :contentReference[oaicite:7]{index=7}
	Limit *struct {
		Bytes int64 `json:"bytes"`
	} `json:"limit,omitempty"`
}

func (c *Client) CreateAccessKey(ctx context.Context, name string) (AccessKey, error) {
	reqBody := createAccessKeyReq{Name: name}

	var out AccessKey
	if err := c.doJSON(ctx, http.MethodPost, "/access-keys", reqBody, &out); err != nil {
		return AccessKey{}, err
	}
	return out, nil
}
