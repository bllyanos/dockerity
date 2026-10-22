package healthprobe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestURL(t *testing.T) {
	cases := map[string]string{
		":8080":          "http://127.0.0.1:8080/healthz",
		"0.0.0.0:80":     "http://127.0.0.1:80/healthz",
		"[::]:80":        "http://127.0.0.1:80/healthz",
		"10.0.0.5:9000":  "http://10.0.0.5:9000/healthz",
		"localhost:1234": "http://localhost:1234/healthz",
	}
	for in, want := range cases {
		got, err := URL(in)
		if err != nil || got != want {
			t.Errorf("URL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := URL("nope"); err == nil {
		t.Error("expected error for address without port")
	}
}

func TestProbe(t *testing.T) {
	code := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}))
	defer srv.Close()

	if err := Probe(context.Background(), srv.URL, time.Second); err != nil {
		t.Fatalf("healthy: %v", err)
	}
	code = http.StatusServiceUnavailable
	if err := Probe(context.Background(), srv.URL, time.Second); err == nil {
		t.Fatal("expected error on 503")
	}
}
