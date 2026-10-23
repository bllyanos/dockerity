package api

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/bllyanos/dockerity/internal/apps"
	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// fakeSessions accepts "pw" for user "alice" and tracks live raw tokens.
type fakeSessions struct {
	live map[string]domain.User
	next int
}

func newFakeSessions() *fakeSessions { return &fakeSessions{live: map[string]domain.User{}} }

var alice = domain.User{ID: 1, Username: "alice", IsAdmin: true, PasswordHash: "secret-hash"}

func (f *fakeSessions) Login(ctx context.Context, username, pw string) (string, domain.Session, domain.User, error) {
	if username != "alice" || pw != "pw" {
		return "", domain.Session{}, domain.User{}, domain.ErrUnauthorized
	}
	raw, sess, err := f.Start(ctx, alice)
	return raw, sess, alice, err
}

func (f *fakeSessions) Start(_ context.Context, u domain.User) (string, domain.Session, error) {
	f.next++
	raw := "dkts_" + string(rune('a'+f.next))
	f.live[raw] = u
	return raw, domain.Session{UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (f *fakeSessions) Authenticate(_ context.Context, raw string) (auth.Principal, domain.Session, error) {
	u, ok := f.live[raw]
	if !ok {
		return auth.Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	return auth.Principal{User: u}, domain.Session{UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (f *fakeSessions) Logout(_ context.Context, raw string) error {
	delete(f.live, raw)
	return nil
}

// fakeAudit keeps events in memory, newest last.
type fakeAudit struct{ events []domain.AuditEvent }

func (f *fakeAudit) Record(_ context.Context, e domain.AuditEvent) error {
	e.ID = int64(len(f.events) + 1)
	f.events = append(f.events, e)
	return nil
}

func (f *fakeAudit) ListAudit(_ context.Context, before int64, limit int) ([]domain.AuditEvent, error) {
	var out []domain.AuditEvent
	for i := len(f.events) - 1; i >= 0 && len(out) < limit; i-- {
		if before <= 0 || f.events[i].ID < before {
			out = append(out, f.events[i])
		}
	}
	return out, nil
}

func (f *fakeAudit) actions() []string {
	var out []string
	for _, e := range f.events {
		out = append(out, e.Action)
	}
	return out
}

// fakeTokens issues "dkta_<n>" tokens for any user.
type fakeTokens struct {
	tokens []domain.APIToken
	raw    map[string]int64
	users  map[int64]domain.User
	used   map[int64]bool
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{raw: map[string]int64{}, users: map[int64]domain.User{}, used: map[int64]bool{}}
}

func (f *fakeTokens) Create(_ context.Context, u domain.User, name string) (string, domain.APIToken, error) {
	if name == "" {
		return "", domain.APIToken{}, domain.Invalid("name", "required")
	}
	t := domain.APIToken{ID: int64(len(f.tokens) + 1), UserID: u.ID, Name: name, TokenDigest: "digest"}
	f.tokens = append(f.tokens, t)
	raw := "dkta_" + string(rune('a'+t.ID))
	f.raw[raw] = t.ID
	f.users[u.ID] = u
	return raw, t, nil
}

func (f *fakeTokens) List(_ context.Context, userID int64) ([]domain.APIToken, error) {
	var out []domain.APIToken
	for _, t := range f.tokens {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeTokens) Revoke(_ context.Context, userID, id int64) (domain.APIToken, error) {
	for i, t := range f.tokens {
		if t.ID == id && t.UserID == userID && !t.Revoked() {
			now := time.Now()
			f.tokens[i].RevokedAt = &now
			return f.tokens[i], nil
		}
	}
	return domain.APIToken{}, domain.ErrNotFound
}

func (f *fakeTokens) Authenticate(_ context.Context, raw string) (auth.Principal, bool, error) {
	id, ok := f.raw[raw]
	if !ok || f.tokens[id-1].Revoked() {
		return auth.Principal{}, false, domain.ErrUnauthorized
	}
	first := !f.used[id]
	f.used[id] = true
	return auth.Principal{User: f.users[f.tokens[id-1].UserID], APITokenID: id}, first, nil
}

// fakeUserAdmin wraps a slice of users.
type fakeUserAdmin struct{ users []domain.User }

func (f *fakeUserAdmin) List(context.Context) ([]domain.User, error) { return f.users, nil }

func (f *fakeUserAdmin) Create(_ context.Context, username, _ string, isAdmin bool) (domain.User, error) {
	u := domain.User{ID: int64(len(f.users) + 1), Username: username, IsAdmin: isAdmin, PasswordHash: "secret-hash"}
	f.users = append(f.users, u)
	return u, nil
}

func (f *fakeUserAdmin) SetDisabled(_ context.Context, actor domain.User, id int64, disabled bool) (domain.User, error) {
	if actor.ID == id && disabled {
		return domain.User{}, domain.Invalid("id", "you cannot disable your own account")
	}
	if id < 1 || int(id) > len(f.users) {
		return domain.User{}, domain.ErrNotFound
	}
	if disabled {
		now := time.Now()
		f.users[id-1].DisabledAt = &now
	} else {
		f.users[id-1].DisabledAt = nil
	}
	return f.users[id-1], nil
}

func (f *fakeUserAdmin) ResetPassword(_ context.Context, id int64, _ string) (domain.User, error) {
	if id < 1 || int(id) > len(f.users) {
		return domain.User{}, domain.ErrNotFound
	}
	return f.users[id-1], nil
}

// fakeApps keeps apps in a slice and applies only the duplicate-name rule.
type fakeApps struct{ apps []domain.App }

func (f *fakeApps) List(context.Context) ([]domain.App, error) { return f.apps, nil }

func (f *fakeApps) Get(_ context.Context, id int64) (domain.App, error) {
	for _, a := range f.apps {
		if a.ID == id {
			return a, nil
		}
	}
	return domain.App{}, domain.ErrNotFound
}

func (f *fakeApps) Create(_ context.Context, in apps.CreateInput) (domain.App, error) {
	for _, a := range f.apps {
		if a.Name == in.Name {
			return domain.App{}, domain.ErrConflict
		}
	}
	if in.Name == "" {
		return domain.App{}, domain.Invalid("name", "required")
	}
	a := domain.App{ID: int64(len(f.apps) + 1), Name: in.Name, RepoURL: in.RepoURL, DefaultBranch: in.DefaultBranch, DeployPublicKey: "ssh-ed25519 AAAA"}
	f.apps = append(f.apps, a)
	return a, nil
}

func (f *fakeApps) Update(ctx context.Context, id int64, in apps.UpdateInput) (domain.App, error) {
	a, err := f.Get(ctx, id)
	if err != nil {
		return a, err
	}
	if in.DefaultBranch != nil {
		a.DefaultBranch = *in.DefaultBranch
	}
	f.apps[id-1] = a
	return a, nil
}

func (f *fakeApps) Delete(ctx context.Context, id int64) (domain.App, error) {
	return f.Get(ctx, id)
}

func (f *fakeApps) RegenerateDeployKey(ctx context.Context, id int64) (domain.App, error) {
	a, err := f.Get(ctx, id)
	if err != nil {
		return a, err
	}
	a.DeployPublicKey += "-regenerated"
	f.apps[id-1] = a
	return a, nil
}

// fakeTester reports success for app 1 and not found otherwise.
type fakeTester struct{}

func (fakeTester) Test(_ context.Context, id int64) (apps.ConnectionResult, error) {
	if id != 1 {
		return apps.ConnectionResult{}, domain.ErrNotFound
	}
	return apps.ConnectionResult{OK: true, DefaultBranchExists: true, BranchCount: 3, Message: "connected"}, nil
}

// fakeBuilds serves fixed builds: 1 is running, 2 has succeeded.
type fakeBuilds struct {
	triggered []string
	canceled  []int64
}

func (f *fakeBuilds) Trigger(_ context.Context, appID int64, ref string, _ *int64) (domain.Build, error) {
	if ref == "" {
		ref = "main"
	}
	f.triggered = append(f.triggered, ref)
	return domain.Build{ID: 3, AppID: appID, Ref: ref, Status: domain.BuildQueued}, nil
}

func (f *fakeBuilds) List(context.Context, int64) ([]domain.Build, error) {
	return []domain.Build{{ID: 2, Status: domain.BuildSucceeded}, {ID: 1, Status: domain.BuildRunning}}, nil
}

func (f *fakeBuilds) Get(_ context.Context, id int64) (domain.Build, []domain.BuildImage, error) {
	switch id {
	case 1:
		return domain.Build{ID: 1, Status: domain.BuildRunning}, nil, nil
	case 2:
		return domain.Build{ID: 2, Status: domain.BuildSucceeded, CommitSHA: "abc"},
			[]domain.BuildImage{{BuildID: 2, Service: "api", ImageTag: "dkt/shop/api:abc", ImageID: "sha256:1"}}, nil
	}
	return domain.Build{}, nil, domain.ErrNotFound
}

func (f *fakeBuilds) Cancel(ctx context.Context, id int64) (domain.Build, error) {
	b, _, err := f.Get(ctx, id)
	if err != nil {
		return b, err
	}
	if b.Status.Terminal() {
		return b, domain.Conflictf("already finished")
	}
	f.canceled = append(f.canceled, id)
	return b, nil
}

// fakeEnvs keeps environments for app 1.
type fakeEnvs struct{ envs []domain.Environment }

func (f *fakeEnvs) List(context.Context, int64) ([]domain.Environment, error) { return f.envs, nil }

func (f *fakeEnvs) Get(_ context.Context, id int64) (domain.Environment, error) {
	for _, e := range f.envs {
		if e.ID == id {
			return e, nil
		}
	}
	return domain.Environment{}, domain.ErrNotFound
}

func (f *fakeEnvs) CreatePermanent(_ context.Context, appID int64, name, host string) (domain.Environment, error) {
	e := domain.Environment{ID: int64(len(f.envs) + 1), AppID: appID, AppName: "shop", Name: name, Type: domain.EnvPermanent, Domain: host}
	f.envs = append(f.envs, e)
	return e, nil
}

func (f *fakeEnvs) UpdateDomain(ctx context.Context, id int64, host string) (domain.Environment, error) {
	e, err := f.Get(ctx, id)
	if err != nil {
		return e, err
	}
	e.Domain = host
	f.envs[id-1] = e
	return e, nil
}

func (f *fakeEnvs) Delete(ctx context.Context, id int64) (domain.Environment, error) {
	return f.Get(ctx, id)
}

// memSecretStore is an in-memory secrets.SecretStore.
type memSecretStore struct {
	versions map[secrets.Scope][]map[string]string
}

func newMemSecretStore() *memSecretStore {
	return &memSecretStore{versions: map[secrets.Scope][]map[string]string{}}
}

func (m *memSecretStore) Update(_ context.Context, s secrets.Scope, c secrets.Changes, _ *int64) (secrets.Version, error) {
	cur := map[string]string{}
	if vs := m.versions[s]; len(vs) > 0 {
		cur = vs[len(vs)-1]
	}
	next, err := c.Apply(cur)
	if err != nil {
		return secrets.Version{}, err
	}
	m.versions[s] = append(m.versions[s], next)
	return secrets.Version{Scope: s, Version: len(m.versions[s]), Keys: secrets.SortedKeys(next)}, nil
}

func (m *memSecretStore) Latest(_ context.Context, s secrets.Scope) (secrets.Version, error) {
	vs := m.versions[s]
	if len(vs) == 0 {
		return secrets.Version{}, domain.ErrNotFound
	}
	return secrets.Version{Scope: s, Version: len(vs), Keys: secrets.SortedKeys(vs[len(vs)-1])}, nil
}

func (m *memSecretStore) Versions(_ context.Context, s secrets.Scope) ([]secrets.Version, error) {
	var out []secrets.Version
	for i := len(m.versions[s]) - 1; i >= 0; i-- {
		out = append(out, secrets.Version{Scope: s, Version: i + 1, Keys: secrets.SortedKeys(m.versions[s][i])})
	}
	return out, nil
}

func (m *memSecretStore) Values(_ context.Context, s secrets.Scope, v int) (map[string]string, error) {
	return m.versions[s][v-1], nil
}

// fakeDeployments: environment 1 accepts one deployment at a time.
type fakeDeployments struct{ deps []domain.Deployment }

func (f *fakeDeployments) Start(_ context.Context, envID, buildID int64, _ *int64) (domain.Deployment, error) {
	for _, d := range f.deps {
		if !d.Status.Terminal() {
			return domain.Deployment{}, domain.Conflictf("a deployment is already in progress")
		}
	}
	d := domain.Deployment{ID: int64(len(f.deps) + 1), EnvID: envID, BuildID: buildID, SpecVersion: 1, Status: domain.DeploymentPending}
	f.deps = append(f.deps, d)
	return d, nil
}

func (f *fakeDeployments) List(context.Context, int64) ([]domain.Deployment, error) {
	return f.deps, nil
}

func (f *fakeDeployments) Get(_ context.Context, id int64) (domain.Deployment, []domain.DeploymentStep, error) {
	if id < 1 || int(id) > len(f.deps) {
		return domain.Deployment{}, nil, domain.ErrNotFound
	}
	return f.deps[id-1], []domain.DeploymentStep{{Position: 1, Kind: "ensure_network", Status: domain.StepSucceeded}}, nil
}
