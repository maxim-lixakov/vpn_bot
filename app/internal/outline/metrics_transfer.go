package outline

import (
	"context"
	"net/http"
)

type metricsTransferResp struct {
	BytesTransferredByUserID map[string]int64 `json:"bytesTransferredByUserId"`
}

func (c *Client) MetricsTransfer(ctx context.Context) (map[string]int64, error) {
	var out metricsTransferResp
	if err := c.doJSON(ctx, http.MethodGet, "/metrics/transfer", nil, &out); err != nil {
		return nil, err
	}
	return out.BytesTransferredByUserID, nil
}
