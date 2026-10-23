package dockerx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/moby/moby/api/types/build"
	"github.com/moby/moby/client"
)

// BuildRequest describes one image build. There is deliberately no field for
// secrets: build args come from the app spec only.
type BuildRequest struct {
	Context    io.Reader // tar archive
	Dockerfile string    // path inside the context
	Target     string
	Args       map[string]string
	Tags       []string
	Labels     map[string]string
}

// buildMessage is one JSON line of the Engine API build stream.
type buildMessage struct {
	ID          string `json:"id"`
	Stream      string `json:"stream"`
	Error       string `json:"error"`
	ErrorDetail *struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
	Aux json.RawMessage `json:"aux"`
}

// BuildImage builds with BuildKit through the Engine API, writing readable
// progress to logs, and returns the image ID. Cancel ctx to stop the build.
func (c *Client) BuildImage(ctx context.Context, req BuildRequest, logs io.Writer) (string, error) {
	args := make(map[string]*string, len(req.Args))
	for k, v := range req.Args {
		args[k] = &v
	}
	res, err := c.api.ImageBuild(ctx, req.Context, client.ImageBuildOptions{
		Version:    build.BuilderBuildKit,
		Dockerfile: req.Dockerfile,
		Target:     req.Target,
		BuildArgs:  args,
		Tags:       req.Tags,
		Labels:     req.Labels,
		Remove:     true,
	})
	if err != nil {
		return "", fmt.Errorf("start build: %w", err)
	}
	defer res.Body.Close()
	return readBuildStream(res.Body, logs)
}

// readBuildStream consumes the build response and returns the built image ID.
func readBuildStream(r io.Reader, logs io.Writer) (string, error) {
	trace := newTraceRenderer(logs)
	dec := json.NewDecoder(r)
	var imageID string
	for {
		var m buildMessage
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("read build output: %w", err)
		}
		switch {
		case m.Error != "" || m.ErrorDetail != nil:
			msg := m.Error
			if m.ErrorDetail != nil && m.ErrorDetail.Message != "" {
				msg = m.ErrorDetail.Message
			}
			_, _ = fmt.Fprintf(logs, "ERROR: %s\n", msg)
			return "", errors.New(msg)
		case m.ID == "moby.buildkit.trace":
			if err := renderTrace(trace, m.Aux); err != nil {
				return "", err
			}
		case m.ID == "moby.image.id":
			var aux struct{ ID string }
			if err := json.Unmarshal(m.Aux, &aux); err != nil {
				return "", fmt.Errorf("read image id: %w", err)
			}
			imageID = aux.ID
		case m.Stream != "":
			if _, err := io.WriteString(logs, m.Stream); err != nil {
				return "", err
			}
		}
	}
	if imageID == "" {
		return "", errors.New("build finished without an image id")
	}
	return imageID, nil
}

func renderTrace(r *traceRenderer, aux json.RawMessage) error {
	var encoded string
	if err := json.Unmarshal(aux, &encoded); err != nil {
		return fmt.Errorf("read build trace: %w", err)
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("read build trace: %w", err)
	}
	s, err := decodeStatus(raw)
	if err != nil {
		return fmt.Errorf("decode build trace: %w", err)
	}
	return r.render(s)
}

// RemoveImage deletes an image by tag or ID, pruning untagged parents.
func (c *Client) RemoveImage(ctx context.Context, ref string) error {
	_, err := c.api.ImageRemove(ctx, ref, client.ImageRemoveOptions{PruneChildren: true})
	return err
}
