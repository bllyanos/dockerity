package dockerx

import (
	"context"

	"github.com/moby/moby/client"
)

// PruneBuildCache removes the least recently used BuildKit cache until it
// uses at most maxBytes, and returns the bytes reclaimed.
func (c *Client) PruneBuildCache(ctx context.Context, maxBytes int64) (uint64, error) {
	res, err := c.api.BuildCachePrune(ctx, client.BuildCachePruneOptions{MaxUsedSpace: maxBytes})
	if err != nil {
		return 0, err
	}
	return res.Report.SpaceReclaimed, nil
}
