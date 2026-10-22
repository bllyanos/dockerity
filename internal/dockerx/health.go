package dockerx

import "context"

// Pinger is the part of Client a health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthCheck reports whether the Docker engine is reachable.
type HealthCheck struct {
	Docker Pinger
}

// Name implements api.HealthCheck.
func (HealthCheck) Name() string { return "docker" }

// Check implements api.HealthCheck.
func (h HealthCheck) Check(ctx context.Context) error {
	return h.Docker.Ping(ctx)
}
