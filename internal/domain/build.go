package domain

import (
	"slices"
	"time"
)

// Audit actions for builds.
const (
	AuditBuildStarted  = "build.started"
	AuditBuildCanceled = "build.canceled"
)

// BuildStatus is the lifecycle state of a build.
type BuildStatus string

// Build statuses.
const (
	BuildQueued    BuildStatus = "queued"
	BuildRunning   BuildStatus = "running"
	BuildSucceeded BuildStatus = "succeeded"
	BuildFailed    BuildStatus = "failed"
	BuildCanceled  BuildStatus = "canceled"
)

// buildTransitions lists, for each status, the statuses it can be reached from.
var buildTransitions = map[BuildStatus][]BuildStatus{
	BuildRunning:   {BuildQueued},
	BuildSucceeded: {BuildRunning},
	BuildFailed:    {BuildQueued, BuildRunning},
	BuildCanceled:  {BuildQueued, BuildRunning},
}

// Terminal reports whether no further transition is possible.
func (s BuildStatus) Terminal() bool {
	return s == BuildSucceeded || s == BuildFailed || s == BuildCanceled
}

// BuildSourceStates returns the statuses a build may move to "to" from.
func BuildSourceStates(to BuildStatus) []BuildStatus {
	return slices.Clone(buildTransitions[to])
}

// CanTransition reports whether from -> to is allowed.
func (s BuildStatus) CanTransition(to BuildStatus) bool {
	return slices.Contains(buildTransitions[to], s)
}

// Build is one build of one commit of an app.
type Build struct {
	ID          int64
	AppID       int64
	Ref         string
	CommitSHA   string
	Status      BuildStatus
	TriggeredBy *int64
	Error       string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

// BuildImage is the image built for one service.
type BuildImage struct {
	BuildID  int64
	Service  string
	ImageTag string
	ImageID  string
}
