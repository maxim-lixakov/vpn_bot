package outline

import (
	"context"
	"net/http"
)

type dataLimitReq struct {
	// В примерах request body выглядит как {"limit":{"bytes":10000}} :contentReference[oaicite:8]{index=8}
	Limit struct {
		Bytes int64 `json:"bytes"`
	} `json:"limit"`
}

func (c *Client) SetAccessKeyDataLimit(ctx context.Context, id string, bytesLimit int64) error {
	var req dataLimitReq
	req.Limit.Bytes = bytesLimit
	return c.doJSON(ctx, http.MethodPut, "/access-keys/"+id+"/data-limit", req, nil)
}
