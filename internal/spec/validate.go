package spec

import (
	"path"
	"regexp"
	"strings"

	"github.com/bllyanos/dockerity/internal/domain"
)

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Names end up in container, network and volume names, so they must be DNS labels.
const nameRule = domain.LabelRule

// Load parses and validates src.
func Load(src []byte) (Spec, error) {
	s, err := Parse(src)
	if err != nil {
		return Spec{}, err
	}
	if err := Validate(s); err != nil {
		return Spec{}, err
	}
	return s, nil
}

// Validate checks the semantic rules of docs/design.md section 4 and returns
// every violation as Problems, sorted by position.
func Validate(s Spec) error {
	var ps Problems
	if s.Name != "" && !domain.IsDNSLabel(s.Name) {
		ps.add(s.NamePos, "name %q %s", s.Name, nameRule)
	}
	if len(s.Services) == 0 {
		ps.add(s.NamePos, "the spec must define at least one service")
	}
	for _, svc := range s.Services {
		validateService(&ps, s, svc)
	}
	if _, cycle := DependencyOrder(s); cycle != nil {
		first, _ := s.Service(cycle[0])
		ps.add(first.At("depends_on"), "depends_on has a cycle: %s", strings.Join(cycle, " -> "))
	}
	return ps.sorted()
}

func validateService(ps *Problems, s Spec, svc Service) {
	if !domain.IsDNSLabel(svc.Name) {
		ps.add(svc.Pos, "service name %q %s", svc.Name, nameRule)
	}
	validateKind(ps, svc)
	validateSource(ps, svc)
	validateEnvAndSecrets(ps, svc)
	validateVolumes(ps, svc)
	validateDependsOn(ps, s, svc)
	validateHTTP(ps, svc)
	validateHealthcheck(ps, svc)
	validateResources(ps, svc)
}

func validateKind(ps *Problems, svc Service) {
	switch svc.Kind {
	case KindJob, KindDeployment, KindStateful:
	case "":
		ps.add(svc.Pos, "service %s needs a kind: job, deployment or stateful", svc.Name)
		return
	default:
		ps.add(svc.At("kind"), "kind %q must be job, deployment or stateful", svc.Kind)
		return
	}

	if svc.Phase != "" {
		if svc.Kind != KindJob {
			ps.add(svc.At("phase"), "phase is only allowed on jobs")
		} else if svc.Phase != PhasePreDeploy && svc.Phase != PhasePostDeploy {
			ps.add(svc.At("phase"), "phase %q must be pre-deploy or post-deploy", svc.Phase)
		}
	}

	if svc.Strategy != "" {
		switch {
		case svc.Kind == KindStateful:
			ps.add(svc.At("strategy"), "stateful services always use recreate (two containers must never share a volume); remove strategy")
		case svc.Kind == KindJob:
			ps.add(svc.At("strategy"), "strategy is only allowed on deployments")
		case svc.Strategy != StrategyBlueGreen && svc.Strategy != StrategyRecreate:
			ps.add(svc.At("strategy"), "strategy %q must be blue-green or recreate", svc.Strategy)
		}
	}

	if svc.EffectiveStrategy() == StrategyBlueGreen && svc.Healthcheck == nil {
		at := svc.At("strategy")
		ps.add(at, "service %s uses blue-green and needs a healthcheck so traffic only moves to a healthy container", svc.Name)
	}
}

func validateSource(ps *Problems, svc Service) {
	switch {
	case svc.Image == "" && svc.Build == nil:
		ps.add(svc.Pos, "service %s needs either image or build", svc.Name)
	case svc.Image != "" && svc.Build != nil:
		ps.add(svc.At("build"), "service %s cannot have both image and build", svc.Name)
	case svc.Build != nil:
		ctx := svc.Build.Context
		if ctx == "" {
			ps.add(svc.At("build"), "build.context is required")
		} else if path.IsAbs(ctx) || path.Clean(ctx) == ".." || strings.HasPrefix(path.Clean(ctx), "../") {
			ps.add(svc.At("build"), "build.context %q must be a path inside the repo", ctx)
		}
		for _, a := range svc.Build.Args {
			if !envNamePattern.MatchString(a.Name) {
				ps.add(a.Pos, "build arg %q is not a valid name", a.Name)
			}
		}
	}
}

func validateEnvAndSecrets(ps *Problems, svc Service) {
	env := map[string]bool{}
	for _, e := range svc.Env {
		if !envNamePattern.MatchString(e.Name) {
			ps.add(e.Pos, "env name %q must be letters, digits and '_', not starting with a digit", e.Name)
		}
		env[e.Name] = true
	}
	seen := map[string]bool{}
	files := map[string]bool{}
	for _, ref := range svc.Secrets {
		if err := domain.ValidateSecretKey(ref.Name); err != nil {
			ps.add(ref.Pos, "secret name %q must be uppercase letters, digits and '_'", ref.Name)
			continue
		}
		if seen[ref.Name] {
			ps.add(ref.Pos, "secret %s is listed twice", ref.Name)
		}
		seen[ref.Name] = true
		if ref.File == "" && env[ref.Name] {
			ps.add(ref.Pos, "%s is both a plain env var and a secret; secrets must not be stored in env", ref.Name)
		}
		if ref.File != "" {
			if !path.IsAbs(ref.File) || path.Clean(ref.File) != ref.File || ref.File == "/" {
				ps.add(ref.Pos, "secret file %q must be an absolute, clean path", ref.File)
			} else if files[ref.File] {
				ps.add(ref.Pos, "secret file %s is used twice", ref.File)
			}
			files[ref.File] = true
		}
	}
}

func validateVolumes(ps *Problems, svc Service) {
	if len(svc.Volumes) > 0 && svc.Kind != KindStateful && svc.Kind != "" {
		ps.add(svc.At("volumes"), "volumes are only allowed on stateful services; %s is a %s", svc.Name, svc.Kind)
		return
	}
	paths := map[string]bool{}
	for _, v := range svc.Volumes {
		if !domain.IsDNSLabel(v.Name) {
			ps.add(v.Pos, "volume name %q %s", v.Name, nameRule)
		}
		if !path.IsAbs(v.Path) || path.Clean(v.Path) != v.Path || v.Path == "/" {
			ps.add(v.Pos, "volume path %q must be an absolute, clean path", v.Path)
		} else if paths[v.Path] {
			ps.add(v.Pos, "volume path %s is mounted twice", v.Path)
		}
		paths[v.Path] = true
	}
}

func validateDependsOn(ps *Problems, s Spec, svc Service) {
	seen := map[string]bool{}
	for _, d := range svc.DependsOn {
		switch {
		case d.Name == svc.Name:
			ps.add(d.Pos, "service %s cannot depend on itself", svc.Name)
		case seen[d.Name]:
			ps.add(d.Pos, "dependency %s is listed twice", d.Name)
		default:
			if _, ok := s.Service(d.Name); !ok {
				ps.add(d.Pos, "depends_on refers to unknown service %q", d.Name)
			}
		}
		seen[d.Name] = true
	}
}

func validateHTTP(ps *Problems, svc Service) {
	if svc.HTTP == nil {
		return
	}
	if svc.Kind == KindJob {
		ps.add(svc.At("http"), "jobs cannot receive HTTP traffic")
	}
	if svc.HTTP.Port < 1 || svc.HTTP.Port > 65535 {
		ps.add(svc.At("http"), "http.port must be between 1 and 65535")
	}
	if svc.HTTP.Path != "" && !strings.HasPrefix(svc.HTTP.Path, "/") {
		ps.add(svc.At("http"), "http.path %q must start with /", svc.HTTP.Path)
	}
}

func validateHealthcheck(ps *Problems, svc Service) {
	hc := svc.Healthcheck
	if hc == nil {
		return
	}
	at := svc.At("healthcheck")
	if svc.Kind == KindJob {
		ps.add(at, "jobs cannot have a healthcheck")
	}
	switch {
	case len(hc.Cmd) == 0 && hc.HTTP == "":
		ps.add(at, "healthcheck needs cmd or http")
	case len(hc.Cmd) > 0 && hc.HTTP != "":
		ps.add(at, "healthcheck cannot have both cmd and http")
	case hc.HTTP != "":
		if !strings.HasPrefix(hc.HTTP, "/") {
			ps.add(at, "healthcheck.http %q must be a path starting with /", hc.HTTP)
		}
		if svc.HTTP == nil {
			ps.add(at, "healthcheck.http needs http.port to know which port to probe")
		}
	}
	if hc.Retries < 0 {
		ps.add(at, "healthcheck.retries cannot be negative")
	}
}

func validateResources(ps *Problems, svc Service) {
	r := svc.Resources
	if r == nil {
		return
	}
	if r.Memory != "" {
		if _, err := ParseMemory(r.Memory); err != nil {
			ps.add(r.MemoryPos, "%s", err)
		}
	}
	if r.CPUs != "" {
		if _, err := ParseCPUs(r.CPUs); err != nil {
			ps.add(r.CPUsPos, "%s", err)
		}
	}
}
