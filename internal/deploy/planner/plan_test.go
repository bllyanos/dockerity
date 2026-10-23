package planner

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

func designExample(t *testing.T) spec.Spec {
	t.Helper()
	src, err := os.ReadFile("../../spec/testdata/example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s, err := spec.Load(src)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func baseInput(t *testing.T) Input {
	return Input{
		App: "shop", Env: "staging", Spec: designExample(t),
		BuildImages:  map[string]string{"migrate": "dkt/shop/migrate:abc", "api": "dkt/shop/api:abc", "scheduler": "dkt/shop/scheduler:abc"},
		SecretNames:  []string{"MYSQL_ROOT_PASSWORD", "DATABASE_URL"},
		Fingerprints: map[string]string{"mysql": "f1"},
	}
}

func summary(p Plan) []string {
	var out []string
	for _, s := range p.Steps {
		out = append(out, strings.TrimSuffix(string(s.Kind)+":"+s.Service, ":"))
	}
	return out
}

func TestFirstDeployOfDesignExample(t *testing.T) {
	p, err := Make(baseInput(t))
	if err != nil {
		t.Fatal(err)
	}
	want := "ensure_network ensure_volumes stateful:mysql job:migrate deploy:api deploy:scheduler finish"
	if got := strings.Join(summary(p), " "); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if p.Downtime() {
		t.Fatal("a first deploy has nothing to take down")
	}
	if v := p.Steps[1].Volumes; len(v) != 1 || v[0] != "dkt-shop-staging-mysql-data" {
		t.Fatalf("volumes %v", v)
	}
	api, sched := p.Steps[4], p.Steps[5]
	if api.Strategy != spec.StrategyBlueGreen || api.Image != "dkt/shop/api:abc" || sched.Strategy != spec.StrategyRecreate {
		t.Fatalf("deploy steps %+v %+v", api, sched)
	}
	if p.Steps[2].Image != "mysql:8.4" || p.Steps[3].Phase != spec.PhasePreDeploy {
		t.Fatalf("stateful/job %+v %+v", p.Steps[2], p.Steps[3])
	}
}

func TestRedeploySkipsUnchangedStateful(t *testing.T) {
	in := baseInput(t)
	in.Current = map[string]Running{
		"mysql":     {Image: "mysql:8.4", Fingerprint: "f1"},
		"api":       {Image: "dkt/shop/api:old"},
		"scheduler": {Image: "dkt/shop/scheduler:old"},
	}
	p, err := Make(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(summary(p), " "); got != "ensure_network ensure_volumes job:migrate deploy:api deploy:scheduler finish" {
		t.Fatalf("got %s", got)
	}
	api, sched := p.Steps[3], p.Steps[4]
	if api.Downtime || !api.ImageChanged || !sched.Downtime {
		t.Fatalf("blue-green has no downtime, recreate does: %+v %+v", api, sched)
	}
}

func TestStatefulImageChangeHasDowntime(t *testing.T) {
	in := baseInput(t)
	in.Current = map[string]Running{"mysql": {Image: "mysql:8.0", Fingerprint: "f1"}}
	p, err := Make(in)
	if err != nil {
		t.Fatal(err)
	}
	st := p.Steps[2]
	if st.Kind != StepStateful || !st.Downtime || !st.ImageChanged || !p.Downtime() {
		t.Fatalf("stateful step %+v", st)
	}

	in.Current = map[string]Running{"mysql": {Image: "mysql:8.4", Fingerprint: "old-config"}}
	p, _ = Make(in)
	if st := p.Steps[2]; st.Kind != StepStateful || st.ImageChanged || !st.Downtime {
		t.Fatalf("config change recreates without an image change: %+v", st)
	}
}

func TestPostDeployJobsRunLast(t *testing.T) {
	s, err := spec.Load([]byte(`
services:
  notify:
    kind: job
    phase: post-deploy
    image: curl
  web:
    kind: deployment
    strategy: recreate
    image: nginx
`))
	if err != nil {
		t.Fatal(err)
	}
	p, err := Make(Input{App: "a", Env: "e", Spec: s})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(summary(p), " "); got != "ensure_network deploy:web job:notify finish" {
		t.Fatalf("got %s", got)
	}
}

func TestValidation(t *testing.T) {
	in := baseInput(t)
	delete(in.BuildImages, "api")
	delete(in.BuildImages, "scheduler")
	_, err := Make(in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) || !strings.Contains(err.Error(), "api, scheduler") {
		t.Fatalf("missing images: %v", err)
	}

	in = baseInput(t)
	in.SecretNames = nil
	_, err = Make(in)
	if !errors.As(err, &ve) || !strings.Contains(err.Error(), "MYSQL_ROOT_PASSWORD (used by mysql)") ||
		!strings.Contains(err.Error(), "DATABASE_URL (used by migrate)") || !strings.Contains(err.Error(), "DATABASE_URL (used by api)") {
		t.Fatalf("missing secrets: %v", err)
	}
}
