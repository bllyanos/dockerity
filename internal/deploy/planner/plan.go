// Package planner turns an app spec, a build and an environment's current
// state into an ordered list of deploy steps (docs/design.md section 6.2).
// It is pure: no Docker or SQLite.
package planner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

// StepKind is what a step does.
type StepKind string

// Step kinds, in the order they appear in a plan.
const (
	StepEnsureNetwork StepKind = "ensure_network"
	StepEnsureVolumes StepKind = "ensure_volumes"
	StepStateful      StepKind = "stateful"
	StepJob           StepKind = "job"
	StepDeploy        StepKind = "deploy"
	StepFinish        StepKind = "finish"
)

// Step is one unit of work of a deployment.
type Step struct {
	Kind     StepKind
	Service  string
	Image    string
	Strategy spec.Strategy
	Phase    spec.Phase
	// Volumes are Docker volume names (ensure_volumes only).
	Volumes []string
	// Downtime is true when the service stops serving during the step.
	Downtime bool
	// ImageChanged is true when a running service gets a different image.
	ImageChanged bool
}

// Running is what currently runs for a service in the environment.
type Running struct {
	Image string
	// Fingerprint identifies the full container configuration.
	Fingerprint string
}

// Input is everything the planner needs.
type Input struct {
	App  string
	Env  string
	Spec spec.Spec
	// BuildImages maps services with a build section to their built image tag.
	BuildImages map[string]string
	// SecretNames are the keys of the secret set version to deploy.
	SecretNames []string
	// Fingerprints are the desired container fingerprints per service.
	Fingerprints map[string]string
	// Current is what runs now, by service.
	Current map[string]Running
}

// Plan is the ordered step list.
type Plan struct {
	Steps []Step
}

// Downtime reports whether any step stops a running service.
func (p Plan) Downtime() bool {
	return slices.ContainsFunc(p.Steps, func(s Step) bool { return s.Downtime })
}

// Make validates the input and computes the plan.
func Make(in Input) (Plan, error) {
	images, err := resolveImages(in)
	if err != nil {
		return Plan{}, err
	}
	if err := checkSecrets(in); err != nil {
		return Plan{}, err
	}
	order, cycle := spec.DependencyOrder(in.Spec)
	if cycle != nil {
		return Plan{}, domain.Invalid("spec", "depends_on has a cycle: %s", strings.Join(cycle, " -> "))
	}

	steps := []Step{{Kind: StepEnsureNetwork}}
	if vols := volumeNames(in); len(vols) > 0 {
		steps = append(steps, Step{Kind: StepEnsureVolumes, Volumes: vols})
	}
	byKind := func(match func(spec.Service) bool, mk func(spec.Service) (Step, bool)) {
		for _, name := range order {
			svc, _ := in.Spec.Service(name)
			if !match(svc) {
				continue
			}
			if step, ok := mk(svc); ok {
				steps = append(steps, step)
			}
		}
	}
	byKind(isKind(spec.KindStateful), func(svc spec.Service) (Step, bool) {
		cur, running := in.Current[svc.Name]
		if running && cur.Image == images[svc.Name] && cur.Fingerprint == in.Fingerprints[svc.Name] {
			return Step{}, false // unchanged: keep it running
		}
		return Step{
			Kind: StepStateful, Service: svc.Name, Image: images[svc.Name], Strategy: spec.StrategyRecreate,
			Downtime: running, ImageChanged: running && cur.Image != images[svc.Name],
		}, true
	})
	byKind(isJob(spec.PhasePreDeploy), jobStep(images))
	byKind(isKind(spec.KindDeployment), func(svc spec.Service) (Step, bool) {
		cur, running := in.Current[svc.Name]
		strategy := svc.EffectiveStrategy()
		return Step{
			Kind: StepDeploy, Service: svc.Name, Image: images[svc.Name], Strategy: strategy,
			Downtime: running && strategy == spec.StrategyRecreate, ImageChanged: running && cur.Image != images[svc.Name],
		}, true
	})
	byKind(isJob(spec.PhasePostDeploy), jobStep(images))
	steps = append(steps, Step{Kind: StepFinish})
	return Plan{Steps: steps}, nil
}

func isKind(k spec.Kind) func(spec.Service) bool {
	return func(s spec.Service) bool { return s.Kind == k }
}

func isJob(p spec.Phase) func(spec.Service) bool {
	return func(s spec.Service) bool { return s.Kind == spec.KindJob && s.EffectivePhase() == p }
}

func jobStep(images map[string]string) func(spec.Service) (Step, bool) {
	return func(svc spec.Service) (Step, bool) {
		return Step{Kind: StepJob, Service: svc.Name, Image: images[svc.Name], Phase: svc.EffectivePhase()}, true
	}
}

// resolveImages picks each service's image: the build's image for build
// services, the spec's image otherwise. All missing builds are reported.
func resolveImages(in Input) (map[string]string, error) {
	images := map[string]string{}
	var missing []string
	for _, svc := range in.Spec.Services {
		if svc.Build == nil {
			images[svc.Name] = svc.Image
			continue
		}
		img, ok := in.BuildImages[svc.Name]
		if !ok {
			missing = append(missing, svc.Name)
			continue
		}
		images[svc.Name] = img
	}
	if len(missing) > 0 {
		return nil, domain.Invalid("build", "the build has no image for %s; build again with the current spec", strings.Join(missing, ", "))
	}
	return images, nil
}

// checkSecrets reports every referenced secret missing from the secret set.
func checkSecrets(in Input) error {
	have := map[string]bool{}
	for _, n := range in.SecretNames {
		have[n] = true
	}
	var missing []string
	for _, svc := range in.Spec.Services {
		for _, ref := range svc.Secrets {
			if !have[ref.Name] {
				missing = append(missing, fmt.Sprintf("%s (used by %s)", ref.Name, svc.Name))
			}
		}
	}
	if len(missing) > 0 {
		return domain.Invalid("secrets", "missing in the environment: %s", strings.Join(missing, ", "))
	}
	return nil
}

func volumeNames(in Input) []string {
	var out []string
	for _, svc := range in.Spec.Services {
		for _, v := range svc.Volumes {
			out = append(out, domain.VolumeName(in.App, in.Env, svc.Name, v.Name))
		}
	}
	return out
}
