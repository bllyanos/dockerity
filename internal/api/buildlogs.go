package api

import "net/http"

// streamLogs serves a build's log, live while the build runs.
func (h buildHandlers) streamLogs(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	b, _, err := h.builds.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	serveLog(w, r, h.logger, h.logs, id, b.Status.Terminal())
}
