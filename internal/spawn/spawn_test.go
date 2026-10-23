package spawn

import (
	"context"
	"errors"
	"maps"
	"regexp"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

type envs struct {
	envs    map[int64]domain.Environment
	deleted []int64
}

func (e *envs) Get(_ context.Context, id int64) (domain.Environment, error) {
	if x, ok := e.envs[id]; ok {
		return x, nil
	}
	return domain.Environment{}, domain.ErrNotFound
}

func (e *envs) CreateEphemeral(_ context.Context, appID int64, name string, ttl time.Duration) (domain.Environment, error) {
	x := domain.Environment{ID: int64(len(e.envs) + 1), AppID: appID, Name: name, Type: domain.EnvEphemeral, TTL: ttl}
	e.envs[x.ID] = x
	return x, nil
}

func (e *envs) Delete(_ context.Context, id int64, skip bool) (domain.Environment, error) {
	if !skip {
		return domain.Environment{}, errors.New("rollback must skip the backup")
	}
	e.deleted = append(e.deleted, id)
	return e.envs[id], nil
}

type memSecrets struct {
	versions map[secrets.Scope][]map[string]string
}

func (m *memSecrets) Update(_ context.Context, s secrets.Scope, c secrets.Changes, _ *int64) (secrets.Version, error) {
	next, err := c.Apply(map[string]string{})
	if err != nil {
		return secrets.Version{}, err
	}
	m.versions[s] = append(m.versions[s], next)
	return secrets.Version{Scope: s, Version: len(m.versions[s])}, nil
}
func (m *memSecrets) Latest(_ context.Context, s secrets.Scope) (secrets.Version, error) {
	if len(m.versions[s]) == 0 {
		return secrets.Version{}, domain.ErrNotFound
	}
	return secrets.Version{Scope: s, Version: len(m.versions[s])}, nil
}
func (m *memSecrets) Versions(context.Context, secrets.Scope) ([]secrets.Version, error) {
	return nil, nil
}
func (m *memSecrets) Values(_ context.Context, s secrets.Scope, v int) (map[string]string, error) {
	return m.versions[s][v-1], nil
}

type deployments struct {
	started []int64
	fail    bool
}

func (d *deployments) Start(_ context.Context, envID, buildID int64, _ *int64) (domain.Deployment, error) {
	if d.fail {
		return domain.Deployment{}, domain.Invalid("buildId", "build failed")
	}
	d.started = append(d.started, envID)
	return domain.Deployment{ID: 9, EnvID: envID, BuildID: buildID}, nil
}

func fixture() (*Service, *envs, *memSecrets, *deployments) {
	e := &envs{envs: map[int64]domain.Environment{1: {ID: 1, AppID: 1, Name: "staging"}, 2: {ID: 2, AppID: 2, Name: "other"}}}
	sec := &memSecrets{versions: map[secrets.Scope][]map[string]string{
		secrets.EnvScope(1): {{"DB_URL": "old"}, {"DB_URL": "mysql://staging", "API_KEY": "k"}},
	}}
	d := &deployments{}
	return &Service{Envs: e, Secrets: sec, Deployments: d}, e, sec, d
}

func TestSpawnCopiesSecretsAndDeploys(t *testing.T) {
	s, _, sec, d := fixture()
	tpl := int64(1)
	env, dep, err := s.Spawn(context.Background(), 1, Input{Name: "pr-7", BuildID: 5, TemplateEnvID: &tpl}, nil)
	if err != nil || env.Name != "pr-7" || dep.BuildID != 5 || len(d.started) != 1 {
		t.Fatalf("spawn %+v %+v %v", env, dep, err)
	}
	got := sec.versions[secrets.EnvScope(env.ID)]
	if len(got) != 1 || !maps.Equal(got[0], map[string]string{"DB_URL": "mysql://staging", "API_KEY": "k"}) {
		t.Fatalf("copied secrets %v", got)
	}
}

func TestSpawnGeneratesNameAndWorksWithoutTemplate(t *testing.T) {
	s, _, sec, _ := fixture()
	env, _, err := s.Spawn(context.Background(), 1, Input{BuildID: 5}, nil)
	if err != nil || !regexp.MustCompile(`^e-[0-9a-f]{6}$`).MatchString(env.Name) {
		t.Fatalf("spawn %+v %v", env, err)
	}
	if len(sec.versions[secrets.EnvScope(env.ID)]) != 0 {
		t.Fatal("no secrets expected without a template")
	}
}

func TestSpawnRollsBack(t *testing.T) {
	s, e, _, d := fixture()
	d.fail = true
	if _, _, err := s.Spawn(context.Background(), 1, Input{Name: "pr-8", BuildID: 5}, nil); err == nil {
		t.Fatal("expected error")
	}
	if len(e.deleted) != 1 {
		t.Fatalf("environment not removed: %v", e.deleted)
	}
}

func TestSpawnRejectsForeignTemplate(t *testing.T) {
	s, e, _, _ := fixture()
	other := int64(2)
	var ve *domain.ValidationError
	if _, _, err := s.Spawn(context.Background(), 1, Input{BuildID: 5, TemplateEnvID: &other}, nil); !errors.As(err, &ve) {
		t.Fatalf("got %v", err)
	}
	if len(e.envs) != 2 {
		t.Fatal("nothing must be created")
	}
}
