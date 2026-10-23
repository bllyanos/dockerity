package drift

import (
	"reflect"
	"testing"
)

func kinds(r Report) map[Kind][]string {
	out := map[Kind][]string{}
	for _, it := range r.Items {
		key := it.Container
		if key == "" {
			key = it.Service
		}
		out[it.Kind] = append(out[it.Kind], key)
	}
	return out
}

func TestComputeClean(t *testing.T) {
	r := Compute(Input{
		Current:  5,
		Services: map[string]ServiceKind{"web": Deployment, "db": Stateful},
		Containers: []Container{
			{ID: "1", Name: "web-5", Service: "web", Deployment: "5", Running: true},
			{ID: "2", Name: "db-3", Service: "db", Deployment: "3", Running: true}, // kept from an earlier deployment
		},
		Routed: map[string]bool{"web-5": true},
	})
	if !r.Clean() || r.RewriteRoutes {
		t.Fatalf("expected no drift, got %+v", r)
	}
}

func TestComputeReportsEachKind(t *testing.T) {
	r := Compute(Input{
		Current:  5,
		Services: map[string]ServiceKind{"web": Deployment, "worker": Deployment, "db": Stateful},
		Containers: []Container{
			{ID: "1", Name: "web-4", Service: "web", Deployment: "4", Running: true},    // old version still running
			{ID: "2", Name: "db-5", Service: "db", Deployment: "5", Running: false},     // expected but stopped
			{ID: "3", Name: "db-3", Service: "db", Deployment: "3", Running: false},     // stopped leftover
			{ID: "4", Name: "gone-2", Service: "gone", Deployment: "2", Running: false}, // service left the spec
			{ID: "5", Name: "web-6", Service: "web", Deployment: "6", Running: false},   // failed later deployment
		},
		Routed: map[string]bool{"web-4": true, "web-5": true},
	})
	want := map[Kind][]string{
		KindMissing:       {"web", "worker"},
		KindStopped:       {"db-5"},
		KindExtra:         {"web-4"},
		KindDanglingRoute: {"web-5"},
	}
	if got := kinds(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("drift\n got %v\nwant %v", got, want)
	}
	if !reflect.DeepEqual(r.Remove, []string{"3", "4", "5"}) || !r.RewriteRoutes {
		t.Fatalf("repairs %+v", r)
	}
}

func TestComputeNeverRemovesRunningOrExpectedContainers(t *testing.T) {
	r := Compute(Input{
		Current:  2,
		Services: map[string]ServiceKind{"db": Stateful},
		Containers: []Container{
			{ID: "a", Name: "db-2", Service: "db", Deployment: "2", Running: false},
			{ID: "b", Name: "extra", Service: "x", Deployment: "1", Running: true},
		},
	})
	if len(r.Remove) != 0 {
		t.Fatalf("removed %v", r.Remove)
	}
}

func TestComputeWithoutCurrentDeployment(t *testing.T) {
	r := Compute(Input{Containers: []Container{
		{ID: "a", Name: "web-1", Service: "web", Deployment: "1", Running: true},
		{ID: "b", Name: "web-2", Service: "web", Deployment: "2"},
	}})
	if got := kinds(r); !reflect.DeepEqual(got, map[Kind][]string{KindExtra: {"web-1"}}) || !reflect.DeepEqual(r.Remove, []string{"b"}) {
		t.Fatalf("got %v remove %v", got, r.Remove)
	}
}
