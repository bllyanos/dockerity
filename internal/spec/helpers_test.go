package spec

import (
	"slices"
	"testing"
	"time"
)

func TestParseMemory(t *testing.T) {
	cases := map[string]int64{"512m": 512 << 20, "1g": 1 << 30, "1.5G": 3 << 29, "10240k": 10 << 20, "64mb": 64 << 20, "7340032": 7 << 20}
	for in, want := range cases {
		if got, err := ParseMemory(in); err != nil || got != want {
			t.Errorf("%s: got %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "lots", "512x", "-1m", "1m", "1t"} {
		if _, err := ParseMemory(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestParseCPUs(t *testing.T) {
	for _, c := range []struct {
		in   string
		want float64
	}{{"0.5", 0.5}, {"2", 2}, {" 1.25 ", 1.25}} {
		if got, err := ParseCPUs(c.in); err != nil || got != c.want {
			t.Errorf("%q: got %v, %v", c.in, got, err)
		}
	}
	for _, bad := range []string{"", "0", "-1", "half", "2000"} {
		if _, err := ParseCPUs(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func svc(name string, deps ...string) Service {
	s := Service{Name: name}
	for _, d := range deps {
		s.DependsOn = append(s.DependsOn, Ref{Name: d})
	}
	return s
}

func TestDependencyOrder(t *testing.T) {
	order, cycle := DependencyOrder(Spec{Services: []Service{svc("api", "db", "cache"), svc("worker", "db"), svc("db"), svc("cache"), svc("x", "missing")}})
	if cycle != nil || !slices.Equal(order, []string{"db", "cache", "api", "worker", "x"}) {
		t.Fatalf("order %v cycle %v", order, cycle)
	}

	_, cycle = DependencyOrder(Spec{Services: []Service{svc("a", "b"), svc("b", "c"), svc("c", "b")}})
	if !slices.Equal(cycle, []string{"b", "c", "b"}) {
		t.Fatalf("cycle %v", cycle)
	}
	_, cycle = DependencyOrder(Spec{Services: []Service{svc("self", "self")}})
	if !slices.Equal(cycle, []string{"self", "self"}) {
		t.Fatalf("self cycle %v", cycle)
	}
}

func TestEffectiveDefaults(t *testing.T) {
	if (Service{Kind: KindDeployment}).EffectiveStrategy() != StrategyBlueGreen ||
		(Service{Kind: KindDeployment, Strategy: StrategyRecreate}).EffectiveStrategy() != StrategyRecreate ||
		(Service{Kind: KindStateful}).EffectiveStrategy() != StrategyRecreate ||
		(Service{Kind: KindJob}).EffectiveStrategy() != "" {
		t.Fatal("strategy defaults")
	}
	if (Service{Kind: KindJob}).EffectivePhase() != PhasePreDeploy || (HTTP{}).RoutePath() != "/" {
		t.Fatal("phase or path defaults")
	}
}

func TestEffectiveTimings(t *testing.T) {
	s := Service{}
	if s.EffectiveDrain() != DefaultDrain || s.EffectiveStopGracePeriod() != DefaultStopGracePeriod {
		t.Fatal("defaults")
	}
	s = Service{Drain: 2 * time.Second, StopGracePeriod: 3 * time.Second}
	if s.EffectiveDrain() != 2*time.Second || s.EffectiveStopGracePeriod() != 3*time.Second {
		t.Fatal("overrides")
	}
}

func TestDependents(t *testing.T) {
	s := Spec{Services: []Service{svc("db"), svc("api", "db"), svc("worker", "api"), svc("cache"), svc("web", "cache")}}
	if got := Dependents(s, "db"); !slices.Equal(got, []string{"api", "worker"}) {
		t.Fatalf("db dependents %v", got)
	}
	if got := Dependents(s, "worker"); len(got) != 0 {
		t.Fatalf("leaf dependents %v", got)
	}
}
