package dockerx

import (
	"context"

	"github.com/moby/moby/client"
)

// ImageInfo is a local image.
type ImageInfo struct {
	ID   string
	Tags []string
}

// ListImages lists local images carrying all labels.
func (c *Client) ListImages(ctx context.Context, labels map[string]string) ([]ImageInfo, error) {
	f := make(client.Filters)
	for k, v := range labels {
		f.Add("label", k+"="+v)
	}
	res, err := c.api.ImageList(ctx, client.ImageListOptions{Filters: f})
	if err != nil {
		return nil, err
	}
	out := make([]ImageInfo, 0, len(res.Items))
	for _, s := range res.Items {
		out = append(out, ImageInfo{ID: s.ID, Tags: s.RepoTags})
	}
	return out, nil
}
