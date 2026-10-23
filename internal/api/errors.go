package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

// writeServiceError maps domain errors to HTTP responses. Unknown errors are
// logged and reported as 500 without leaking details.
func writeServiceError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	var (
		ve       *domain.ValidationError
		problems spec.Problems
	)
	switch {
	case errors.As(err, &problems):
		writeJSON(w, http.StatusBadRequest, problemsResponse(problems))
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

type problemView struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

type problemsBody struct {
	Error    string        `json:"error"`
	Problems []problemView `json:"problems"`
}

// problemsResponse lists spec problems so the UI can show them next to lines.
func problemsResponse(ps spec.Problems) problemsBody {
	body := problemsBody{Error: "the spec is invalid", Problems: make([]problemView, 0, len(ps))}
	for _, p := range ps {
		body.Problems = append(body.Problems, problemView{Line: p.Pos.Line, Column: p.Pos.Column, Message: p.Message})
	}
	return body
}
