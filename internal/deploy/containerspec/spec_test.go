package containerspec

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/routing"
	"github.com/bllyanos/dockerity/internal/secrets/inject"
	"github.com/bllyanos/dockerity/internal/spec"
)

func service(t *testing.T, yaml, name string) spec.Service {
	t.Helper()
	s, err := spec.Load([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	svc, _ := s.Service(name)
	return svc
}

const apiSpec = `
services:
  api:
    kind: deployment
    image: shop-api
    command: ["./app", "serve"]
    env:
      PORT: "8080"
    secrets:
      - DATABASE_URL
      - name: TLS_KEY
        file: /run/secrets/tls.key
    http:
      port: 8080
      path: /api
    healthcheck:
      http: /healthz
      interval: 2s
    resources:
      memory: 512m
      cpus: "0.5"
  db:
    kind: stateful
    image: mysql:8.4
    volumes:
      data: /var/lib/mysql
    healthcheck:
      cmd: ["mysqladmin", "ping"]
  migrate:
    kind: job
    image: shop-api
`

var secretValues = map[string]string{"DATABASE_URL": "mysql://u:pw@db/shop", "TLS_KEY": "PEM", "UNUSED": "x"}

func input(svc spec.Service) Input {
	return Input{App: "shop", Env: "staging", Domain: "staging.shop.test", DeploymentID: 42, Service: svc,
		Image: "dkt/shop/api:abc", Secrets: secretValues, Routing: routing.DefaultOptions}
}

func TestBuildDeployment(t *testing.T) {
	s, err := Build(input(service(t, apiSpec, "api")))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "dkt-shop-staging-api-d42" || s.Network != "dkt-shop-staging" || !slices.Equal(s.Aliases, []string{"api"}) {
		t.Fatalf("names %+v", s)
	}
	if !slices.Equal(s.Env, []string{"DATABASE_URL=mysql://u:pw@db/shop", "PORT=8080"}) {
		t.Fatalf("env %v", s.Env)
	}
	if len(s.Files) != 1 || s.Files[0].Path != "/run/secrets/tls.key" {
		t.Fatalf("files %+v", s.Files)
	}
	if s.MemoryBytes != 512<<20 || s.NanoCPUs != 500_000_000 || s.RestartPolicy != RestartUnlessStopped {
		t.Fatalf("resources %+v", s)
	}
	if s.HTTPCheck == nil || s.HTTPCheck.Port != 8080 || s.HTTPCheck.Path != "/healthz" || s.HTTPCheck.Interval != 2*time.Second || s.HTTPCheck.Retries != DefaultRetries {
		t.Fatalf("http check %+v", s.HTTPCheck)
	}
	for k, want := range map[string]string{
		"dockerity.app": "shop", "dockerity.env": "staging", "dockerity.service": "api", "dockerity.deployment": "42",
		LabelKind: "deployment", "traefik.http.routers.dkt-shop-staging-api.rule": "Host(`staging.shop.test`) && PathPrefix(`/api`)",
	} {
		if s.Labels[k] != want {
			t.Errorf("label %s = %q, want %q", k, s.Labels[k], want)
		}
	}
	for k, v := range s.Labels {
		for _, secret := range secretValues {
			if strings.Contains(v, secret) && len(secret) > 3 {
				t.Fatalf("secret value in label %s", k)
			}
		}
	}
	if str := fmt.Sprintf("%v %+v %#v", s, s, s); strings.Contains(str, "pw@db") {
		t.Fatal("formatting leaks secrets")
	}
}

func TestBuildStatefulAndJob(t *testing.T) {
	db, err := Build(input(service(t, apiSpec, "db")))
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Mounts) != 1 || db.Mounts[0] != (Mount{Volume: "dkt-shop-staging-db-data", Target: "/var/lib/mysql"}) {
		t.Fatalf("mounts %+v", db.Mounts)
	}
	if db.Healthcheck == nil || !slices.Equal(db.Healthcheck.Test, []string{"CMD", "mysqladmin", "ping"}) || db.Healthcheck.Interval != DefaultInterval {
		t.Fatalf("healthcheck %+v", db.Healthcheck)
	}
	if _, routed := db.Labels["traefik.enable"]; routed {
		t.Fatal("service without http must not be routed")
	}

	job, err := Build(input(service(t, apiSpec, "migrate")))
	if err != nil {
		t.Fatal(err)
	}
	if job.RestartPolicy != RestartNo || job.Labels["traefik.enable"] != "" {
		t.Fatalf("job %+v", job)
	}
}

func TestMissingSecretFails(t *testing.T) {
	in := input(service(t, apiSpec, "api"))
	in.Secrets = map[string]string{}
	var me *inject.MissingError
	if _, err := Build(in); !errors.As(err, &me) {
		t.Fatalf("got %v", err)
	}
}

func TestFingerprint(t *testing.T) {
	svc := service(t, apiSpec, "db")
	a, _ := Build(input(svc))
	other := input(svc)
	other.DeploymentID = 43
	b, _ := Build(other)
	if a.Fingerprint() != b.Fingerprint() || a.Labels[LabelFingerprint] != a.Fingerprint() {
		t.Fatal("fingerprint must not depend on the deployment")
	}
	other.Image = "mysql:9"
	c, _ := Build(other)
	if c.Fingerprint() == a.Fingerprint() {
		t.Fatal("fingerprint must change with the image")
	}
	if strings.Contains(a.Fingerprint(), "mysql") {
		t.Fatal("fingerprint must be a hash")
	}
}
