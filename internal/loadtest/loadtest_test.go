package loadtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunCountsFailures(t *testing.T) {
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1)%4 == 0 {
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	defer srv.Close()

	rep := Run(context.Background(), Options{URL: srv.URL, Rate: 200, Duration: 300 * time.Millisecond, Timeout: time.Second, Workers: 4})
	if rep.Total < 20 || rep.Non2xx == 0 || rep.OK == 0 || rep.Errors != 0 || rep.Failed() != rep.Non2xx {
		t.Fatalf("report %+v", rep)
	}
	if len(rep.Samples) == 0 || len(rep.Samples) > maxSamples {
		t.Fatalf("samples %v", rep.Samples)
	}
}

func TestRunConnectionErrors(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	rep := Run(context.Background(), Options{URL: url, Rate: 50, Duration: 100 * time.Millisecond, Timeout: 100 * time.Millisecond})
	if rep.Total == 0 || rep.Errors != rep.Total {
		t.Fatalf("report %+v", rep)
	}
}

func TestRunAllOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	rep := Run(context.Background(), Options{URL: srv.URL, Rate: 100, Duration: 200 * time.Millisecond, Timeout: time.Second, Workers: 2})
	if rep.Total == 0 || rep.Failed() != 0 {
		t.Fatalf("report %+v", rep)
	}
}
