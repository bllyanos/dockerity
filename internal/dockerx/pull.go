package dockerx

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/moby/moby/client"
)

// PullImage pulls ref from its registry, writing one line per layer state
// change (not every progress tick) to logs. Builds pull their base images
// first, because BuildKit through the Engine API cannot resolve registry
// images without a client session on every Docker setup.
func (c *Client) PullImage(ctx context.Context, ref string, logs io.Writer) error {
	res, err := c.api.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull %s: %w", ref, err)
	}
	defer res.Close()
	last := map[string]string{}
	for msg, err := range res.JSONMessages(ctx) {
		if err != nil {
			return fmt.Errorf("pull %s: %w", ref, err)
		}
		if msg.Error != nil {
			return fmt.Errorf("pull %s: %w", ref, errors.New(msg.Error.Message))
		}
		if msg.Status == "" || last[msg.ID] == msg.Status {
			continue
		}
		last[msg.ID] = msg.Status
		if msg.ID != "" {
			_, _ = fmt.Fprintf(logs, "%s: %s\n", msg.ID, msg.Status)
		} else {
			_, _ = fmt.Fprintln(logs, msg.Status)
		}
	}
	return nil
}
