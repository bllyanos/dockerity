package dockerx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

// HelperImage runs short-lived archive and restore jobs.
const HelperImage = "busybox:1.37"

// LabelHelper marks helper containers so they can be found and cleaned up.
const LabelHelper = "dockerity.helper"

var archiveName = regexp.MustCompile(`^[A-Za-z0-9._-]+\.tar\.gz$`)

func checkArchiveName(file string) error {
	if !archiveName.MatchString(file) || strings.HasPrefix(file, ".") {
		return fmt.Errorf("invalid archive name %q", file)
	}
	return nil
}

// ArchiveVolume writes volume as a gzipped tar into the backups volume and
// returns the archive's size. The volume is mounted read-only; the caller
// stops containers using it first so the archive is consistent.
func (c *Client) ArchiveVolume(ctx context.Context, volume, backupsVolume, file string) (int64, error) {
	if err := checkArchiveName(file); err != nil {
		return 0, err
	}
	script := fmt.Sprintf("tar czf /backups/.%[1]s.tmp -C /volume . && mv /backups/.%[1]s.tmp /backups/%[1]s && stat -c %%s /backups/%[1]s", file)
	out, err := c.runHelper(ctx, "backup", script,
		mount.Mount{Type: mount.TypeVolume, Source: volume, Target: "/volume", ReadOnly: true},
		mount.Mount{Type: mount.TypeVolume, Source: backupsVolume, Target: "/backups"},
	)
	if err != nil {
		return 0, fmt.Errorf("archive %s: %w", volume, err)
	}
	size, err := strconv.ParseInt(strings.TrimSpace(lastLine(out)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("archive %s: unexpected output %q", volume, out)
	}
	return size, nil
}

// RestoreVolume replaces the content of volume with an archive from the
// backups volume. Containers using the volume must be stopped first.
func (c *Client) RestoreVolume(ctx context.Context, volume, backupsVolume, file string) error {
	if err := checkArchiveName(file); err != nil {
		return err
	}
	script := fmt.Sprintf("test -f /backups/%[1]s && find /volume -mindepth 1 -delete && tar xzf /backups/%[1]s -C /volume", file)
	if _, err := c.runHelper(ctx, "restore", script,
		mount.Mount{Type: mount.TypeVolume, Source: volume, Target: "/volume"},
		mount.Mount{Type: mount.TypeVolume, Source: backupsVolume, Target: "/backups", ReadOnly: true},
	); err != nil {
		return fmt.Errorf("restore %s into %s: %w", file, volume, err)
	}
	return nil
}

// RemoveArchive deletes an archive from the backups volume.
func (c *Client) RemoveArchive(ctx context.Context, backupsVolume, file string) error {
	if err := checkArchiveName(file); err != nil {
		return err
	}
	_, err := c.runHelper(ctx, "prune", "rm -f /backups/"+file,
		mount.Mount{Type: mount.TypeVolume, Source: backupsVolume, Target: "/backups"})
	return err
}

// RemoveVolume deletes a named volume; a missing volume is not an error.
func (c *Client) RemoveVolume(ctx context.Context, name string) error {
	_, err := c.api.VolumeRemove(ctx, name, client.VolumeRemoveOptions{})
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

// runHelper runs script in a throwaway helper container and returns its
// output. A non-zero exit is an error that includes the output.
func (c *Client) runHelper(ctx context.Context, purpose, script string, mounts ...mount.Mount) (string, error) {
	if err := c.ensureImage(ctx, HelperImage); err != nil {
		return "", err
	}
	res, err := c.api.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image: HelperImage, Cmd: []string{"sh", "-c", script},
			Labels: map[string]string{LabelHelper: purpose},
		},
		HostConfig: &container.HostConfig{Mounts: mounts, NetworkMode: "none"},
	})
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = c.api.ContainerRemove(context.Background(), res.ID, client.ContainerRemoveOptions{Force: true})
	}()
	if _, err := c.api.ContainerStart(ctx, res.ID, client.ContainerStartOptions{}); err != nil {
		return "", err
	}
	code, err := c.WaitExit(ctx, res.ID)
	var out bytes.Buffer
	_ = c.Logs(context.Background(), res.ID, false, &out)
	if err != nil {
		return out.String(), err
	}
	if code != 0 {
		return out.String(), fmt.Errorf("helper exited with code %d: %s", code, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// ensureImage pulls ref if it is not present locally.
func (c *Client) ensureImage(ctx context.Context, ref string) error {
	if _, err := c.api.ImageInspect(ctx, ref); err == nil {
		return nil
	}
	pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return c.PullImage(pullCtx, ref, io.Discard)
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

// ListVolumes returns the names of volumes carrying all labels.
func (c *Client) ListVolumes(ctx context.Context, labels map[string]string) ([]string, error) {
	f := make(client.Filters)
	for k, v := range labels {
		f.Add("label", k+"="+v)
	}
	res, err := c.api.VolumeList(ctx, client.VolumeListOptions{Filters: f})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(res.Items))
	for _, v := range res.Items {
		out = append(out, v.Name)
	}
	return out, nil
}
