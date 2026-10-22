package dockerx

import (
	"context"
	"errors"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestHealthCheck(t *testing.T) {
	hc := HealthCheck{Docker: fakePinger{}}
	if hc.Name() != "docker" {
		t.Fatalf("name %q", hc.Name())
	}
	if err := hc.Check(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	down := errors.New("socket missing")
	if err := (HealthCheck{Docker: fakePinger{err: down}}).Check(context.Background()); !errors.Is(err, down) {
		t.Fatalf("got %v, want %v", err, down)
	}
}
