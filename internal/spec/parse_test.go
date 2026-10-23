package spec

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestParseDesignExample(t *testing.T) {
	src, err := os.ReadFile("testdata/example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.Name != "shop" || len(s.Services) != 4 {
		t.Fatalf("unexpected spec %+v", s)
	}
	names := []string{}
	for _, svc := range s.Services {
		names = append(names, svc.Name)
	}
	if !slices.Equal(names, []string{"mysql", "migrate", "api", "scheduler"}) {
		t.Fatalf("services out of order: %v", names)
	}

	mysql, _ := s.Service("mysql")
	if mysql.Kind != KindStateful || mysql.Image != "mysql:8.4" || mysql.Pos.Line != 3 ||
		len(mysql.Volumes) != 1 || mysql.Volumes[0] != (Volume{Name: "data", Path: "/var/lib/mysql", Pos: Pos{7, 7}}) ||
		len(mysql.Secrets) != 1 || mysql.Secrets[0].Name != "MYSQL_ROOT_PASSWORD" ||
		!slices.Equal(mysql.Healthcheck.Cmd, []string{"mysqladmin", "ping", "-h", "127.0.0.1"}) {
		t.Fatalf("mysql: %+v", mysql)
	}

	migrate, _ := s.Service("migrate")
	if migrate.Kind != KindJob || migrate.Phase != PhasePreDeploy || migrate.Build == nil ||
		migrate.Build.Context != "." || migrate.Build.Dockerfile != "Dockerfile" ||
		len(migrate.DependsOn) != 1 || migrate.DependsOn[0].Name != "mysql" {
		t.Fatalf("migrate: %+v", migrate)
	}

	api, _ := s.Service("api")
	if api.Strategy != StrategyBlueGreen || api.HTTP == nil || api.HTTP.Port != 8080 || api.HTTP.Path != "/api" ||
		api.Healthcheck.HTTP != "/healthz" || api.Resources.Memory != "512m" || api.Resources.CPUs != "0.5" {
		t.Fatalf("api: %+v", api)
	}
	if p := api.At("strategy"); p != (Pos{26, 5}) {
		t.Fatalf("strategy position %+v", p)
	}

	scheduler, _ := s.Service("scheduler")
	if scheduler.Build == nil || scheduler.Build.Context != "." || scheduler.Strategy != StrategyRecreate {
		t.Fatalf("scheduler: %+v", scheduler)
	}
}

func TestParseFields(t *testing.T) {
	src := `
name: app
services:
  web:
    kind: deployment
    build: ./web
    env:
      PORT: 8080
      DEBUG: true
      EMPTY:
    secrets:
      - API_KEY
      - name: TLS_KEY
        file: /run/secrets/tls.key
    healthcheck:
      http: /healthz
      interval: 10s
      timeout: 2s
      start_period: 30s
      retries: 3
`
	s, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	web := s.Services[0]
	if web.Build.Context != "./web" {
		t.Fatalf("build shorthand: %+v", web.Build)
	}
	wantEnv := []EnvVar{{"PORT", "8080", Pos{8, 7}}, {"DEBUG", "true", Pos{9, 7}}, {"EMPTY", "", Pos{10, 7}}}
	if !slices.Equal(web.Env, wantEnv) {
		t.Fatalf("env %+v", web.Env)
	}
	if !slices.Equal(web.Secrets, []SecretRef{{"API_KEY", "", Pos{12, 9}}, {"TLS_KEY", "/run/secrets/tls.key", Pos{13, 9}}}) {
		t.Fatalf("secrets %+v", web.Secrets)
	}
	hc := web.Healthcheck
	if hc.Interval != 10*time.Second || hc.Timeout != 2*time.Second || hc.StartPeriod != 30*time.Second || hc.Retries != 3 {
		t.Fatalf("healthcheck %+v", hc)
	}
}

// problemLines parses src and returns "line: message fragment" checks.
func problems(t *testing.T, src string) Problems {
	t.Helper()
	_, err := Parse([]byte(src))
	var ps Problems
	if !errors.As(err, &ps) {
		t.Fatalf("expected Problems, got %v", err)
	}
	return ps
}

func TestParseProblems(t *testing.T) {
	cases := []struct {
		name string
		src  string
		line int
		want string
	}{
		{"syntax", "name: x\nservices:\n  a: [\n", 3, "invalid YAML"},
		{"empty", "", 1, "spec is empty"},
		{"not a mapping", "- a\n- b\n", 1, "the spec must be a mapping"},
		{"no services", "name: x\n", 1, "must define services"},
		{"unknown top field", "name: x\nversion: 3\nservices: {}\n", 2, `unknown field "version"`},
		{"container_name", "services:\n  a:\n    image: x\n    container_name: foo\n", 4, "Dockerity owns all container names"},
		{"ports", "services:\n  a:\n    image: x\n    ports: [\"80:80\"]\n", 4, "host port bindings are not allowed"},
		{"unknown service field", "services:\n  a:\n    imgae: x\n", 3, `unknown field "imgae" in service a`},
		{"wrong type", "services:\n  a:\n    command: ./run\n", 3, "command must be a list of strings"},
		{"bad port", "services:\n  a:\n    http:\n      port: eighty\n", 4, "http.port must be an integer"},
		{"bad duration", "services:\n  a:\n    healthcheck:\n      interval: soon\n", 4, "positive duration"},
		{"duplicate key", "services:\n  a:\n    image: x\n    image: y\n", 4, `duplicate key "image"`},
		{"service not mapping", "services:\n  a: nginx\n", 2, "service a must be a mapping"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ps := problems(t, c.src)
			for _, p := range ps {
				if p.Pos.Line == c.line && strings.Contains(p.Message, c.want) {
					return
				}
			}
			t.Fatalf("want %q at line %d, got:\n%v", c.want, c.line, ps)
		})
	}
}

func TestParseReportsAllProblemsInOrder(t *testing.T) {
	ps := problems(t, "services:\n  a:\n    imgae: x\n  b:\n    command: run\n    container_name: b\n")
	if len(ps) != 3 {
		t.Fatalf("want 3 problems, got %v", ps)
	}
	for i := 1; i < len(ps); i++ {
		if ps[i].Pos.Line < ps[i-1].Pos.Line {
			t.Fatalf("problems not sorted: %v", ps)
		}
	}
	if !strings.HasPrefix(ps.Error(), "line 3, column 5: ") {
		t.Fatalf("error format: %q", ps.Error())
	}
}
