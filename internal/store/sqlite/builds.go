package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Builds is the build repository. Status changes are compare-and-set, so a
// build can never move through an invalid transition.
type Builds struct {
	DB *sql.DB
}

const buildColumns = `id, app_id, ref, commit_sha, status, triggered_by, error, created_at, started_at, finished_at`

// CreateBuild inserts a queued build.
func (r Builds) CreateBuild(ctx context.Context, b domain.Build) (domain.Build, error) {
	b.Status = domain.BuildQueued
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO builds (app_id, ref, status, triggered_by, created_at) VALUES (?, ?, ?, ?, ?)`,
		b.AppID, b.Ref, string(b.Status), b.TriggeredBy, formatTime(b.CreatedAt))
	if err != nil {
		return domain.Build{}, err
	}
	if b.ID, err = res.LastInsertId(); err != nil {
		return domain.Build{}, err
	}
	return b, nil
}

// GetBuild returns the build or domain.ErrNotFound.
func (r Builds) GetBuild(ctx context.Context, id int64) (domain.Build, error) {
	return scanBuild(r.DB.QueryRowContext(ctx, `SELECT `+buildColumns+` FROM builds WHERE id = ?`, id))
}

// ListBuilds returns the app's newest builds first.
func (r Builds) ListBuilds(ctx context.Context, appID int64, limit int) ([]domain.Build, error) {
	return r.query(ctx, `SELECT `+buildColumns+` FROM builds WHERE app_id = ? ORDER BY id DESC LIMIT ?`, appID, limit)
}

// ListBuildsByStatus returns builds in the given status, oldest first.
func (r Builds) ListBuildsByStatus(ctx context.Context, status domain.BuildStatus) ([]domain.Build, error) {
	return r.query(ctx, `SELECT `+buildColumns+` FROM builds WHERE status = ? ORDER BY id`, string(status))
}

// StartBuild moves a queued build to running with its resolved commit.
// It returns domain.ErrConflict if the build is no longer queued.
func (r Builds) StartBuild(ctx context.Context, id int64, commitSHA string, at time.Time) error {
	return r.transition(ctx, id, domain.BuildRunning,
		`commit_sha = ?, started_at = ?`, commitSHA, formatTime(at))
}

// FinishBuild moves a build to a terminal status. It returns
// domain.ErrConflict if the transition is not allowed from the current status.
func (r Builds) FinishBuild(ctx context.Context, id int64, to domain.BuildStatus, errMsg string, at time.Time) error {
	if !to.Terminal() {
		return domain.Invalid("status", "%s is not a terminal status", to)
	}
	return r.transition(ctx, id, to, `error = ?, finished_at = ?`, errMsg, formatTime(at))
}

// transition sets status to "to" plus extra columns, only from an allowed source status.
func (r Builds) transition(ctx context.Context, id int64, to domain.BuildStatus, set string, args ...any) error {
	from := domain.BuildSourceStates(to)
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(from)), ", ")
	query := `UPDATE builds SET status = ?, ` + set + ` WHERE id = ? AND status IN (` + placeholders + `)`
	all := append([]any{string(to)}, args...)
	all = append(all, id)
	for _, s := range from {
		all = append(all, string(s))
	}
	err := execOne(ctx, r.DB, query, all...)
	if errors.Is(err, domain.ErrNotFound) {
		if _, getErr := r.GetBuild(ctx, id); getErr != nil {
			return getErr
		}
		return domain.Conflictf("build %d cannot move to %s from its current status", id, to)
	}
	return err
}

// AddBuildImage records the image built for a service.
func (r Builds) AddBuildImage(ctx context.Context, img domain.BuildImage) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO build_images (build_id, service, image_tag, image_id) VALUES (?, ?, ?, ?)`,
		img.BuildID, img.Service, img.ImageTag, img.ImageID)
	return err
}

// ListBuildImages returns a build's images ordered by service.
func (r Builds) ListBuildImages(ctx context.Context, buildID int64) ([]domain.BuildImage, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT build_id, service, image_tag, image_id FROM build_images WHERE build_id = ? ORDER BY service`, buildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BuildImage
	for rows.Next() {
		var img domain.BuildImage
		if err := rows.Scan(&img.BuildID, &img.Service, &img.ImageTag, &img.ImageID); err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

func (r Builds) query(ctx context.Context, q string, args ...any) ([]domain.Build, error) {
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Build
	for rows.Next() {
		b, err := scanBuild(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func scanBuild(row rowScanner) (domain.Build, error) {
	var (
		b                 domain.Build
		status, created   string
		triggeredBy       sql.NullInt64
		started, finished sql.NullString
	)
	err := row.Scan(&b.ID, &b.AppID, &b.Ref, &b.CommitSHA, &status, &triggeredBy, &b.Error, &created, &started, &finished)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Build{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Build{}, err
	}
	b.Status = domain.BuildStatus(status)
	if triggeredBy.Valid {
		b.TriggeredBy = &triggeredBy.Int64
	}
	if b.CreatedAt, err = parseTime(created); err != nil {
		return domain.Build{}, err
	}
	if b.StartedAt, err = parseNullTime(started); err != nil {
		return domain.Build{}, err
	}
	if b.FinishedAt, err = parseNullTime(finished); err != nil {
		return domain.Build{}, err
	}
	return b, nil
}
