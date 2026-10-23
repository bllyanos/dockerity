// Package spawn creates an ephemeral environment and deploys a build into it
// in one action (docs/design.md section 6.3).
package spawn

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// Input is what a spawn request carries.
type Input struct {
	// Name of the environment; generated when empty.
	Name    string
	BuildID int64
	// TemplateEnvID, when set, is an environment of the same app whose latest
	// secrets are copied into the new one.
	TemplateEnvID *int64
	// TTL of the environment; the default when zero.
	TTL time.Duration
}

// Envs creates, reads and deletes environments.
type Envs interface {
	Get(ctx context.Context, id int64) (domain.Environment, error)
	CreateEphemeral(ctx context.Context, appID int64, name string, ttl time.Duration) (domain.Environment, error)
	Delete(ctx context.Context, id int64, skipBackup bool) (domain.Environment, error)
}

// Deployments starts deployments.
type Deployments interface {
	Start(ctx context.Context, envID, buildID int64, actor *int64) (domain.Deployment, error)
}

// Service spawns ephemeral environments.
type Service struct {
	Envs        Envs
	Secrets     secrets.SecretStore
	Deployments Deployments
}

// Spawn creates the environment, copies the template's secrets as its first
// secret version, and starts deploying the build. Volumes start empty. If the
// secrets cannot be copied or the deployment cannot start, the new
// environment is removed again.
func (s *Service) Spawn(ctx context.Context, appID int64, in Input, actor *int64) (domain.Environment, domain.Deployment, error) {
	var values map[string]string
	if in.TemplateEnvID != nil {
		tpl, err := s.Envs.Get(ctx, *in.TemplateEnvID)
		if err != nil {
			return domain.Environment{}, domain.Deployment{}, err
		}
		if tpl.AppID != appID {
			return domain.Environment{}, domain.Deployment{}, domain.Invalid("templateEnvId", "environment %s belongs to another app", tpl.Name)
		}
		if values, err = s.latestValues(ctx, tpl.ID); err != nil {
			return domain.Environment{}, domain.Deployment{}, err
		}
	}
	name := in.Name
	if name == "" {
		var err error
		if name, err = generateName(); err != nil {
			return domain.Environment{}, domain.Deployment{}, err
		}
	}
	env, err := s.Envs.CreateEphemeral(ctx, appID, name, in.TTL)
	if err != nil {
		return domain.Environment{}, domain.Deployment{}, err
	}
	rollback := func(cause error) (domain.Environment, domain.Deployment, error) {
		if _, err := s.Envs.Delete(context.Background(), env.ID, true); err != nil {
			cause = errors.Join(cause, err)
		}
		return domain.Environment{}, domain.Deployment{}, cause
	}
	if len(values) > 0 {
		if _, err := s.Secrets.Update(ctx, secrets.EnvScope(env.ID), secrets.Changes{Set: values}, actor); err != nil {
			return rollback(err)
		}
	}
	d, err := s.Deployments.Start(ctx, env.ID, in.BuildID, actor)
	if err != nil {
		return rollback(err)
	}
	return env, d, nil
}

func (s *Service) latestValues(ctx context.Context, envID int64) (map[string]string, error) {
	scope := secrets.EnvScope(envID)
	v, err := s.Secrets.Latest(ctx, scope)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.Secrets.Values(ctx, scope, v.Version)
}

// generateName returns a short random environment name such as "e-3f9a1c".
func generateName() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "e-" + hex.EncodeToString(b), nil
}
