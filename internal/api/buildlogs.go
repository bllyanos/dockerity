package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bllyanos/dockerity/internal/buildlog"
)

// streamLogs serves a build's log: the whole file as text/plain, or, for
// clients asking for text/event-stream, the file followed by live chunks as
// Server-Sent Events. Each "log" event's data is a JSON string; an "end" event
// closes the stream when the build finished.
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
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		data, err := h.logs.Store.Read(id)
		if err != nil {
			writeServiceError(w, r, h.logger, err)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(data)
		return
	}

	// Subscribe before reading the file so nothing written in between is lost;
	// chunk offsets drop what the file already contained.
	var live <-chan buildlog.Chunk
	if !b.Status.Terminal() {
		ch, unsubscribe := h.logs.Hub.Subscribe(id)
		defer unsubscribe()
		live = ch
	}
	data, err := h.logs.Store.Read(id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	rc := http.NewResponseController(w)
	send := func(event string, payload string) bool {
		encoded, _ := json.Marshal(payload)
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, encoded); err != nil {
			return false
		}
		return rc.Flush() == nil
	}

	if len(data) > 0 && !send("log", string(data)) {
		return
	}
	seen := int64(len(data))
	for live != nil {
		select {
		case <-r.Context().Done():
			return
		case c, open := <-live:
			if !open {
				live = nil
				continue
			}
			if part := buildlog.Since(c, seen); len(part) > 0 {
				seen = c.Offset + int64(len(c.Data))
				if !send("log", string(part)) {
					return
				}
			}
		}
	}
	send("end", "")
}
