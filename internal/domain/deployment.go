package domain

import (
	"slices"
	"time"
)

// Audit actions for deployments.
const (
	AuditDeploymentStarted  = "deployment.started"
	AuditDeploymentRollback = "deployment.rollback"
)

// DeploymentStatus is the lifecycle state of a deployment.
type DeploymentStatus string

// Deployment statuses.
const (
	DeploymentPending   DeploymentStatus = "pending"
	DeploymentRunning   DeploymentStatus = "running"
	DeploymentSucceeded DeploymentStatus = "succeeded"
	DeploymentFailed    DeploymentStatus = "failed"
	DeploymentCanceled  DeploymentStatus = "canceled"
)

var deploymentTransitions = map[DeploymentStatus][]DeploymentStatus{
	DeploymentRunning:   {DeploymentPending},
	DeploymentSucceeded: {DeploymentRunning},
	DeploymentFailed:    {DeploymentPending, DeploymentRunning},
	DeploymentCanceled:  {DeploymentPending},
}

// Terminal reports whether no further transition is possible.
func (s DeploymentStatus) Terminal() bool {
	return s == DeploymentSucceeded || s == DeploymentFailed || s == DeploymentCanceled
}

// CanTransition reports whether from -> to is allowed.
func (s DeploymentStatus) CanTransition(to DeploymentStatus) bool {
	return slices.Contains(deploymentTransitions[to], s)
}

// DeploymentSourceStates returns the statuses a deployment may move to "to" from.
func DeploymentSourceStates(to DeploymentStatus) []DeploymentStatus {
	return slices.Clone(deploymentTransitions[to])
}

// Deployment is "build X with spec Y and secrets Z on environment E".
// Deployments are immutable history; rollback is a new deployment.
type Deployment struct {
	ID            int64
	EnvID         int64
	BuildID       int64
	SpecVersion   int
	SecretVersion int // 0 when the environment has no secrets
	Status        DeploymentStatus
	Current       bool
	TriggeredBy   *int64
	Error         string
	CreatedAt     time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
}

// StepStatus is the state of one deployment step.
type StepStatus string

// Step statuses.
const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepSucceeded StepStatus = "succeeded"
	StepFailed    StepStatus = "failed"
	StepSkipped   StepStatus = "skipped"
)

// DeploymentStep records one planned step and its outcome.
type DeploymentStep struct {
	ID           int64
	DeploymentID int64
	Position     int
	Kind         string
	Service      string
	Status       StepStatus
	ExitCode     *int
	Message      string
	StartedAt    *time.Time
	FinishedAt   *time.Time
}
