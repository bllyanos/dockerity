package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/buildlog"
	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/routing"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// fakeDocker keeps containers in memory and records operations.
type fakeDocker struct {
	mu         sync.Mutex
	containers map[string]*dockerx.ContainerInfo
	ops        []string
	next       int
	unhealthy  map[string]bool // service -> fails health check
	exitCodes  map[string]int  // service -> job exit code
}

func newFakeDocker() *fakeDocker {
	return &fakeDocker{containers: map[string]*dockerx.ContainerInfo{}, unhealthy: map[string]bool{}, exitCodes: map[string]int{}}
}

func (f *fakeDocker) op(format string, args ...any) {
	f.ops = append(f.ops, fmt.Sprintf(format, args...))
}

func (f *fakeDocker) EnsureNetwork(_ context.Context, name string, _ map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.op("network %s", name)
	return nil
}

func (f *fakeDocker) EnsureVolume(_ context.Context, name string, _ map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.op("volume %s", name)
	return nil
}

func (f *fakeDocker) CreateContainer(_ context.Context, s containerspec.Spec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	id := fmt.Sprintf("c%d", f.next)
	f.containers[id] = &dockerx.ContainerInfo{ID: id, Name: s.Name, Image: s.Image, Labels: maps.Clone(s.Labels)}
	f.op("create %s", s.Name)
	return id, nil
}

func (f *fakeDocker) StartContainer(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.containers[id].Running = true
	f.op("start %s", f.containers[id].Name)
	return nil
}

func (f *fakeDocker) StopContainer(_ context.Context, id string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.containers[id]; ok {
		c.Running = false
		f.op("stop %s", c.Name)
	}
	return nil
}

func (f *fakeDocker) RemoveContainer(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.containers[id]; ok {
		f.op("remove %s", c.Name)
		delete(f.containers, id)
	}
	return nil
}

func (f *fakeDocker) WaitHealthy(_ context.Context, id string, s containerspec.Spec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.unhealthy[s.Labels[domain.LabelService]] {
		return errors.New("did not become ready")
	}
	return nil
}

func (f *fakeDocker) WaitExit(_ context.Context, id string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := f.containers[id]
	c.Running = false
	return f.exitCodes[c.Labels[domain.LabelService]], nil
}

func (f *fakeDocker) Logs(_ context.Context, id string, _ bool, w io.Writer) error {
	f.mu.Lock()
	name := ""
	if c, ok := f.containers[id]; ok {
		name = c.Name
	}
	f.mu.Unlock()
	_, _ = fmt.Fprintf(w, "output of %s\n", name)
	return nil
}

func (f *fakeDocker) ListManaged(_ context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dockerx.ContainerInfo
	for i := 1; i <= f.next; i++ {
		c, ok := f.containers[fmt.Sprintf("c%d", i)]
		if !ok {
			continue
		}
		match := true
		for k, v := range labels {
			if c.Labels[k] != v {
				match = false
			}
		}
		if match {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (f *fakeDocker) running(service string) []string {
	list, _ := f.ListManaged(context.Background(), map[string]string{domain.LabelService: service})
	var out []string
	for _, c := range list {
		if c.Running {
			out = append(out, c.Name)
		}
	}
	return out
}

// memStore implements Store in memory.
type memStore struct {
	mu    sync.Mutex
	deps  map[int64]*domain.Deployment
	steps map[int64][]domain.DeploymentStep
}

func (m *memStore) GetDeployment(_ context.Context, id int64) (domain.Deployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return *m.deps[id], nil
}

func (m *memStore) CurrentDeployment(_ context.Context, envID int64) (domain.Deployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var cur *domain.Deployment
	for _, d := range m.deps {
		if d.EnvID == envID && d.Status == domain.DeploymentSucceeded && (cur == nil || d.ID > cur.ID) {
			cur = d
		}
	}
	if cur == nil {
		return domain.Deployment{}, domain.ErrNotFound
	}
	return *cur, nil
}

func (m *memStore) ListDeploymentsByStatus(_ context.Context, s domain.DeploymentStatus) ([]domain.Deployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Deployment
	for _, d := range m.deps {
		if d.Status == s {
			out = append(out, *d)
		}
	}
	return out, nil
}

func (m *memStore) move(id int64, to domain.DeploymentStatus) error {
	d := m.deps[id]
	if !d.Status.CanTransition(to) {
		return domain.ErrConflict
	}
	d.Status = to
	return nil
}

func (m *memStore) StartDeployment(_ context.Context, id int64, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.move(id, domain.DeploymentRunning)
}

func (m *memStore) FinishDeployment(_ context.Context, id int64, to domain.DeploymentStatus, msg string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deps[id].Error = msg
	return m.move(id, to)
}

func (m *memStore) InsertSteps(_ context.Context, id int64, steps []domain.DeploymentStep) ([]domain.DeploymentStep, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range steps {
		steps[i].ID, steps[i].Position, steps[i].Status = int64(i+1), i+1, domain.StepPending
	}
	m.steps[id] = steps
	return steps, nil
}

func (m *memStore) setStep(stepID int64, status domain.StepStatus, code *int) {
	for depID, steps := range m.steps {
		for i := range steps {
			if steps[i].ID == stepID {
				m.steps[depID][i].Status = status
				m.steps[depID][i].ExitCode = code
			}
		}
	}
}

func (m *memStore) StartStep(_ context.Context, stepID int64, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setStep(stepID, domain.StepRunning, nil)
	return nil
}

func (m *memStore) FinishStep(_ context.Context, stepID int64, s domain.StepStatus, code *int, _ string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setStep(stepID, s, code)
	return nil
}

func (m *memStore) SkipPendingSteps(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.steps[id] {
		if m.steps[id][i].Status == domain.StepPending {
			m.steps[id][i].Status = domain.StepSkipped
		}
	}
	return nil
}

type staticEnv struct{}

func (staticEnv) Get(_ context.Context, id int64) (domain.Environment, error) {
	return domain.Environment{ID: id, AppID: 1, AppName: "shop", Name: "staging", Domain: "staging.shop.test"}, nil
}

type specVersions map[int]string

func (s specVersions) Get(_ context.Context, _ int64, v int) (domain.AppSpecVersion, error) {
	return domain.AppSpecVersion{Version: v, YAML: s[v]}, nil
}

type buildImages map[int64][]domain.BuildImage

func (b buildImages) ListBuildImages(_ context.Context, id int64) ([]domain.BuildImage, error) {
	return b[id], nil
}

type secretValues map[string]string

func (s secretValues) Values(context.Context, secrets.Scope, int) (map[string]string, error) {
	return s, nil
}

const appSpec = `
services:
  db:
    kind: stateful
    image: mysql:8.4
    volumes:
      data: /var/lib/mysql
  migrate:
    kind: job
    build: .
    depends_on: [db]
    secrets: [DATABASE_URL]
  web:
    kind: deployment
    strategy: recreate
    build: .
    depends_on: [db]
    secrets: [DATABASE_URL]
    http:
      port: 8080
`

// fakeRoutes records the routed containers of each write into the Docker op log.
type fakeRoutes struct {
	docker *fakeDocker
	writes []string
}

func (r *fakeRoutes) Write(app, env string, data []byte) error {
	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "url: http://"); i >= 0 {
			names = append(names, strings.SplitN(line[i+len("url: http://"):], ":", 2)[0])
		}
	}
	entry := "routes [" + strings.Join(names, " ") + "]"
	r.writes = append(r.writes, entry)
	r.docker.mu.Lock()
	r.docker.op("%s", entry)
	r.docker.mu.Unlock()
	return nil
}

func (r *fakeRoutes) Remove(string, string) error { return nil }

type fixture struct {
	exec   *Executor
	routes *fakeRoutes
	store  *memStore
	docker *fakeDocker
	logs   buildlog.Store
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		store:  &memStore{deps: map[int64]*domain.Deployment{}, steps: map[int64][]domain.DeploymentStep{}},
		docker: newFakeDocker(),
		logs:   buildlog.Store{Dir: t.TempDir()},
	}
	f.exec = &Executor{
		Store: f.store, Envs: staticEnv{},
		Specs: specVersions{1: appSpec, 2: strings.Replace(appSpec, "  web:", "  api:", 1), 3: blueGreenSpec},
		Builds: buildImages{
			1: {{Service: "migrate", ImageTag: "dkt/shop/migrate:a"}, {Service: "web", ImageTag: "dkt/shop/web:a"}},
			2: {{Service: "migrate", ImageTag: "dkt/shop/migrate:b"}, {Service: "web", ImageTag: "dkt/shop/web:b"}, {Service: "api", ImageTag: "dkt/shop/api:b"}},
		},
		Secrets: secretValues{"DATABASE_URL": "mysql://db"},
		Docker:  f.docker, Logs: f.logs, Hub: buildlog.NewHub(), Routing: routing.DefaultOptions,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: time.Now,
	}
	return f
}

// withRoutes makes the fixture record route writes in the op log.
func (f *fixture) withRoutes() *fixture {
	f.routes = &fakeRoutes{docker: f.docker}
	f.exec.Routes = f.routes
	return f
}

func (f *fixture) deploy(id, buildID int64, specVersion int) domain.Deployment {
	f.store.mu.Lock()
	f.store.deps[id] = &domain.Deployment{ID: id, EnvID: 1, BuildID: buildID, SpecVersion: specVersion, SecretVersion: 1, Status: domain.DeploymentPending}
	f.store.mu.Unlock()
	f.exec.Run(context.Background(), id)
	d, _ := f.store.GetDeployment(context.Background(), id)
	return d
}

func TestFirstDeploy(t *testing.T) {
	f := newFixture(t)
	d := f.deploy(1, 1, 1)
	if d.Status != domain.DeploymentSucceeded {
		t.Fatalf("status %s: %s", d.Status, d.Error)
	}
	want := []string{
		"network dkt-shop-staging", "volume dkt-shop-staging-db-data",
		"create dkt-shop-staging-db-d1", "start dkt-shop-staging-db-d1",
		"create dkt-shop-staging-migrate-d1", "start dkt-shop-staging-migrate-d1", "remove dkt-shop-staging-migrate-d1",
		"create dkt-shop-staging-web-d1", "start dkt-shop-staging-web-d1",
	}
	if got := strings.Join(f.docker.ops, "\n"); got != strings.Join(want, "\n") {
		t.Fatalf("ops:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
	log, _ := f.logs.Read(1)
	for _, s := range []string{"[1/6] ensure_network", "output of dkt-shop-staging-migrate-d1", "Job migrate exited with code 0", "Deployment succeeded."} {
		if !strings.Contains(string(log), s) {
			t.Errorf("log missing %q:\n%s", s, log)
		}
	}
	for _, st := range f.store.steps[1] {
		if st.Status != domain.StepSucceeded {
			t.Fatalf("step %+v", st)
		}
	}
}

func TestRedeployRecreatesAndKeepsStateful(t *testing.T) {
	f := newFixture(t)
	f.deploy(1, 1, 1)
	f.docker.ops = nil
	d := f.deploy(2, 1, 1)
	if d.Status != domain.DeploymentSucceeded {
		t.Fatalf("status %s: %s", d.Status, d.Error)
	}
	ops := strings.Join(f.docker.ops, "\n")
	if strings.Contains(ops, "db-d2") {
		t.Fatalf("unchanged stateful service was recreated:\n%s", ops)
	}
	if !strings.Contains(ops, "stop dkt-shop-staging-web-d1\nremove dkt-shop-staging-web-d1\ncreate dkt-shop-staging-web-d2") {
		t.Fatalf("recreate order wrong:\n%s", ops)
	}
	if got := f.docker.running("web"); len(got) != 1 || got[0] != "dkt-shop-staging-web-d2" {
		t.Fatalf("running web %v", got)
	}
}

func TestFailingPreDeployJobLeavesServicesUntouched(t *testing.T) {
	f := newFixture(t)
	f.deploy(1, 1, 1)
	f.docker.exitCodes["migrate"] = 1
	f.docker.ops = nil
	d := f.deploy(2, 1, 1)
	if d.Status != domain.DeploymentFailed || !strings.Contains(d.Error, "job migrate exited with code 1") {
		t.Fatalf("status %s: %s", d.Status, d.Error)
	}
	if got := f.docker.running("web"); len(got) != 1 || got[0] != "dkt-shop-staging-web-d1" {
		t.Fatalf("the running web container must be untouched, got %v", got)
	}
	for _, op := range f.docker.ops {
		if strings.Contains(op, "web") {
			t.Fatalf("web was touched: %v", f.docker.ops)
		}
	}
	steps := f.store.steps[2]
	last := steps[len(steps)-1]
	if last.Status != domain.StepSkipped {
		t.Fatalf("remaining steps must be skipped: %+v", steps)
	}
	for _, st := range steps {
		if st.Service == "migrate" && (st.Status != domain.StepFailed || st.ExitCode == nil || *st.ExitCode != 1) {
			t.Fatalf("job step %+v", st)
		}
	}
}

func TestUnhealthyServiceFailsDeploy(t *testing.T) {
	f := newFixture(t)
	f.docker.unhealthy["web"] = true
	d := f.deploy(1, 1, 1)
	if d.Status != domain.DeploymentFailed || !strings.Contains(d.Error, "deploy web: did not become ready") {
		t.Fatalf("status %s: %s", d.Status, d.Error)
	}
	log, _ := f.logs.Read(1)
	if !strings.Contains(string(log), "its last output") {
		t.Fatalf("container output not logged:\n%s", log)
	}
}

func TestMissingSecretFailsBeforeStarting(t *testing.T) {
	f := newFixture(t)
	f.exec.Secrets = secretValues{}
	d := f.deploy(1, 1, 1)
	if d.Status != domain.DeploymentFailed || !strings.Contains(d.Error, "DATABASE_URL") || len(f.docker.ops) != 0 {
		t.Fatalf("status %s: %s, ops %v", d.Status, d.Error, f.docker.ops)
	}
}

func TestFinishRemovesServicesDroppedFromSpec(t *testing.T) {
	f := newFixture(t)
	f.deploy(1, 1, 1)
	d := f.deploy(2, 2, 2) // spec v2 renames web to api
	if d.Status != domain.DeploymentSucceeded {
		t.Fatalf("status %s: %s", d.Status, d.Error)
	}
	if got := f.docker.running("web"); len(got) != 0 {
		t.Fatalf("web should be removed, got %v", got)
	}
	if got := f.docker.running("api"); len(got) != 1 {
		t.Fatalf("api should run, got %v", got)
	}
}

func TestRunnerRecoversInterruptedDeployments(t *testing.T) {
	f := newFixture(t)
	f.store.deps[1] = &domain.Deployment{ID: 1, Status: domain.DeploymentRunning}
	f.store.deps[2] = &domain.Deployment{ID: 2, Status: domain.DeploymentPending}
	r := &Runner{Exec: f.exec}
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for id := int64(1); id <= 2; id++ {
		if d, _ := f.store.GetDeployment(context.Background(), id); d.Status != domain.DeploymentFailed || !strings.Contains(d.Error, "interrupted") {
			t.Fatalf("deployment %d: %+v", id, d)
		}
	}
}

func TestRoutesOnlyEverContainHealthyContainers(t *testing.T) {
	f := newFixture(t).withRoutes()
	f.deploy(1, 1, 1)
	f.docker.ops = nil
	f.deploy(2, 1, 1)
	ops := strings.Join(f.docker.ops, "\n")
	// Recreate: the old container leaves routing before it stops; the new one
	// is routed only after it started and passed its health check.
	want := "routes []\nstop dkt-shop-staging-web-d1\nremove dkt-shop-staging-web-d1\n" +
		"create dkt-shop-staging-web-d2\nstart dkt-shop-staging-web-d2\nroutes [dkt-shop-staging-web-d2]"
	if !strings.Contains(ops, want) {
		t.Fatalf("route order wrong:\n%s", ops)
	}
	if last := f.routes.writes[len(f.routes.writes)-1]; last != "routes [dkt-shop-staging-web-d2]" {
		t.Fatalf("final routes %s", last)
	}
}

func TestUnhealthyContainerIsNeverRouted(t *testing.T) {
	f := newFixture(t).withRoutes()
	f.docker.unhealthy["web"] = true
	f.deploy(1, 1, 1)
	for _, w := range f.routes.writes {
		if strings.Contains(w, "web-d1") {
			t.Fatalf("unhealthy container routed: %v", f.routes.writes)
		}
	}
	if got := f.docker.running("web"); len(got) != 0 {
		t.Fatalf("unhealthy container must be removed, running %v", got)
	}
}

func TestRunnerRebuildsRoutesOnStart(t *testing.T) {
	f := newFixture(t).withRoutes()
	f.deploy(1, 1, 1)
	f.exec.AllEnvs = allEnvs{}
	f.routes.writes = nil
	if err := (&Runner{Exec: f.exec}).Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.routes.writes) != 1 || f.routes.writes[0] != "routes [dkt-shop-staging-web-d1]" {
		t.Fatalf("rebuilt routes %v", f.routes.writes)
	}
}

type allEnvs struct{}

func (allEnvs) ListAllEnvironments(ctx context.Context) ([]domain.Environment, error) {
	e, _ := staticEnv{}.Get(ctx, 1)
	return []domain.Environment{e}, nil
}

const blueGreenSpec = `
services:
  web:
    kind: deployment
    image: shop-web
    http:
      port: 8080
    healthcheck:
      http: /healthz
    drain: 2s
    stop_grace_period: 7s
`

func (f *fixture) recordSleeps() {
	f.exec.Sleep = func(_ context.Context, d time.Duration) error {
		f.docker.mu.Lock()
		f.docker.op("drain %s", d)
		f.docker.mu.Unlock()
		return nil
	}
}

func TestBlueGreenSwitch(t *testing.T) {
	f := newFixture(t).withRoutes()
	f.recordSleeps()
	if d := f.deploy(1, 1, 3); d.Status != domain.DeploymentSucceeded {
		t.Fatalf("first deploy: %s", d.Error)
	}
	f.docker.ops = nil
	if d := f.deploy(2, 1, 3); d.Status != domain.DeploymentSucceeded {
		t.Fatalf("second deploy: %s", d.Error)
	}
	want := strings.Join([]string{
		"create dkt-shop-staging-web-d2",
		"start dkt-shop-staging-web-d2",
		"routes [dkt-shop-staging-web-d1 dkt-shop-staging-web-d2]", // both serve
		"routes [dkt-shop-staging-web-d2]",                         // old leaves routing
		"drain 2s",
		"stop dkt-shop-staging-web-d1",
		"remove dkt-shop-staging-web-d1",
	}, "\n")
	if ops := strings.Join(f.docker.ops, "\n"); !strings.Contains(ops, want) {
		t.Fatalf("blue-green order wrong:\n%s\nwant:\n%s", ops, want)
	}
	log, _ := f.logs.Read(2)
	if !strings.Contains(string(log), "Blue-green: dkt-shop-staging-web-d2 replaces dkt-shop-staging-web-d1") ||
		!strings.Contains(string(log), "Stopping dkt-shop-staging-web-d1 (grace 7s)") {
		t.Fatalf("log:\n%s", log)
	}
}

func TestBlueGreenAbortKeepsOldServing(t *testing.T) {
	f := newFixture(t).withRoutes()
	f.recordSleeps()
	f.deploy(1, 1, 3)
	f.docker.unhealthy["web"] = true
	f.routes.writes = nil
	d := f.deploy(2, 1, 3)
	if d.Status != domain.DeploymentFailed {
		t.Fatalf("status %s", d.Status)
	}
	for _, w := range f.routes.writes {
		if strings.Contains(w, "web-d2") || !strings.Contains(w, "web-d1") {
			t.Fatalf("routes during abort: %v", f.routes.writes)
		}
	}
	if got := f.docker.running("web"); len(got) != 1 || got[0] != "dkt-shop-staging-web-d1" {
		t.Fatalf("old version must keep running alone, got %v", got)
	}
	log, _ := f.logs.Read(2)
	if !strings.Contains(string(log), "Aborted: dkt-shop-staging-web-d1 keeps serving") {
		t.Fatalf("log:\n%s", log)
	}
}
