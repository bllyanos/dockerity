package sqlite

import (
	"context"
	"maps"
	"testing"
	"time"
)

func TestSettings(t *testing.T) {
	r := Settings{DB: migratedDB(t)}
	ctx := context.Background()
	if got, err := r.AllSettings(ctx); err != nil || len(got) != 0 {
		t.Fatalf("empty: %v %v", got, err)
	}
	if err := r.SaveSettings(ctx, map[string]string{"a": "1", "b": "2"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveSettings(ctx, map[string]string{"a": "3"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, _ := r.AllSettings(ctx)
	if !maps.Equal(got, map[string]string{"a": "3", "b": "2"}) {
		t.Fatalf("got %v", got)
	}
}
