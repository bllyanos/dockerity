// Package executor runs planned deployments against Docker
// (docs/design.md section 6.2).
package executor

import (
	"context"
	"io"
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/containerspec"
	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// Docker is the part of dockerx the executor uses.
type Docker interface {
	EnsureNetwork(ctx context.Context, name string, labels map[string]string) error
	EnsureVolume(ctx context.Context, name string, labels map[string]string) error
	CreateContainer(ctx context.Context, s containerspec.Spec) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, grace time.Duration) error
	RemoveContainer(ctx context.Context, id string) error
	WaitHealthy(ctx context.Context, id string, s containerspec.Spec) error
	WaitExit(ctx context.Context, id string) (int, error)
	Logs(ctx context.Context, id string, follow bool, w io.Writer) error
	ListManaged(ctx context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error)
}

// Store persists deployments and their steps.
type Store interface {
	GetDeployment(ctx context.Context, id int64) (domain.Deployment, error)
	CurrentDeployment(ctx context.Context, envID int64) (domain.Deployment, error)
	ListDeploymentsByStatus(ctx context.Context, status domain.DeploymentStatus) ([]domain.Deployment, error)
	StartDeployment(ctx context.Context, id int64, at time.Time) error
	FinishDeployment(ctx context.Context, id int64, to domain.DeploymentStatus, errMsg string, at time.Time) error
	InsertSteps(ctx context.Context, deploymentID int64, steps []domain.DeploymentStep) ([]domain.DeploymentStep, error)
	StartStep(ctx context.Context, stepID int64, at time.Time) error
	FinishStep(ctx context.Context, stepID int64, status domain.StepStatus, exitCode *int, message string, at time.Time) error
	SkipPendingSteps(ctx context.Context, deploymentID int64) error
}

// Envs reads environments.
type Envs interface {
	Get(ctx context.Context, id int64) (domain.Environment, error)
}

// Specs reads spec versions.
type Specs interface {
	Get(ctx context.Context, appID int64, version int) (domain.AppSpecVersion, error)
}

// BuildImages reads a build's images.
type BuildImages interface {
	ListBuildImages(ctx context.Context, buildID int64) ([]domain.BuildImage, error)
}

// Secrets decrypts secret set versions.
type Secrets interface {
	Values(ctx context.Context, scope secrets.Scope, version int) (map[string]string, error)
}
