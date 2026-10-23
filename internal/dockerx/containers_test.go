package dockerx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
	"github.com/bllyanos/dockerity/internal/secrets/inject"
	"github.com/moby/moby/client"
)

// dockerTest returns a client for integration tests, or skips without Docker.
func dockerTest(t *testing.T) (*Client, context.Context) {
	t.Helper()
	c, err := New()
	if err != nil {
		t.Skip("docker client unavailable:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	if err := c.Ping(ctx); err != nil {
		t.Skip("docker engine unavailable:", err)
	}
	if err := c.PullImage(ctx, "busybox:1.37", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	return c, ctx
}

func TestContainerLifecycleWithDocker(t *testing.T) {
	c, ctx := dockerTest(t)
	run := fmt.Sprintf("t%d", time.Now().UnixNano()%1e9)
	netName := "dkt-test-" + run
	volName := "dkt-test-" + run + "-data"
	labels := map[string]string{"dockerity.test": run}
	var ids []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range ids {
			_, _ = c.api.ContainerRemove(bg, id, client.ContainerRemoveOptions{Force: true})
		}
		_ = c.RemoveNetwork(bg, netName)
		_, _ = c.api.VolumeRemove(bg, volName, client.VolumeRemoveOptions{Force: true})
	})

	if err := c.EnsureNetwork(ctx, netName, labels); err != nil {
		t.Fatal(err)
	}
	if err := c.EnsureNetwork(ctx, netName, labels); err != nil {
		t.Fatalf("ensure is idempotent: %v", err)
	}
	if err := c.EnsureVolume(ctx, volName, labels); err != nil {
		t.Fatal(err)
	}

	create := func(name string, cmd []string, mod func(*containerspec.Spec)) (string, containerspec.Spec) {
		t.Helper()
		s := containerspec.Spec{
			Name: netName + "-" + name, Image: "busybox:1.37", Cmd: cmd, Network: netName, Aliases: []string{name},
			Labels: map[string]string{"dockerity.test": run}, RestartPolicy: containerspec.RestartNo, StopTimeout: time.Second,
		}
		if mod != nil {
			mod(&s)
		}
		id, err := c.CreateContainer(ctx, s)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		return id, s
	}

	// A job with a secret file and a volume: exit code and logs.
	jobID, _ := create("job", []string{"sh", "-c", "cat /run/secrets/token && echo hi > /data/x && exit 3"}, func(s *containerspec.Spec) {
		s.Files = []inject.File{{Path: "/run/secrets/token", Content: []byte("tok-123\n"), Mode: 0o400}}
		s.Mounts = []containerspec.Mount{{Volume: volName, Target: "/data"}}
	})
	if err := c.StartContainer(ctx, jobID); err != nil {
		t.Fatal(err)
	}
	code, err := c.WaitExit(ctx, jobID)
	if err != nil || code != 3 {
		t.Fatalf("exit code %d %v", code, err)
	}
	var logs bytes.Buffer
	if err := c.Logs(ctx, jobID, false, &logs); err != nil || !strings.Contains(logs.String(), "tok-123") {
		t.Fatalf("logs %q %v", logs.String(), err)
	}

	// Docker-native health check.
	hcID, hcSpec := create("hc", []string{"sleep", "300"}, func(s *containerspec.Spec) {
		s.Healthcheck = &containerspec.Healthcheck{Test: []string{"CMD", "true"}, Interval: time.Second, Timeout: time.Second, Retries: 2}
	})
	if err := c.StartContainer(ctx, hcID); err != nil {
		t.Fatal(err)
	}
	if err := c.WaitHealthy(ctx, hcID, hcSpec); err != nil {
		t.Fatalf("docker health check: %v", err)
	}

	// HTTP health check probed over the environment network.
	webID, webSpec := create("web", []string{"sh", "-c", "mkdir -p /www && echo ok > /www/healthz && httpd -f -p 8080 -h /www"}, func(s *containerspec.Spec) {
		s.HTTPCheck = &containerspec.HTTPCheck{Port: 8080, Path: "/healthz", Interval: time.Second, Timeout: time.Second, Retries: 3}
	})
	if err := c.StartContainer(ctx, webID); err != nil {
		t.Fatal(err)
	}
	if err := c.WaitHealthy(ctx, webID, webSpec); err != nil {
		t.Fatalf("http health check: %v", err)
	}

	// A container that exits is never healthy.
	badID, badSpec := create("bad", []string{"false"}, nil)
	_ = c.StartContainer(ctx, badID)
	if err := c.WaitHealthy(ctx, badID, badSpec); err == nil || !strings.Contains(err.Error(), "exited") {
		t.Fatalf("exited container: %v", err)
	}

	// The system container guard.
	sysID, _ := create("sys", []string{"sleep", "300"}, func(s *containerspec.Spec) {
		s.Labels["dockerity.system"] = "true"
	})
	if err := c.StopContainer(ctx, sysID, time.Second); !errors.Is(err, ErrSystemContainer) {
		t.Fatalf("stop system container: %v", err)
	}
	if err := c.RemoveContainer(ctx, sysID); !errors.Is(err, ErrSystemContainer) {
		t.Fatalf("remove system container: %v", err)
	}

	managed, err := c.ListManaged(ctx, map[string]string{"dockerity.test": run})
	if err != nil || len(managed) != 4 {
		t.Fatalf("managed %d %v", len(managed), err)
	}
	for _, m := range managed {
		if m.ID == sysID {
			t.Fatal("system container listed as managed")
		}
	}

	if err := c.StopContainer(ctx, webID, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveContainer(ctx, webID); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveContainer(ctx, webID); err != nil {
		t.Fatalf("removing a missing container: %v", err)
	}
}
