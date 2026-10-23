package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/bllyanos/dockerity/internal/domain"
)

// execOne runs a statement that must affect a row; none affected is domain.ErrNotFound.
func execOne(ctx context.Context, db *sql.DB, query string, args ...any) error {
	res, err := db.ExecContext(ctx, query, args...)
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

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
