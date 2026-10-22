package sqlite

import (
	"context"
	"database/sql"
)

// HealthCheck reports whether the database answers queries.
type HealthCheck struct {
	DB *sql.DB
}

// Name implements api.HealthCheck.
func (HealthCheck) Name() string { return "database" }

// Check implements api.HealthCheck.
func (h HealthCheck) Check(ctx context.Context) error {
	return h.DB.PingContext(ctx)
}
