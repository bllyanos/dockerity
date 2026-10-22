package api

import (
	"context"
	"net/http"
	"time"
)

// HealthCheck reports whether one dependency (database, Docker, ...) is usable.
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
}

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

const healthTimeout = 3 * time.Second

// healthHandler returns 200 when every check passes, otherwise 503.
func healthHandler(checks []HealthCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
		defer cancel()

		resp := healthResponse{Status: "ok", Checks: map[string]string{}}
		code := http.StatusOK
		for _, c := range checks {
			if err := c.Check(ctx); err != nil {
				resp.Checks[c.Name()] = err.Error()
				resp.Status = "unavailable"
				code = http.StatusServiceUnavailable
				continue
			}
			resp.Checks[c.Name()] = "ok"
		}
		writeJSON(w, code, resp)
	}
}
