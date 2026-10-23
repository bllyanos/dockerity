package api

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/buildlog"
	"github.com/bllyanos/dockerity/internal/domain"
)

func buildsRouter(t *testing.T) (http.Handler, string, *fakeBuilds, LogSource, *fakeAudit) {
	t.Helper()
	d := fullDeps()
	logs := LogSource{Store: buildlog.Store{Dir: t.TempDir()}, Hub: buildlog.NewHub()}
	d.BuildLogs = logs
	raw, _, _ := d.Tokens.(*fakeTokens).Create(t.Context(), alice, "t")
	return NewRouter(d), raw, d.Builds.(*fakeBuilds), logs, d.Audit.(*fakeAudit)
}

func TestBuildsAPI(t *testing.T) {
	h, raw, fb, _, log := buildsRouter(t)

	rec := bearer(h, http.MethodPost, APIPrefix+"/apps/1/builds", raw, `{"ref":"v1"}`)
	if rec.Code != http.StatusAccepted || !strings.Contains(rec.Body.String(), `"ref":"v1"`) {
		t.Fatalf("trigger: %d %s", rec.Code, rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/apps/1/builds", raw, `{}`); rec.Code != http.StatusAccepted || fb.triggered[1] != "main" {
		t.Fatalf("trigger default: %d %v", rec.Code, fb.triggered)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/apps/1/builds", raw, ""); !strings.Contains(rec.Body.String(), `"status":"running"`) {
		t.Fatalf("list: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodGet, APIPrefix+"/builds/2", raw, ""); !strings.Contains(rec.Body.String(), `"imageTag":"dkt/shop/api:abc"`) {
		t.Fatalf("get: %s", rec.Body)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/builds/1/cancel", raw, ""); rec.Code != http.StatusAccepted {
		t.Fatalf("cancel: %d", rec.Code)
	}
	if rec = bearer(h, http.MethodPost, APIPrefix+"/builds/2/cancel", raw, ""); rec.Code != http.StatusConflict {
		t.Fatalf("cancel finished: %d", rec.Code)
	}
	want := []string{domain.AuditTokenUsed, domain.AuditBuildStarted, domain.AuditBuildStarted, domain.AuditBuildCanceled}
	if got := log.actions(); !slices.Equal(got, want) {
		t.Fatalf("audit %v", got)
	}
}

func TestBuildLogsPlainText(t *testing.T) {
	h, raw, _, logs, _ := buildsRouter(t)
	w, _ := logs.Store.Open(2)
	_, _ = io.WriteString(w, "step 1\nstep 2\n")
	_ = w.Close()
	rec := bearer(h, http.MethodGet, APIPrefix+"/builds/2/logs", raw, "")
	if rec.Code != http.StatusOK || rec.Body.String() != "step 1\nstep 2\n" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("plain log: %d %q", rec.Code, rec.Body)
	}
}

// sseEvents reads "event:"/"data:" pairs until the stream ends.
func sseEvents(t *testing.T, body io.Reader) [][2]string {
	t.Helper()
	var out [][2]string
	var event string
	sc := bufio.NewScanner(body)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			out = append(out, [2]string{event, strings.TrimPrefix(line, "data: ")})
		}
	}
	return out
}

func sseRequest(t *testing.T, srv *httptest.Server, raw, path string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+APIPrefix+path, nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("Accept", "text/event-stream")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("sse: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	return res
}

func TestBuildLogsStreamLive(t *testing.T) {
	h, raw, _, logs, _ := buildsRouter(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	sink, err := logs.Store.OpenSink(1, logs.Hub)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(sink, "before\n")

	res := sseRequest(t, srv, raw, "/builds/1/logs")
	defer res.Body.Close()
	go func() {
		time.Sleep(50 * time.Millisecond)
		_, _ = io.WriteString(sink, "during\n")
		_ = sink.Close()
	}()

	events := sseEvents(t, res.Body)
	want := [][2]string{{"log", `"before\n"`}, {"log", `"during\n"`}, {"end", `""`}}
	if !slices.Equal(events, want) {
		t.Fatalf("events %v, want %v", events, want)
	}
}

func TestBuildLogsStreamFinishedBuild(t *testing.T) {
	h, raw, _, logs, _ := buildsRouter(t)
	srv := httptest.NewServer(h)
	defer srv.Close()
	w, _ := logs.Store.Open(2)
	_, _ = io.WriteString(w, "all done\n")
	_ = w.Close()

	res := sseRequest(t, srv, raw, "/builds/2/logs")
	defer res.Body.Close()
	events := sseEvents(t, res.Body)
	if len(events) != 2 || events[0][1] != `"all done\n"` || events[1][0] != "end" {
		t.Fatalf("events %v", events)
	}
}
