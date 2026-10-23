// Package build turns an app spec and a checked-out commit into images.
package build

import (
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

// DefaultDockerfile is used when a service does not name one.
const DefaultDockerfile = "Dockerfile"

// ServiceBuild is what to build for one service.
type ServiceBuild struct {
	Service    string
	Context    string // relative to the repo root
	Dockerfile string // relative to the context
	Target     string
	Args       map[string]string
	ImageTag   string
	Labels     map[string]string
}

// Plan lists the services of s that are built from the repo, in spec order.
// Services using a public image are skipped. Build args come only from the
// spec, never from secrets.
func Plan(app string, s spec.Spec, commitSHA string) []ServiceBuild {
	var out []ServiceBuild
	for _, svc := range s.Services {
		if svc.Build == nil {
			continue
		}
		b := ServiceBuild{
			Service:    svc.Name,
			Context:    svc.Build.Context,
			Dockerfile: svc.Build.Dockerfile,
			Target:     svc.Build.Target,
			Args:       map[string]string{},
			ImageTag:   domain.ImageTag(app, svc.Name, commitSHA),
			Labels:     domain.ImageLabels(app, svc.Name, commitSHA),
		}
		if b.Dockerfile == "" {
			b.Dockerfile = DefaultDockerfile
		}
		for _, a := range svc.Build.Args {
			b.Args[a.Name] = a.Value
		}
		out = append(out, b)
	}
	return out
}
