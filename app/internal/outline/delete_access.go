package outline

import (
	"context"
	"net/http"
)

func (c *Client) DeleteAccessKey(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/access-keys/"+id, nil, nil)
}
