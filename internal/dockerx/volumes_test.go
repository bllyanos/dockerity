package dockerx

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
	"github.com/moby/moby/client"
)

func TestCheckArchiveName(t *testing.T) {
	for _, ok := range []string{"dkt-shop-prod-db-data-20260101T000000Z.tar.gz", "a.tar.gz"} {
		if err := checkArchiveName(ok); err != nil {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"", "x.tar", "../x.tar.gz", "a b.tar.gz", "a;rm.tar.gz", ".hidden.tar.gz", "a/b.tar.gz"} {
		if err := checkArchiveName(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// TestArchiveAndRestoreWithDocker round-trips data through a backup.
func TestArchiveAndRestoreWithDocker(t *testing.T) {
	c, ctx := dockerTest(t)
	run := fmt.Sprintf("t%d", time.Now().UnixNano()%1e9)
	data, backups := "dkt-test-"+run+"-data", "dkt-test-"+run+"-backups"
	file := "dkt-test-" + run + ".tar.gz"
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = c.api.VolumeRemove(bg, data, client.VolumeRemoveOptions{Force: true})
		_, _ = c.api.VolumeRemove(bg, backups, client.VolumeRemoveOptions{Force: true})
	})
	for _, v := range []string{data, backups} {
		if err := c.EnsureVolume(ctx, v, nil); err != nil {
			t.Fatal(err)
		}
	}
	// writes or reads /data/hello through a throwaway container.
	exec := func(cmd string) string {
		t.Helper()
		id, err := c.CreateContainer(ctx, containerspec.Spec{
			Name: "dkt-test-" + run + "-" + fmt.Sprint(time.Now().UnixNano()), Image: HelperImage, Cmd: []string{"sh", "-c", cmd},
			Mounts: []containerspec.Mount{{Volume: data, Target: "/data"}}, RestartPolicy: containerspec.RestartNo, StopTimeout: time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = c.RemoveContainer(context.Background(), id) }()
		_ = c.StartContainer(ctx, id)
		if code, err := c.WaitExit(ctx, id); err != nil || code != 0 {
			t.Fatalf("%s: exit %d %v", cmd, code, err)
		}
		var out bytes.Buffer
		_ = c.Logs(ctx, id, false, &out)
		return strings.TrimSpace(out.String())
	}

	exec("mkdir -p /data/sub && echo precious > /data/sub/hello")
	size, err := c.ArchiveVolume(ctx, data, backups, file)
	if err != nil || size <= 0 {
		t.Fatalf("archive: %d %v", size, err)
	}
	exec("rm -rf /data/sub && echo junk > /data/junk")
	if err := c.RestoreVolume(ctx, data, backups, file); err != nil {
		t.Fatal(err)
	}
	if got := exec("cat /data/sub/hello; ls /data"); got != "precious\nsub" {
		t.Fatalf("restored content %q", got)
	}
	if err := c.RemoveArchive(ctx, backups, file); err != nil {
		t.Fatal(err)
	}
	if err := c.RestoreVolume(ctx, data, backups, file); err == nil {
		t.Fatal("restoring a removed archive must fail")
	}
}
