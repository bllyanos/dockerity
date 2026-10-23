// Package spec parses and validates the app spec: the compose-like YAML that
// Dockerity holds and versions (docs/design.md section 4). It is pure: no
// Docker, SQLite or HTTP dependencies.
package spec

import "time"

// Pos is a 1-based line and column in the spec source.
type Pos struct {
	Line   int
	Column int
}

// Kind is the kind of a service.
type Kind string

// Service kinds.
const (
	KindJob        Kind = "job"
	KindDeployment Kind = "deployment"
	KindStateful   Kind = "stateful"
)

// Phase is when a job runs during a deploy.
type Phase string

// Job phases.
const (
	PhasePreDeploy  Phase = "pre-deploy"
	PhasePostDeploy Phase = "post-deploy"
)

// Strategy is how a deployment service is updated.
type Strategy string

// Update strategies.
const (
	StrategyBlueGreen Strategy = "blue-green"
	StrategyRecreate  Strategy = "recreate"
)

// Spec is a parsed app spec.
type Spec struct {
	Name     string
	NamePos  Pos
	Services []Service // in source order
}

// Service returns the service with the given name.
func (s Spec) Service(name string) (Service, bool) {
	for _, svc := range s.Services {
		if svc.Name == name {
			return svc, true
		}
	}
	return Service{}, false
}

// Service is one entry under `services`.
type Service struct {
	Name string
	// Pos is the position of the service's key.
	Pos Pos
	// FieldPos holds the position of every field that was set, by YAML key.
	FieldPos map[string]Pos

	Kind        Kind
	Phase       Phase
	Strategy    Strategy
	Image       string
	Build       *Build
	Command     []string
	Env         []EnvVar
	Secrets     []SecretRef
	Volumes     []Volume
	DependsOn   []Ref
	HTTP        *HTTP
	Healthcheck *Healthcheck
	Resources   *Resources
	// Drain is how long a replaced container stays up after leaving routing.
	Drain time.Duration
	// StopGracePeriod is how long a container gets between SIGTERM and SIGKILL.
	StopGracePeriod time.Duration
}

// At returns the position of field, or of the service when the field is unset.
func (s Service) At(field string) Pos {
	if p, ok := s.FieldPos[field]; ok {
		return p
	}
	return s.Pos
}

// Build describes how to build a service image from the app repo.
type Build struct {
	Context    string
	Dockerfile string
	Target     string
	Args       []EnvVar
}

// EnvVar is a name/value pair, kept in source order.
type EnvVar struct {
	Name  string
	Value string
	Pos   Pos
}

// SecretRef asks for a secret as an env var, or as a file when File is set.
type SecretRef struct {
	Name string
	File string
	Pos  Pos
}

// Volume is a named persistent volume mounted at Path (stateful services).
type Volume struct {
	Name string
	Path string
	Pos  Pos
}

// Ref names another service.
type Ref struct {
	Name string
	Pos  Pos
}

// HTTP exposes a service through Traefik.
type HTTP struct {
	Port int
	Path string
	Pos  Pos
}

// Healthcheck is either a command or an HTTP path.
type Healthcheck struct {
	Cmd         []string
	HTTP        string
	Interval    time.Duration
	Timeout     time.Duration
	StartPeriod time.Duration
	Retries     int
	Pos         Pos
}

// Resources are container limits, kept as written and checked by Validate.
type Resources struct {
	Memory    string
	CPUs      string
	MemoryPos Pos
	CPUsPos   Pos
}
