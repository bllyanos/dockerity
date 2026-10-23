package backups

import (
	"context"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

// CurrentSpecs reads the spec of an environment's current deployment.
type CurrentSpecs struct {
	Deployments interface {
		CurrentDeployment(ctx context.Context, envID int64) (domain.Deployment, error)
	}
	Specs interface {
		Get(ctx context.Context, appID int64, version int) (domain.AppSpecVersion, error)
	}
}

// CurrentSpec implements Specs.
func (c CurrentSpecs) CurrentSpec(ctx context.Context, env domain.Environment) (spec.Spec, error) {
	d, err := c.Deployments.CurrentDeployment(ctx, env.ID)
	if errors.Is(err, domain.ErrNotFound) {
		return spec.Spec{}, domain.Conflictf("nothing is deployed in %s yet", env.Name)
	}
	if err != nil {
		return spec.Spec{}, err
	}
	sv, err := c.Specs.Get(ctx, env.AppID, d.SpecVersion)
	if err != nil {
		return spec.Spec{}, err
	}
	return spec.Load([]byte(sv.YAML))
}
