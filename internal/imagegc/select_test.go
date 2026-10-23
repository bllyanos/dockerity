package imagegc

import (
	"slices"
	"testing"
)

func TestSelect(t *testing.T) {
	images := []Image{
		{ID: "sha256:old", Tags: []string{"dkt/shop/web:aaa"}},
		{ID: "sha256:kept", Tags: []string{"dkt/shop/web:bbb"}},
		{ID: "sha256:used-by-tag", Tags: []string{"dkt/shop/web:ccc"}},
		{ID: "sha256:used-by-id", Tags: []string{"dkt/shop/api:ddd"}},
		{ID: "sha256:two-tags", Tags: []string{"dkt/shop/web:eee", "dkt/shop/web:fff"}},
		{ID: "sha256:foreign", Tags: []string{"dkt/shop/web:ggg", "nginx:1"}},
		{ID: "sha256:system", Tags: []string{"dockerity:latest"}},
		{ID: "sha256:untagged"},
		{ID: "sha256:partly-kept", Tags: []string{"dkt/shop/web:hhh", "dkt/shop/web:iii"}},
	}
	got := Select(images, Keep{
		Tags:  map[string]bool{"dkt/shop/web:bbb": true, "dkt/shop/web:iii": true},
		InUse: map[string]bool{"dkt/shop/web:ccc": true, "sha256:used-by-id": true},
	})
	want := []string{"dkt/shop/web:aaa", "dkt/shop/web:eee", "dkt/shop/web:fff"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSelectNothing(t *testing.T) {
	if got := Select(nil, Keep{}); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
