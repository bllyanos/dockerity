// Package drift compares what runs in Docker with an environment's current
// deployment (docs/design.md section 6.5). It is pure: the reconciler gathers
// the input and applies the repairs it returns.
package drift

import (
	"sort"
	"strconv"
)

// Kind is a kind of drift.
type Kind string

// Drift kinds.
const (
	// KindMissing: a long-running service of the current deployment has no container.
	KindMissing Kind = "missing"
	// KindStopped: the service's expected container exists but does not run.
	KindStopped Kind = "stopped"
	// KindExtra: a container runs that the current deployment does not expect.
	KindExtra Kind = "extra"
	// KindDanglingRoute: a route points at a container that does not run.
	KindDanglingRoute Kind = "dangling_route"
)

// ServiceKind mirrors the spec's long-running kinds.
type ServiceKind string

// Long-running service kinds.
const (
	Deployment ServiceKind = "deployment"
	Stateful   ServiceKind = "stateful"
)

// Container is one managed container of the environment.
type Container struct {
	ID         string
	Name       string
	Service    string
	Deployment string // the dockerity.deployment label
	Running    bool
}

// Input describes an environment.
type Input struct {
	// Current is the current deployment's ID, 0 when there is none.
	Current int64
	// Services are the long-running services of the current deployment's spec.
	Services map[string]ServiceKind
	// Containers are the environment's managed containers.
	Containers []Container
	// Routed are the container names the route file sends traffic to.
	Routed map[string]bool
}

// Item is one drift finding.
type Item struct {
	Kind      Kind
	Service   string
	Container string // empty for KindMissing
	Detail    string
}

// Report is the drift of an environment and the repairs that are safe.
type Report struct {
	Items []Item
	// Remove are stopped containers nothing expects, by ID.
	Remove []string
	// RewriteRoutes is true when the route file names containers that do not run.
	RewriteRoutes bool
}

// Clean reports whether nothing drifted.
func (r Report) Clean() bool { return len(r.Items) == 0 && len(r.Remove) == 0 }

// Compute finds the drift of an environment. A deployment service expects a
// container of the current deployment; a stateful service is kept across
// deployments when unchanged, so it expects its newest container from the
// current deployment or an earlier one.
func Compute(in Input) Report {
	expected := expectedContainers(in)
	var r Report
	for _, name := range sortedKeys(in.Services) {
		c, ok := expected[name]
		switch {
		case !ok:
			r.Items = append(r.Items, Item{Kind: KindMissing, Service: name, Detail: "no container; redeploy to recreate it"})
		case !c.Running:
			r.Items = append(r.Items, Item{Kind: KindStopped, Service: name, Container: c.Name, Detail: "expected container is not running"})
		}
	}
	running := map[string]bool{}
	for _, c := range sortedContainers(in.Containers) {
		if c.Running {
			running[c.Name] = true
		}
		if e, ok := expected[c.Service]; ok && e.Name == c.Name {
			continue
		}
		if c.Running {
			r.Items = append(r.Items, Item{Kind: KindExtra, Service: c.Service, Container: c.Name, Detail: "runs but the current deployment does not expect it"})
		} else {
			r.Remove = append(r.Remove, c.ID)
		}
	}
	for _, name := range sortedKeys(in.Routed) {
		if in.Routed[name] && !running[name] {
			r.Items = append(r.Items, Item{Kind: KindDanglingRoute, Container: name, Detail: "routed but not running"})
			r.RewriteRoutes = true
		}
	}
	return r
}

// expectedContainers picks, per long-running service, the container the
// current deployment expects.
func expectedContainers(in Input) map[string]Container {
	out := map[string]Container{}
	if in.Current == 0 {
		return out
	}
	for _, c := range in.Containers {
		kind, ok := in.Services[c.Service]
		if !ok {
			continue
		}
		id, err := strconv.ParseInt(c.Deployment, 10, 64)
		if err != nil || id > in.Current || (kind == Deployment && id != in.Current) {
			continue
		}
		if prev, ok := out[c.Service]; ok && !newer(c, id, prev) {
			continue
		}
		out[c.Service] = c
	}
	return out
}

// newer prefers a later deployment, then a running container.
func newer(c Container, id int64, prev Container) bool {
	prevID, _ := strconv.ParseInt(prev.Deployment, 10, 64)
	if id != prevID {
		return id > prevID
	}
	return c.Running && !prev.Running
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedContainers(cs []Container) []Container {
	out := append([]Container(nil), cs...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
