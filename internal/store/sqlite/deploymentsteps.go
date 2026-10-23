package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// InsertSteps stores a deployment's planned steps, all pending, and returns them with IDs.
func (r Deployments) InsertSteps(ctx context.Context, deploymentID int64, steps []domain.DeploymentStep) ([]domain.DeploymentStep, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	out := make([]domain.DeploymentStep, len(steps))
	for i, s := range steps {
		s.DeploymentID, s.Position, s.Status = deploymentID, i+1, domain.StepPending
		res, err := tx.ExecContext(ctx,
			`INSERT INTO deployment_steps (deployment_id, position, kind, service, status) VALUES (?, ?, ?, ?, ?)`,
			s.DeploymentID, s.Position, s.Kind, s.Service, string(s.Status))
		if err != nil {
			return nil, err
		}
		if s.ID, err = res.LastInsertId(); err != nil {
			return nil, err
		}
		out[i] = s
	}
	return out, tx.Commit()
}

// StartStep marks a step running.
func (r Deployments) StartStep(ctx context.Context, stepID int64, at time.Time) error {
	return execOne(ctx, r.DB, `UPDATE deployment_steps SET status = ?, started_at = ? WHERE id = ?`,
		string(domain.StepRunning), formatTime(at), stepID)
}

// FinishStep records a step's outcome.
func (r Deployments) FinishStep(ctx context.Context, stepID int64, status domain.StepStatus, exitCode *int, message string, at time.Time) error {
	return execOne(ctx, r.DB, `UPDATE deployment_steps SET status = ?, exit_code = ?, message = ?, finished_at = ? WHERE id = ?`,
		string(status), exitCode, message, formatTime(at), stepID)
}

// SkipPendingSteps marks the remaining steps of a deployment skipped.
func (r Deployments) SkipPendingSteps(ctx context.Context, deploymentID int64) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE deployment_steps SET status = ? WHERE deployment_id = ? AND status = ?`,
		string(domain.StepSkipped), deploymentID, string(domain.StepPending))
	return err
}

// ListSteps returns a deployment's steps in order.
func (r Deployments) ListSteps(ctx context.Context, deploymentID int64) ([]domain.DeploymentStep, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, deployment_id, position, kind, service, status, exit_code, message, started_at, finished_at
		FROM deployment_steps WHERE deployment_id = ? ORDER BY position`, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DeploymentStep
	for rows.Next() {
		var (
			s                 domain.DeploymentStep
			status            string
			exit              sql.NullInt64
			started, finished sql.NullString
		)
		if err := rows.Scan(&s.ID, &s.DeploymentID, &s.Position, &s.Kind, &s.Service, &status, &exit, &s.Message, &started, &finished); err != nil {
			return nil, err
		}
		s.Status = domain.StepStatus(status)
		if exit.Valid {
			code := int(exit.Int64)
			s.ExitCode = &code
		}
		if s.StartedAt, err = parseNullTime(started); err != nil {
			return nil, err
		}
		if s.FinishedAt, err = parseNullTime(finished); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
