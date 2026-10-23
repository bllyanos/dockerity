// Package containerspec builds the container Dockerity runs for a service in
// a deployment. It is pure; dockerx turns a Spec into Docker API calls.
package containerspec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets/inject"
	"github.com/bllyanos/dockerity/internal/spec"
)

// Label keys specific to deployed containers.
const (
	LabelKind        = "dockerity.kind"
	LabelFingerprint = "dockerity.fingerprint"
)

// Restart policies.
const (
	RestartUnlessStopped = "unless-stopped"
	RestartNo            = "no"
)

// Healthcheck defaults, used when the spec leaves a timing unset.
const (
	DefaultInterval = 10 * time.Second
	DefaultTimeout  = 5 * time.Second
	DefaultRetries  = 3
)

// Mount attaches a named volume.
type Mount struct {
	Volume string
	Target string
}

// Healthcheck is a Docker-native check (the spec's `cmd` form).
type Healthcheck struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	StartPeriod time.Duration
	Retries     int
}

// HTTPCheck is probed by Dockerity over the environment network.
type HTTPCheck struct {
	Port        int
	Path        string
	Interval    time.Duration
	Timeout     time.Duration
	StartPeriod time.Duration
	Retries     int
}

// Spec is everything needed to create one container.
type Spec struct {
	Name          string
	Image         string
	Cmd           []string
	Env           []string // "NAME=value", sorted; includes secret env vars
	Files         []inject.File
	Labels        map[string]string
	Network       string
	Aliases       []string
	Mounts        []Mount
	MemoryBytes   int64
	NanoCPUs      int64
	RestartPolicy string
	Healthcheck   *Healthcheck
	HTTPCheck     *HTTPCheck
	StopTimeout   time.Duration
	// Drain is how long a replaced container keeps running after leaving routing.
	Drain time.Duration
}

// String hides env values and files, which may hold secrets.
func (s Spec) String() string { return "containerspec.Spec{" + s.Name + " " + s.Image + "}" }

// GoString hides secrets from %#v.
func (s Spec) GoString() string { return s.String() }

// Input describes the container to build.
type Input struct {
	App          string
	Env          string
	DeploymentID int64
	Service      spec.Service
	Image        string
	// Secrets holds the decrypted secret set; only referenced keys are used.
	Secrets map[string]string
}

// ShortID is the deployment part of container names.
func ShortID(deploymentID int64) string { return "d" + strconv.FormatInt(deploymentID, 10) }

// Build returns the container spec for one service of a deployment.
func Build(in Input) (Spec, error) {
	svc := in.Service
	plan, err := inject.Build(secretRefs(svc), in.Secrets)
	if err != nil {
		return Spec{}, err
	}
	s := Spec{
		Name:          domain.ContainerName(in.App, in.Env, svc.Name, ShortID(in.DeploymentID)),
		Image:         in.Image,
		Cmd:           slices.Clone(svc.Command),
		Env:           env(svc, plan),
		Files:         plan.Files,
		Network:       domain.NetworkName(in.App, in.Env),
		Aliases:       []string{svc.Name},
		RestartPolicy: RestartUnlessStopped,
		StopTimeout:   svc.EffectiveStopGracePeriod(),
		Drain:         svc.EffectiveDrain(),
	}
	if svc.Kind == spec.KindJob {
		s.RestartPolicy = RestartNo
	}
	for _, v := range svc.Volumes {
		s.Mounts = append(s.Mounts, Mount{Volume: domain.VolumeName(in.App, in.Env, svc.Name, v.Name), Target: v.Path})
	}
	if err := resources(&s, svc.Resources); err != nil {
		return Spec{}, err
	}
	healthchecks(&s, svc)

	s.Labels = map[string]string{
		domain.LabelApp:        in.App,
		domain.LabelEnv:        in.Env,
		domain.LabelService:    svc.Name,
		domain.LabelDeployment: strconv.FormatInt(in.DeploymentID, 10),
		LabelKind:              string(svc.Kind),
	}
	s.Labels[LabelFingerprint] = s.Fingerprint()
	return s, nil
}

func secretRefs(svc spec.Service) []inject.Ref {
	refs := make([]inject.Ref, 0, len(svc.Secrets))
	for _, r := range svc.Secrets {
		refs = append(refs, inject.Ref{Name: r.Name, File: r.File})
	}
	return refs
}

func env(svc spec.Service, plan inject.Plan) []string {
	out := make([]string, 0, len(svc.Env)+len(plan.Env))
	for _, e := range svc.Env {
		out = append(out, e.Name+"="+e.Value)
	}
	out = append(out, plan.Env...)
	slices.Sort(out)
	return out
}

func resources(s *Spec, r *spec.Resources) error {
	if r == nil {
		return nil
	}
	if r.Memory != "" {
		b, err := spec.ParseMemory(r.Memory)
		if err != nil {
			return err
		}
		s.MemoryBytes = b
	}
	if r.CPUs != "" {
		c, err := spec.ParseCPUs(r.CPUs)
		if err != nil {
			return err
		}
		s.NanoCPUs = int64(c * 1e9)
	}
	return nil
}

func healthchecks(s *Spec, svc spec.Service) {
	hc := svc.Healthcheck
	if hc == nil {
		return
	}
	interval, timeout, retries := or(hc.Interval, DefaultInterval), or(hc.Timeout, DefaultTimeout), hc.Retries
	if retries == 0 {
		retries = DefaultRetries
	}
	if len(hc.Cmd) > 0 {
		s.Healthcheck = &Healthcheck{
			Test: append([]string{"CMD"}, hc.Cmd...), Interval: interval, Timeout: timeout,
			StartPeriod: hc.StartPeriod, Retries: retries,
		}
		return
	}
	if hc.HTTP != "" && svc.HTTP != nil {
		s.HTTPCheck = &HTTPCheck{
			Port: svc.HTTP.Port, Path: hc.HTTP, Interval: interval, Timeout: timeout,
			StartPeriod: hc.StartPeriod, Retries: retries,
		}
	}
}

func or(d, def time.Duration) time.Duration {
	if d == 0 {
		return def
	}
	return d
}

// Fingerprint identifies the container configuration, ignoring the name and
// the deployment that created it. Secret values are hashed, never exposed.
func (s Spec) Fingerprint() string {
	labels := maps.Clone(s.Labels)
	delete(labels, domain.LabelDeployment)
	delete(labels, LabelFingerprint)
	data, _ := json.Marshal(struct {
		Image       string
		Cmd         []string
		Env         []string
		Files       []inject.File
		Labels      map[string]string
		Mounts      []Mount
		Memory      int64
		CPUs        int64
		Restart     string
		Healthcheck *Healthcheck
		HTTPCheck   *HTTPCheck
	}{s.Image, s.Cmd, s.Env, s.Files, labels, s.Mounts, s.MemoryBytes, s.NanoCPUs, s.RestartPolicy, s.Healthcheck, s.HTTPCheck})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}
