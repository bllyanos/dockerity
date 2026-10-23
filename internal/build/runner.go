package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/bllyanos/dockerity/internal/buildlog"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/gitx"
	"github.com/bllyanos/dockerity/internal/spec"
)

// Store persists builds.
type Store interface {
	GetBuild(ctx context.Context, id int64) (domain.Build, error)
	ListBuildsByStatus(ctx context.Context, status domain.BuildStatus) ([]domain.Build, error)
	StartBuild(ctx context.Context, id int64, commitSHA string, at time.Time) error
	SetBuildCommit(ctx context.Context, id int64, commitSHA string) error
	FinishBuild(ctx context.Context, id int64, to domain.BuildStatus, errMsg string, at time.Time) error
	AddBuildImage(ctx context.Context, img domain.BuildImage) error
}

// Git checks out an app's repo.
type Git interface {
	Checkout(ctx context.Context, app, repoURL string, privatePEM []byte, ref string) (gitx.Checkout, error)
}

// Docker pulls and builds images.
type Docker interface {
	PullImage(ctx context.Context, ref string, logs io.Writer) error
	BuildImage(ctx context.Context, req dockerx.BuildRequest, logs io.Writer) (string, error)
}

// Runner executes queued builds with a bounded number of workers.
type Runner struct {
	Builds Store
	Apps   interface {
		Get(ctx context.Context, id int64) (domain.App, error)
	}
	Specs interface {
		Latest(ctx context.Context, appID int64) (domain.AppSpecVersion, error)
	}
	Keys interface {
		PrivateKey(ctx context.Context, appID int64) ([]byte, error)
	}
	Git    Git
	Docker Docker
	Logs   buildlog.Store
	Hub    *buildlog.Hub
	Logger *slog.Logger
	Now    func() time.Time
	// Workers is how many builds run at once (default 1).
	Workers int

	queue   chan int64
	mu      sync.Mutex
	running map[int64]context.CancelFunc
	wg      sync.WaitGroup
}

// queueSize bounds pending builds kept in memory.
const queueSize = 1024

// ErrQueueFull is returned when too many builds are waiting.
var ErrQueueFull = errors.New("build queue is full, try again later")

// Start recovers from a previous crash, re-queues waiting builds and starts
// the workers. They stop when ctx is cancelled; Wait blocks until they did.
func (r *Runner) Start(ctx context.Context) error {
	r.queue = make(chan int64, queueSize)
	r.running = map[int64]context.CancelFunc{}

	interrupted, err := r.Builds.ListBuildsByStatus(ctx, domain.BuildRunning)
	if err != nil {
		return err
	}
	for _, b := range interrupted {
		if err := r.Builds.FinishBuild(ctx, b.ID, domain.BuildFailed, "interrupted: Dockerity restarted during the build", r.Now().UTC()); err != nil {
			return err
		}
		r.Logger.Warn("marked interrupted build as failed", "build", b.ID)
	}
	queued, err := r.Builds.ListBuildsByStatus(ctx, domain.BuildQueued)
	if err != nil {
		return err
	}
	for _, b := range queued {
		if err := r.Enqueue(b.ID); err != nil {
			return err
		}
	}

	workers := max(r.Workers, 1)
	for range workers {
		r.wg.Go(func() { r.work(ctx) })
	}
	return nil
}

// Wait blocks until all workers stopped.
func (r *Runner) Wait() { r.wg.Wait() }

// Enqueue schedules a queued build.
func (r *Runner) Enqueue(buildID int64) error {
	select {
	case r.queue <- buildID:
		return nil
	default:
		return ErrQueueFull
	}
}

// Cancel stops a running build, or marks a queued one canceled.
func (r *Runner) Cancel(ctx context.Context, buildID int64) error {
	// A queued build may start between the two checks, so look twice.
	for range 2 {
		if r.cancelRunning(buildID) {
			return nil
		}
		err := r.Builds.FinishBuild(ctx, buildID, domain.BuildCanceled, "", r.Now().UTC())
		if err == nil {
			r.Hub.Close(buildID)
			return nil
		}
		if !errors.Is(err, domain.ErrConflict) {
			return err
		}
	}
	if r.cancelRunning(buildID) {
		return nil
	}
	return domain.Conflictf("build %d already finished", buildID)
}

func (r *Runner) cancelRunning(buildID int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cancel, ok := r.running[buildID]
	if ok {
		cancel()
	}
	return ok
}

func (r *Runner) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-r.queue:
			r.run(ctx, id)
		}
	}
}

// run executes one build. Every outcome ends in a terminal status.
func (r *Runner) run(parent context.Context, id int64) {
	if err := r.Builds.StartBuild(parent, id, "", r.Now().UTC()); err != nil {
		// Canceled while queued, or picked up twice: nothing to do.
		if !errors.Is(err, domain.ErrConflict) {
			r.Logger.Error("start build", "build", id, "error", err)
		}
		return
	}
	ctx, cancel := context.WithCancel(parent)
	r.mu.Lock()
	r.running[id] = cancel
	r.mu.Unlock()
	defer func() {
		cancel()
		r.mu.Lock()
		delete(r.running, id)
		r.mu.Unlock()
	}()

	sink, err := r.Logs.OpenSink(id, r.Hub)
	if err != nil {
		r.finish(id, domain.BuildFailed, fmt.Errorf("open build log: %w", err))
		return
	}
	defer func() { _ = sink.Close() }()

	err = r.execute(ctx, id, sink)
	switch {
	case err == nil:
		_, _ = fmt.Fprintln(sink, "Build succeeded.")
		r.finish(id, domain.BuildSucceeded, nil)
	case ctx.Err() != nil && parent.Err() == nil:
		_, _ = fmt.Fprintln(sink, "Build canceled.")
		r.finish(id, domain.BuildCanceled, nil)
	default:
		_, _ = fmt.Fprintf(sink, "Build failed: %s\n", err)
		r.finish(id, domain.BuildFailed, err)
	}
}

func (r *Runner) finish(id int64, status domain.BuildStatus, cause error) {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	// Use a fresh context: the build's own context may be canceled already.
	if err := r.Builds.FinishBuild(context.Background(), id, status, msg, r.Now().UTC()); err != nil {
		r.Logger.Error("finish build", "build", id, "status", status, "error", err)
	}
}

func (r *Runner) execute(ctx context.Context, id int64, logs io.Writer) error {
	b, err := r.Builds.GetBuild(ctx, id)
	if err != nil {
		return err
	}
	app, err := r.Apps.Get(ctx, b.AppID)
	if err != nil {
		return err
	}
	specVersion, err := r.Specs.Latest(ctx, app.ID)
	if err != nil {
		return fmt.Errorf("the app has no spec: %w", err)
	}
	s, err := spec.Load([]byte(specVersion.YAML))
	if err != nil {
		return fmt.Errorf("spec v%d is invalid: %w", specVersion.Version, err)
	}
	key, err := r.Keys.PrivateKey(ctx, app.ID)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(logs, "Fetching %s and checking out %s\n", app.RepoURL, b.Ref)
	co, err := r.Git.Checkout(ctx, app.Name, app.RepoURL, key, b.Ref)
	if err != nil {
		return err
	}
	if err := r.Builds.SetBuildCommit(ctx, id, co.CommitSHA); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(logs, "Checked out %s at %s using spec v%d\n", b.Ref, co.CommitSHA, specVersion.Version)

	plan := Plan(app.Name, s, co.CommitSHA)
	if len(plan) == 0 {
		_, _ = fmt.Fprintln(logs, "No service has a build section; nothing to build.")
	}
	for _, sb := range plan {
		if err := r.buildService(ctx, id, co.Dir, sb, logs); err != nil {
			return fmt.Errorf("service %s: %w", sb.Service, err)
		}
	}
	return nil
}

func (r *Runner) buildService(ctx context.Context, id int64, repoDir string, sb ServiceBuild, logs io.Writer) error {
	_, _ = fmt.Fprintf(logs, "\n==> Building %s as %s\n", sb.Service, sb.ImageTag)
	ctxDir, err := ContextDir(repoDir, sb.Context)
	if err != nil {
		return err
	}
	if err := CheckDockerfile(ctxDir, sb.Dockerfile); err != nil {
		return err
	}
	bases, err := BaseImagesFromFile(filepath.Join(ctxDir, filepath.FromSlash(sb.Dockerfile)), sb.Args)
	if err != nil {
		return err
	}
	for _, img := range bases {
		_, _ = fmt.Fprintf(logs, "Pulling %s\n", img)
		if err := r.Docker.PullImage(ctx, img, logs); err != nil {
			return err
		}
	}
	tarball, err := Tar(ctxDir, sb.Dockerfile)
	if err != nil {
		return err
	}
	defer tarball.Close()
	imageID, err := r.Docker.BuildImage(ctx, dockerx.BuildRequest{
		Context: tarball, Dockerfile: sb.Dockerfile, Target: sb.Target,
		Args: sb.Args, Tags: []string{sb.ImageTag}, Labels: sb.Labels,
	}, logs)
	if err != nil {
		return err
	}
	return r.Builds.AddBuildImage(ctx, domain.BuildImage{BuildID: id, Service: sb.Service, ImageTag: sb.ImageTag, ImageID: imageID})
}
