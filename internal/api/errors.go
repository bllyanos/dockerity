package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/bllyanos/dockerity/internal/domain"
)

// writeServiceError maps domain errors to HTTP responses. Unknown errors are
// logged and reported as 500 without leaking details.
func writeServiceError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, ve.Error())
	// Not-found and conflict messages are written by services for the caller.
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	default:
		logger.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
