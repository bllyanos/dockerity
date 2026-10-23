package inject

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

var values = map[string]string{"DB_PASSWORD": "hunter2", "API_KEY": "k-123", "TLS_KEY": "-----PEM-----"}

func TestBuildEnvAndFiles(t *testing.T) {
	plan, err := Build([]Ref{
		{Name: "DB_PASSWORD"},
		{Name: "API_KEY"},
		{Name: "API_KEY"},
		{Name: "TLS_KEY", File: "/run/secrets/tls.key"},
	}, values)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(plan.Env, []string{"API_KEY=k-123", "DB_PASSWORD=hunter2"}) {
		t.Fatalf("env %v", plan.Env)
	}
	if len(plan.Files) != 1 || plan.Files[0].Path != "/run/secrets/tls.key" ||
		string(plan.Files[0].Content) != "-----PEM-----" || plan.Files[0].Mode != 0o400 {
		t.Fatalf("files %+v", plan.Files)
	}
}

func TestBuildMissing(t *testing.T) {
	_, err := Build([]Ref{{Name: "ZETA"}, {Name: "DB_PASSWORD"}, {Name: "ALPHA", File: "/run/a"}, {Name: "ZETA"}}, values)
	var me *MissingError
	if !errors.As(err, &me) || !slices.Equal(me.Names, []string{"ALPHA", "ZETA"}) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildFileErrors(t *testing.T) {
	cases := map[string][]Ref{
		"relative":  {{Name: "API_KEY", File: "secrets/key"}},
		"unclean":   {{Name: "API_KEY", File: "/run/../etc/key"}},
		"root":      {{Name: "API_KEY", File: "/"}},
		"collision": {{Name: "API_KEY", File: "/run/k"}, {Name: "DB_PASSWORD", File: "/run/k"}},
	}
	for name, refs := range cases {
		_, err := Build(refs, values)
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		for _, v := range values {
			if strings.Contains(err.Error(), v) {
				t.Errorf("%s: error leaks a value: %v", name, err)
			}
		}
	}
}

func TestPlanNeverPrintsValues(t *testing.T) {
	plan, _ := Build([]Ref{{Name: "DB_PASSWORD"}, {Name: "TLS_KEY", File: "/run/k"}}, values)
	for _, s := range []string{fmt.Sprint(plan), fmt.Sprintf("%v %+v %#v", plan, plan, plan)} {
		if strings.Contains(s, "hunter2") || strings.Contains(s, "PEM") {
			t.Fatalf("plan formatting leaks values: %s", s)
		}
	}
}
