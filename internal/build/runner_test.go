package build

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/buildlog"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/gitx"
)

type memBuilds struct {
	mu     sync.Mutex
	builds map[int64]*domain.Build
	images []domain.BuildImage
}

func newMemBuilds(bs ...domain.Build) *memBuilds {
	m := &memBuilds{builds: map[int64]*domain.Build{}}
	for i := range bs {
		b := bs[i]
		m.builds[b.ID] = &b
	}
	return m
}

func (m *memBuilds) GetBuild(_ context.Context, id int64) (domain.Build, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.builds[id]
	if !ok {
		return domain.Build{}, domain.ErrNotFound
	}
	return *b, nil
}

func (m *memBuilds) ListBuildsByStatus(_ context.Context, s domain.BuildStatus) ([]domain.Build, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Build
	for id := int64(1); id <= int64(len(m.builds)); id++ {
		if b, ok := m.builds[id]; ok && b.Status == s {
			out = append(out, *b)
		}
	}
	return out, nil
}

func (m *memBuilds) move(id int64, to domain.BuildStatus) (*domain.Build, error) {
	b, ok := m.builds[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if !b.Status.CanTransition(to) {
		return nil, domain.ErrConflict
	}
	b.Status = to
	return b, nil
}

func (m *memBuilds) StartBuild(_ context.Context, id int64, sha string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, err := m.move(id, domain.BuildRunning)
	if err == nil {
		b.CommitSHA = sha
	}
	return err
}

func (m *memBuilds) SetBuildCommit(_ context.Context, id int64, sha string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.builds[id].CommitSHA = sha
	return nil
}

func (m *memBuilds) FinishBuild(_ context.Context, id int64, to domain.BuildStatus, msg string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, err := m.move(id, to)
	if err == nil {
		b.Error = msg
	}
	return err
}

func (m *memBuilds) AddBuildImage(_ context.Context, img domain.BuildImage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.images = append(m.images, img)
	return nil
}

func (m *memBuilds) status(id int64) domain.BuildStatus {
	b, _ := m.GetBuild(context.Background(), id)
	return b.Status
}

type fakeApps struct{}

func (fakeApps) Get(_ context.Context, id int64) (domain.App, error) {
	return domain.App{ID: id, Name: "shop", RepoURL: "git@h:o/shop.git"}, nil
}

type fakeSpecs struct{ yaml string }

func (f fakeSpecs) Latest(context.Context, int64) (domain.AppSpecVersion, error) {
	if f.yaml == "" {
		return domain.AppSpecVersion{}, domain.ErrNotFound
	}
	return domain.AppSpecVersion{Version: 3, YAML: f.yaml}, nil
}

type fakeKeys struct{}

func (fakeKeys) PrivateKey(context.Context, int64) ([]byte, error) { return []byte("pem"), nil }

type fakeGit struct{ dir string }

func (f fakeGit) Checkout(context.Context, string, string, []byte, string) (gitx.Checkout, error) {
	return gitx.Checkout{Dir: f.dir, CommitSHA: "abc123"}, nil
}

type fakeDocker struct {
	mu      sync.Mutex
	pulled  []string
	built   []dockerx.BuildRequest
	block   chan struct{} // when set, BuildImage waits for ctx or this channel
	started chan struct{}
	fail    error
	active  int
	maxSeen int
}

func (f *fakeDocker) PullImage(_ context.Context, ref string, _ io.Writer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pulled = append(f.pulled, ref)
	return nil
}

func (f *fakeDocker) BuildImage(ctx context.Context, req dockerx.BuildRequest, logs io.Writer) (string, error) {
	_, _ = io.Copy(io.Discard, req.Context)
	f.mu.Lock()
	f.built = append(f.built, req)
	f.active++
	f.maxSeen = max(f.maxSeen, f.active)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.active--
		f.mu.Unlock()
	}()
	_, _ = io.WriteString(logs, "#1 building "+req.Tags[0]+"\n")
	if f.started != nil {
		f.started <- struct{}{}
	}
	if f.block != nil {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-f.block:
		}
	}
	if f.fail != nil {
		return "", f.fail
	}
	return "sha256:" + req.Tags[0], nil
}

const runnerSpec = `
services:
  api:
    kind: deployment
    strategy: recreate
    build: .
  db:
    kind: stateful
    image: mysql:8.4
`

type fixture struct {
	runner *Runner
	builds *memBuilds
	docker *fakeDocker
	logs   buildlog.Store
	cancel context.CancelFunc
}

func newFixture(t *testing.T, builds *memBuilds, docker *fakeDocker, workers int) *fixture {
	return newFixtureWithSpec(t, builds, docker, workers, runnerSpec)
}

func newFixtureWithSpec(t *testing.T, builds *memBuilds, docker *fakeDocker, workers int, specYAML string) *fixture {
	t.Helper()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "Dockerfile"), []byte("FROM golang:1.27\nRUN true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logs := buildlog.Store{Dir: t.TempDir()}
	r := &Runner{
		Builds: builds, Apps: fakeApps{}, Specs: fakeSpecs{yaml: specYAML}, Keys: fakeKeys{},
		Git: fakeGit{dir: repo}, Docker: docker, Logs: logs, Hub: buildlog.NewHub(),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: time.Now, Workers: workers,
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := r.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); r.Wait() })
	return &fixture{runner: r, builds: builds, docker: docker, logs: logs, cancel: cancel}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func queued(id int64) domain.Build {
	return domain.Build{ID: id, AppID: 1, Ref: "main", Status: domain.BuildQueued}
}

func TestRunnerSuccess(t *testing.T) {
	builds := newMemBuilds(queued(1))
	f := newFixture(t, builds, &fakeDocker{}, 1)
	waitFor(t, "build to succeed", func() bool { return builds.status(1) == domain.BuildSucceeded })

	if len(f.docker.built) != 1 || f.docker.built[0].Tags[0] != "dkt/shop/api:abc123" {
		t.Fatalf("built %+v", f.docker.built)
	}
	if len(f.docker.pulled) != 1 || f.docker.pulled[0] != "golang:1.27" {
		t.Fatalf("pulled %v", f.docker.pulled)
	}
	if len(builds.images) != 1 || builds.images[0].ImageID != "sha256:dkt/shop/api:abc123" {
		t.Fatalf("images %+v", builds.images)
	}
	if b, _ := builds.GetBuild(context.Background(), 1); b.CommitSHA != "abc123" {
		t.Fatalf("commit not recorded: %+v", b)
	}
	log, _ := f.logs.Read(1)
	for _, want := range []string{"Checked out main at abc123 using spec v3", "==> Building api", "#1 building", "Build succeeded."} {
		if !strings.Contains(string(log), want) {
			t.Errorf("log missing %q:\n%s", want, log)
		}
	}
}

func TestRunnerFailure(t *testing.T) {
	builds := newMemBuilds(queued(1))
	newFixture(t, builds, &fakeDocker{fail: errors.New("RUN exited with 1")}, 1)
	waitFor(t, "build to fail", func() bool { return builds.status(1) == domain.BuildFailed })
	if b, _ := builds.GetBuild(context.Background(), 1); !strings.Contains(b.Error, "service api: RUN exited with 1") {
		t.Fatalf("error %q", b.Error)
	}
}

func TestRunnerCancelRunningAndQueued(t *testing.T) {
	builds := newMemBuilds(queued(1), queued(2))
	docker := &fakeDocker{block: make(chan struct{}), started: make(chan struct{}, 2)}
	f := newFixture(t, builds, docker, 1)

	<-docker.started
	if err := f.runner.Cancel(context.Background(), 2); err != nil {
		t.Fatalf("cancel queued: %v", err)
	}
	if err := f.runner.Cancel(context.Background(), 1); err != nil {
		t.Fatalf("cancel running: %v", err)
	}
	waitFor(t, "running build canceled", func() bool { return builds.status(1) == domain.BuildCanceled })
	if builds.status(2) != domain.BuildCanceled {
		t.Fatalf("queued build: %s", builds.status(2))
	}
	if err := f.runner.Cancel(context.Background(), 1); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("cancel finished build: %v", err)
	}
	if len(docker.built) != 1 {
		t.Fatal("a canceled queued build must not run")
	}
}

func TestRunnerRecoversAfterCrash(t *testing.T) {
	crashed := queued(1)
	crashed.Status = domain.BuildRunning
	builds := newMemBuilds(crashed, queued(2))
	newFixture(t, builds, &fakeDocker{}, 1)

	waitFor(t, "queued build to run", func() bool { return builds.status(2) == domain.BuildSucceeded })
	if b, _ := builds.GetBuild(context.Background(), 1); b.Status != domain.BuildFailed || !strings.Contains(b.Error, "interrupted") {
		t.Fatalf("crashed build: %+v", b)
	}
}

func TestRunnerConcurrencyLimit(t *testing.T) {
	builds := newMemBuilds(queued(1), queued(2), queued(3))
	docker := &fakeDocker{block: make(chan struct{}), started: make(chan struct{}, 3)}
	newFixture(t, builds, docker, 2)

	<-docker.started
	<-docker.started
	time.Sleep(50 * time.Millisecond)
	docker.mu.Lock()
	active := docker.active
	docker.mu.Unlock()
	if active != 2 {
		t.Fatalf("active builds %d, want 2", active)
	}
	close(docker.block)
	waitFor(t, "all builds", func() bool { return builds.status(3) == domain.BuildSucceeded })
	if docker.maxSeen != 2 {
		t.Fatalf("max concurrent builds %d, want 2", docker.maxSeen)
	}
}

func TestRunnerFailsWithoutSpec(t *testing.T) {
	builds := newMemBuilds(queued(1))
	newFixtureWithSpec(t, builds, &fakeDocker{}, 1, "")
	waitFor(t, "build to fail", func() bool { return builds.status(1) == domain.BuildFailed })
	if b, _ := builds.GetBuild(context.Background(), 1); !strings.Contains(b.Error, "no spec") {
		t.Fatalf("error %q", b.Error)
	}
}

func TestRunnerFailsWithInvalidSpec(t *testing.T) {
	builds := newMemBuilds(queued(1))
	newFixtureWithSpec(t, builds, &fakeDocker{}, 1, "services:\n  a:\n    kind: daemon\n")
	waitFor(t, "build to fail", func() bool { return builds.status(1) == domain.BuildFailed })
	if b, _ := builds.GetBuild(context.Background(), 1); !strings.Contains(b.Error, "spec v3 is invalid") {
		t.Fatalf("error %q", b.Error)
	}
}
