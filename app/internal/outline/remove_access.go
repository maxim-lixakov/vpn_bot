package outline

import (
	"context"
	"net/http"
)

func (c *Client) RemoveAccessKeyDataLimit(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/access-keys/"+id+"/data-limit", nil, nil)
}
