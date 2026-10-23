package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Deployments is the deployment repository. The environment lock lives in the
// schema: a second pending or running deployment cannot be inserted.
type Deployments struct {
	DB *sql.DB
}

const deploymentColumns = `id, env_id, build_id, spec_version, secret_version, status, current, triggered_by, error, created_at, started_at, finished_at`

// CreateDeployment inserts a pending deployment. If the environment already
// has a pending or running one, it returns domain.ErrConflict.
func (r Deployments) CreateDeployment(ctx context.Context, d domain.Deployment) (domain.Deployment, error) {
	d.Status = domain.DeploymentPending
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO deployments (env_id, build_id, spec_version, secret_version, status, triggered_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		d.EnvID, d.BuildID, d.SpecVersion, d.SecretVersion, string(d.Status), d.TriggeredBy, formatTime(d.CreatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Deployment{}, domain.ErrConflict
		}
		return domain.Deployment{}, err
	}
	if d.ID, err = res.LastInsertId(); err != nil {
		return domain.Deployment{}, err
	}
	return d, nil
}

// GetDeployment returns the deployment or domain.ErrNotFound.
func (r Deployments) GetDeployment(ctx context.Context, id int64) (domain.Deployment, error) {
	return scanDeployment(r.DB.QueryRowContext(ctx, `SELECT `+deploymentColumns+` FROM deployments WHERE id = ?`, id))
}

// ListDeployments returns an environment's deployments, newest first.
func (r Deployments) ListDeployments(ctx context.Context, envID int64, limit int) ([]domain.Deployment, error) {
	return r.query(ctx, `SELECT `+deploymentColumns+` FROM deployments WHERE env_id = ? ORDER BY id DESC LIMIT ?`, envID, limit)
}

// ListDeploymentsByStatus returns deployments in a status, oldest first.
func (r Deployments) ListDeploymentsByStatus(ctx context.Context, status domain.DeploymentStatus) ([]domain.Deployment, error) {
	return r.query(ctx, `SELECT `+deploymentColumns+` FROM deployments WHERE status = ? ORDER BY id`, string(status))
}

// CurrentDeployment returns the environment's current deployment or domain.ErrNotFound.
func (r Deployments) CurrentDeployment(ctx context.Context, envID int64) (domain.Deployment, error) {
	return scanDeployment(r.DB.QueryRowContext(ctx,
		`SELECT `+deploymentColumns+` FROM deployments WHERE env_id = ? AND current = 1`, envID))
}

// StartDeployment moves a pending deployment to running.
func (r Deployments) StartDeployment(ctx context.Context, id int64, at time.Time) error {
	return r.transition(ctx, id, domain.DeploymentRunning, `started_at = ?`, formatTime(at))
}

// FinishDeployment moves a deployment to a terminal status. A succeeded
// deployment becomes the environment's current one, atomically.
func (r Deployments) FinishDeployment(ctx context.Context, id int64, to domain.DeploymentStatus, errMsg string, at time.Time) error {
	if !to.Terminal() {
		return domain.Invalid("status", "%s is not a terminal status", to)
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := transitionTx(ctx, tx, id, to, `error = ?, finished_at = ?`, errMsg, formatTime(at)); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			if _, getErr := r.GetDeployment(ctx, id); getErr != nil {
				return getErr
			}
			return domain.Conflictf("deployment %d cannot move to %s from its current status", id, to)
		}
		return err
	}
	if to == domain.DeploymentSucceeded {
		if _, err := tx.ExecContext(ctx,
			`UPDATE deployments SET current = 0 WHERE env_id = (SELECT env_id FROM deployments WHERE id = ?) AND current = 1`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE deployments SET current = 1 WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r Deployments) transition(ctx context.Context, id int64, to domain.DeploymentStatus, set string, args ...any) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := transitionTx(ctx, tx, id, to, set, args...); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			if _, getErr := r.GetDeployment(ctx, id); getErr != nil {
				return getErr
			}
			return domain.Conflictf("deployment %d cannot move to %s from its current status", id, to)
		}
		return err
	}
	return tx.Commit()
}

func transitionTx(ctx context.Context, tx *sql.Tx, id int64, to domain.DeploymentStatus, set string, args ...any) error {
	from := domain.DeploymentSourceStates(to)
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(from)), ", ")
	all := append([]any{string(to)}, args...)
	all = append(all, id)
	for _, s := range from {
		all = append(all, string(s))
	}
	res, err := tx.ExecContext(ctx, `UPDATE deployments SET status = ?, `+set+` WHERE id = ? AND status IN (`+placeholders+`)`, all...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r Deployments) query(ctx context.Context, q string, args ...any) ([]domain.Deployment, error) {
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Deployment
	for rows.Next() {
		d, err := scanDeployment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanDeployment(row rowScanner) (domain.Deployment, error) {
	var (
		d                 domain.Deployment
		status, created   string
		triggeredBy       sql.NullInt64
		started, finished sql.NullString
	)
	err := row.Scan(&d.ID, &d.EnvID, &d.BuildID, &d.SpecVersion, &d.SecretVersion, &status, &d.Current,
		&triggeredBy, &d.Error, &created, &started, &finished)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Deployment{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Deployment{}, err
	}
	d.Status = domain.DeploymentStatus(status)
	if triggeredBy.Valid {
		d.TriggeredBy = &triggeredBy.Int64
	}
	if d.CreatedAt, err = parseTime(created); err != nil {
		return domain.Deployment{}, err
	}
	if d.StartedAt, err = parseNullTime(started); err != nil {
		return domain.Deployment{}, err
	}
	d.FinishedAt, err = parseNullTime(finished)
	return d, err
}
