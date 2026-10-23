package spec

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestDesignExampleIsValid(t *testing.T) {
	src, _ := os.ReadFile("testdata/example.yaml")
	if _, err := Load(src); err != nil {
		t.Fatalf("design example should be valid:\n%v", err)
	}
}

// validProblems loads src and returns its validation problems.
func validProblems(t *testing.T, src string) Problems {
	t.Helper()
	_, err := Load([]byte(src))
	var ps Problems
	if !errors.As(err, &ps) {
		t.Fatalf("expected Problems, got %v", err)
	}
	return ps
}

func TestValidationRules(t *testing.T) {
	cases := []struct {
		name string
		src  string
		line int
		want string
	}{
		{"bad app name", "name: Shop_App\nservices:\n  a:\n    kind: job\n    image: x\n", 1, `name "Shop_App" must be`},
		{"bad service name", "services:\n  Web:\n    kind: job\n    image: x\n", 2, `service name "Web"`},
		{"missing kind", "services:\n  a:\n    image: x\n", 2, "needs a kind"},
		{"bad kind", "services:\n  a:\n    kind: daemon\n    image: x\n", 3, `kind "daemon"`},
		{"no source", "services:\n  a:\n    kind: job\n", 2, "needs either image or build"},
		{"both sources", "services:\n  a:\n    kind: job\n    image: x\n    build: .\n", 5, "cannot have both image and build"},
		{"context escapes", "services:\n  a:\n    kind: job\n    build: ../other\n", 4, "must be a path inside the repo"},
		{"phase on deployment", "services:\n  a:\n    kind: deployment\n    strategy: recreate\n    image: x\n    phase: pre-deploy\n", 6, "phase is only allowed on jobs"},
		{"bad phase", "services:\n  a:\n    kind: job\n    image: x\n    phase: later\n", 5, `phase "later"`},
		{"stateful blue-green", "services:\n  db:\n    kind: stateful\n    image: x\n    strategy: blue-green\n", 5, "stateful services always use recreate"},
		{"strategy on job", "services:\n  a:\n    kind: job\n    image: x\n    strategy: recreate\n", 5, "only allowed on deployments"},
		{"bad strategy", "services:\n  a:\n    kind: deployment\n    image: x\n    strategy: rolling\n", 5, `strategy "rolling"`},
		{"blue-green default needs healthcheck", "services:\n  api:\n    kind: deployment\n    image: x\n", 2, "needs a healthcheck"},
		{"explicit blue-green needs healthcheck", "services:\n  api:\n    kind: deployment\n    image: x\n    strategy: blue-green\n", 5, "needs a healthcheck"},
		{"volumes on deployment", "services:\n  a:\n    kind: deployment\n    strategy: recreate\n    image: x\n    volumes:\n      data: /data\n", 6, "only allowed on stateful"},
		{"relative volume path", "services:\n  db:\n    kind: stateful\n    image: x\n    volumes:\n      data: data\n", 6, "must be an absolute, clean path"},
		{"unknown dependency", "services:\n  a:\n    kind: job\n    image: x\n    depends_on: [db]\n", 5, `unknown service "db"`},
		{"self dependency", "services:\n  a:\n    kind: job\n    image: x\n    depends_on: [a]\n", 5, "cannot depend on itself"},
		{"cycle", "services:\n  a:\n    kind: job\n    image: x\n    depends_on: [b]\n  b:\n    kind: job\n    image: x\n    depends_on: [a]\n", 5, "cycle: a -> b -> a"},
		{"http on job", "services:\n  a:\n    kind: job\n    image: x\n    http:\n      port: 80\n", 5, "jobs cannot receive HTTP"},
		{"bad port", "services:\n  a:\n    kind: deployment\n    strategy: recreate\n    image: x\n    http:\n      port: 70000\n", 6, "between 1 and 65535"},
		{"http path", "services:\n  a:\n    kind: deployment\n    strategy: recreate\n    image: x\n    http:\n      port: 80\n      path: api\n", 6, "must start with /"},
		{"healthcheck empty", "services:\n  a:\n    kind: deployment\n    image: x\n    healthcheck:\n      retries: 3\n", 5, "needs cmd or http"},
		{"healthcheck both", "services:\n  a:\n    kind: deployment\n    image: x\n    http: {port: 80}\n    healthcheck:\n      cmd: [true]\n      http: /h\n", 6, "both cmd and http"},
		{"http healthcheck needs port", "services:\n  a:\n    kind: deployment\n    image: x\n    healthcheck:\n      http: /healthz\n", 5, "needs http.port"},
		{"bad secret name", "services:\n  a:\n    kind: job\n    image: x\n    secrets: [db_password]\n", 5, `secret name "db_password"`},
		{"secret also in env", "services:\n  a:\n    kind: job\n    image: x\n    env:\n      TOKEN: x\n    secrets: [TOKEN]\n", 7, "both a plain env var and a secret"},
		{"relative secret file", "services:\n  a:\n    kind: job\n    image: x\n    secrets:\n      - name: TLS\n        file: tls.key\n", 6, "absolute, clean path"},
		{"bad env name", "services:\n  a:\n    kind: job\n    image: x\n    env:\n      1BAD: x\n", 6, `env name "1BAD"`},
		{"bad memory", "services:\n  a:\n    kind: job\n    image: x\n    resources:\n      memory: lots\n", 6, `memory "lots"`},
		{"bad cpus", "services:\n  a:\n    kind: job\n    image: x\n    resources:\n      cpus: 0\n", 6, `cpus "0"`},
		{"no services", "name: x\nservices: {}\n", 1, "at least one service"},
		{"drain too long", "services:\n  a:\n    kind: deployment\n    strategy: recreate\n    image: x\n    drain: 11m\n", 6, "drain must be at most"},
		{"grace too long", "services:\n  a:\n    kind: job\n    image: x\n    stop_grace_period: 1h\n", 5, "stop_grace_period must be at most"},
		{"drain on job", "services:\n  a:\n    kind: job\n    image: x\n    drain: 5s\n", 5, "no drain period"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ps := validProblems(t, c.src)
			for _, p := range ps {
				if p.Pos.Line == c.line && strings.Contains(p.Message, c.want) {
					return
				}
			}
			t.Fatalf("want %q at line %d, got:\n%v", c.want, c.line, ps)
		})
	}
}

func TestValidateCollectsEveryProblem(t *testing.T) {
	ps := validProblems(t, "services:\n  a:\n    kind: daemon\n  b:\n    kind: job\n    image: x\n    depends_on: [nope]\n")
	if len(ps) != 3 { // bad kind, no source for a, unknown dependency
		t.Fatalf("want 3 problems, got:\n%v", ps)
	}
}
