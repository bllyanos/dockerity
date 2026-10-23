package build

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/bllyanos/dockerity/internal/spec"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func tarNames(t *testing.T, r io.ReadCloser) map[string]string {
	t.Helper()
	defer r.Close()
	out := map[string]string{}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(tr)
		if hdr.Typeflag == tar.TypeSymlink {
			body = []byte("-> " + hdr.Linkname)
		}
		out[hdr.Name] = string(body)
	}
}

func TestTarHonorsDockerignore(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Dockerfile":          "FROM scratch",
		".dockerignore":       "node_modules\n*.log\nDockerfile\nsecrets/\n!secrets/keep.txt\n",
		"main.go":             "package main",
		"debug.log":           "noise",
		"node_modules/a/b.js": "x",
		"secrets/key":         "hidden",
		"secrets/keep.txt":    "kept",
	})
	_ = os.Symlink("../../etc/passwd", filepath.Join(dir, "link"))

	r, err := Tar(dir, "Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	got := tarNames(t, r)
	for _, want := range []string{"Dockerfile", ".dockerignore", "main.go", "secrets/keep.txt"} {
		if _, ok := got[want]; !ok {
			t.Errorf("%s missing from context: %v", want, keys(got))
		}
	}
	for _, unwanted := range []string{"debug.log", "node_modules/", "node_modules/a/b.js", "secrets/key"} {
		if _, ok := got[unwanted]; ok {
			t.Errorf("%s should be ignored", unwanted)
		}
	}
	if got["link"] != "-> ../../etc/passwd" {
		t.Errorf("symlink must be archived as a link, got %q", got["link"])
	}
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func TestContextDir(t *testing.T) {
	repo := t.TempDir()
	writeFiles(t, repo, map[string]string{"web/Dockerfile": "FROM x", "file.txt": "x"})
	outside := t.TempDir()
	_ = os.Symlink(outside, filepath.Join(repo, "escape"))

	if dir, err := ContextDir(repo, "web"); err != nil || filepath.Base(dir) != "web" {
		t.Fatalf("web: %q %v", dir, err)
	}
	if _, err := ContextDir(repo, "."); err != nil {
		t.Fatalf("root context: %v", err)
	}
	for _, bad := range []string{"", "/web", "../x", "escape", "file.txt", "missing"} {
		if _, err := ContextDir(repo, bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	ctx, _ := ContextDir(repo, "web")
	if err := CheckDockerfile(ctx, "Dockerfile"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"../Dockerfile", "/Dockerfile", "Missing", "."} {
		if err := CheckDockerfile(ctx, bad); err == nil {
			t.Errorf("dockerfile %q accepted", bad)
		}
	}
}

func TestPlan(t *testing.T) {
	s, err := spec.Load([]byte(`
services:
  db:
    kind: stateful
    image: mysql:8.4
  api:
    kind: deployment
    strategy: recreate
    build:
      context: ./api
      target: prod
      args:
        GO_VERSION: "1.27"
  worker:
    kind: deployment
    strategy: recreate
    build: .
`))
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan("shop", s, "abc123")
	if len(plan) != 2 || plan[0].Service != "api" || plan[1].Service != "worker" {
		t.Fatalf("plan %+v", plan)
	}
	api := plan[0]
	if api.Context != "./api" || api.Dockerfile != "Dockerfile" || api.Target != "prod" ||
		api.Args["GO_VERSION"] != "1.27" || api.ImageTag != "dkt/shop/api:abc123" || api.Labels["dockerity.commit"] != "abc123" {
		t.Fatalf("api build %+v", api)
	}
}
