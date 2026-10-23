package dockerx

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// ErrSystemContainer is returned when an operation targets Dockerity or Traefik.
var ErrSystemContainer = errors.New("refusing to touch a system container (dockerity.system=true)")

// ContainerInfo is what Dockerity needs to know about a container.
type ContainerInfo struct {
	ID       string
	Name     string
	Image    string
	Labels   map[string]string
	Running  bool
	ExitCode int
	// Health is "", "starting", "healthy" or "unhealthy".
	Health string
	// IPs maps network names to the container's address on them.
	IPs map[string]string
}

// EnsureNetwork creates a bridge network if it does not exist yet, and
// connects every system container (Dockerity and Traefik) to it so Traefik
// can route and Dockerity can probe health checks.
func (c *Client) EnsureNetwork(ctx context.Context, name string, labels map[string]string) error {
	if _, err := c.api.NetworkInspect(ctx, name, client.NetworkInspectOptions{}); err != nil {
		if !isNotFound(err) {
			return err
		}
		if _, err := c.api.NetworkCreate(ctx, name, client.NetworkCreateOptions{Driver: "bridge", Labels: labels}); err != nil && !isConflict(err) {
			return fmt.Errorf("create network %s: %w", name, err)
		}
	}
	system, err := c.api.ContainerList(ctx, client.ContainerListOptions{
		Filters: make(client.Filters).Add("label", domain.LabelSystem+"=true"),
	})
	if err != nil {
		return err
	}
	for _, s := range system.Items {
		if err := c.connect(ctx, name, s.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) connect(ctx context.Context, networkName, containerID string) error {
	_, err := c.api.NetworkConnect(ctx, networkName, client.NetworkConnectOptions{Container: containerID})
	if err != nil && !isConflict(err) && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("connect %s to %s: %w", containerID, networkName, err)
	}
	return nil
}

// RemoveNetwork deletes a network; a missing network is not an error.
func (c *Client) RemoveNetwork(ctx context.Context, name string) error {
	_, err := c.api.NetworkRemove(ctx, name, client.NetworkRemoveOptions{})
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

// EnsureVolume creates a named volume if it does not exist yet.
func (c *Client) EnsureVolume(ctx context.Context, name string, labels map[string]string) error {
	_, err := c.api.VolumeCreate(ctx, client.VolumeCreateOptions{Name: name, Labels: labels})
	return err // creating an existing local volume is a no-op
}

// CreateContainer creates (but does not start) a container from s and copies
// its secret files in.
func (c *Client) CreateContainer(ctx context.Context, s containerspec.Spec) (string, error) {
	stop := int(s.StopTimeout / time.Second)
	cfg := &container.Config{
		Image:       s.Image,
		Cmd:         s.Cmd,
		Env:         s.Env,
		Labels:      s.Labels,
		StopTimeout: &stop,
	}
	if hc := s.Healthcheck; hc != nil {
		cfg.Healthcheck = &container.HealthConfig{
			Test: hc.Test, Interval: hc.Interval, Timeout: hc.Timeout, StartPeriod: hc.StartPeriod, Retries: hc.Retries,
		}
	}
	host := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyMode(s.RestartPolicy)},
		Resources:     container.Resources{Memory: s.MemoryBytes, NanoCPUs: s.NanoCPUs},
	}
	for _, m := range s.Mounts {
		host.Mounts = append(host.Mounts, mount.Mount{Type: mount.TypeVolume, Source: m.Volume, Target: m.Target})
	}
	netCfg := &network.NetworkingConfig{}
	if s.Network != "" {
		netCfg.EndpointsConfig = map[string]*network.EndpointSettings{s.Network: {Aliases: s.Aliases}}
	}
	res, err := c.api.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name: s.Name, Config: cfg, HostConfig: host, NetworkingConfig: netCfg,
	})
	if err != nil {
		return "", fmt.Errorf("create container %s: %w", s.Name, err)
	}
	if len(s.Files) > 0 {
		archive, err := filesTar(s)
		if err != nil {
			return "", err
		}
		if _, err := c.api.CopyToContainer(ctx, res.ID, client.CopyToContainerOptions{DestinationPath: "/", Content: archive}); err != nil {
			_ = c.RemoveContainer(context.Background(), res.ID)
			return "", fmt.Errorf("copy secret files into %s: %w", s.Name, err)
		}
	}
	return res.ID, nil
}

// filesTar archives the spec's secret files with their parent directories.
func filesTar(s containerspec.Spec) (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	dirs := map[string]bool{}
	for _, f := range s.Files {
		var parents []string
		for d := path.Dir(f.Path); d != "/" && !dirs[d]; d = path.Dir(d) {
			parents = append(parents, d)
			dirs[d] = true
		}
		sort.Strings(parents)
		for _, d := range parents {
			if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: strings.TrimPrefix(d, "/") + "/", Mode: 0o755}); err != nil {
				return nil, err
			}
		}
		hdr := &tar.Header{Name: strings.TrimPrefix(f.Path, "/"), Mode: int64(f.Mode), Size: int64(len(f.Content))}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(f.Content); err != nil {
			return nil, err
		}
	}
	return &buf, tw.Close()
}

// StartContainer starts a created container.
func (c *Client) StartContainer(ctx context.Context, id string) error {
	_, err := c.api.ContainerStart(ctx, id, client.ContainerStartOptions{})
	return err
}

// StopContainer sends SIGTERM and kills after the grace period. System
// containers are refused.
func (c *Client) StopContainer(ctx context.Context, id string, grace time.Duration) error {
	if err := c.guard(ctx, id); err != nil {
		return err
	}
	secs := int(grace / time.Second)
	_, err := c.api.ContainerStop(ctx, id, client.ContainerStopOptions{Timeout: &secs})
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// RemoveContainer force-removes a container. System containers are refused;
// a missing container is not an error.
func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	if err := c.guard(ctx, id); err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}
	_, err := c.api.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// guard refuses operations on system containers.
func (c *Client) guard(ctx context.Context, id string) error {
	info, err := c.InspectContainer(ctx, id)
	if err != nil {
		return err
	}
	if domain.IsSystem(info.Labels) {
		return ErrSystemContainer
	}
	return nil
}

// InspectContainer returns a container's state.
func (c *Client) InspectContainer(ctx context.Context, id string) (ContainerInfo, error) {
	res, err := c.api.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return ContainerInfo{}, err
	}
	ct := res.Container
	info := ContainerInfo{ID: ct.ID, Name: strings.TrimPrefix(ct.Name, "/"), IPs: map[string]string{}}
	if ct.Config != nil {
		info.Image, info.Labels = ct.Config.Image, ct.Config.Labels
	}
	if ct.State != nil {
		info.Running, info.ExitCode = ct.State.Running, ct.State.ExitCode
		if ct.State.Health != nil {
			info.Health = string(ct.State.Health.Status)
		}
	}
	if ct.NetworkSettings != nil {
		for name, ep := range ct.NetworkSettings.Networks {
			if ep != nil && ep.IPAddress.IsValid() {
				info.IPs[name] = ep.IPAddress.String()
			}
		}
	}
	return info, nil
}

// WaitExit blocks until the container stops and returns its exit code.
func (c *Client) WaitExit(ctx context.Context, id string) (int, error) {
	res := c.api.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})
	select {
	case r := <-res.Result:
		if r.Error != nil && r.Error.Message != "" {
			return int(r.StatusCode), errors.New(r.Error.Message)
		}
		return int(r.StatusCode), nil
	case err := <-res.Error:
		return -1, err
	}
}

// Logs copies the container's stdout and stderr to w; with follow it keeps
// streaming until the container stops or ctx is cancelled.
func (c *Client) Logs(ctx context.Context, id string, follow bool, w io.Writer) error {
	rc, err := c.api.ContainerLogs(ctx, id, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: follow})
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = stdcopy.StdCopy(w, w, rc)
	return err
}

// ListManaged lists containers (running or not) matching all labels. System
// containers are never returned.
func (c *Client) ListManaged(ctx context.Context, labels map[string]string) ([]ContainerInfo, error) {
	f := make(client.Filters)
	for k, v := range labels {
		f.Add("label", k+"="+v)
	}
	res, err := c.api.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: f})
	if err != nil {
		return nil, err
	}
	var out []ContainerInfo
	for _, s := range res.Items {
		if domain.IsSystem(s.Labels) {
			continue
		}
		name := ""
		if len(s.Names) > 0 {
			name = strings.TrimPrefix(s.Names[0], "/")
		}
		out = append(out, ContainerInfo{ID: s.ID, Name: name, Image: s.Image, Labels: s.Labels, Running: s.State == container.StateRunning})
	}
	return out, nil
}
