package dockerx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
)

// settle is how long a container without a health check must keep running
// before it counts as started.
const settle = 3 * time.Second

// WaitHealthy waits until a started container is ready: its Docker health
// check reports healthy, its HTTP check answers 2xx/3xx over the environment
// network, or, without any check, it is still running after a short settle
// period. A container that exits or turns unhealthy fails immediately.
func (c *Client) WaitHealthy(ctx context.Context, id string, s containerspec.Spec) error {
	switch {
	case s.Healthcheck != nil:
		hc := s.Healthcheck
		deadline := hc.StartPeriod + time.Duration(hc.Retries+2)*(hc.Interval+hc.Timeout)
		return c.poll(ctx, id, deadline, func(info ContainerInfo) (bool, error) {
			switch info.Health {
			case "healthy":
				return true, nil
			case "unhealthy":
				return false, fmt.Errorf("container %s is unhealthy", info.Name)
			}
			return false, nil
		})
	case s.HTTPCheck != nil:
		hc := s.HTTPCheck
		deadline := hc.StartPeriod + time.Duration(hc.Retries+2)*(hc.Interval+hc.Timeout)
		probe := &http.Client{Timeout: hc.Timeout}
		return c.poll(ctx, id, deadline, func(info ContainerInfo) (bool, error) {
			ip := info.IPs[s.Network]
			if ip == "" {
				return false, nil
			}
			url := "http://" + net.JoinHostPort(ip, strconv.Itoa(hc.Port)) + hc.Path
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return false, err
			}
			res, err := probe.Do(req)
			if err != nil {
				return false, nil // not listening yet
			}
			_ = res.Body.Close()
			return res.StatusCode >= 200 && res.StatusCode < 400, nil
		})
	default:
		start := time.Now()
		return c.poll(ctx, id, settle*2, func(ContainerInfo) (bool, error) {
			return time.Since(start) >= settle, nil
		})
	}
}

// poll checks the container every half second until ready reports true, the
// container stops running, or the deadline passes.
func (c *Client) poll(ctx context.Context, id string, deadline time.Duration, ready func(ContainerInfo) (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		info, err := c.InspectContainer(ctx, id)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("container %s did not become ready within %s", id, deadline)
			}
			return err
		}
		if !info.Running {
			return fmt.Errorf("container %s exited with code %d before becoming ready", info.Name, info.ExitCode)
		}
		ok, err := ready(info)
		if err != nil || ok {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("container %s did not become ready within %s", info.Name, deadline)
		case <-tick.C:
		}
	}
}
