package gitx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// origin is a local upstream repo for workspace tests.
type origin struct {
	t    *testing.T
	dir  string
	repo *git.Repository
}

func newOrigin(t *testing.T) *origin {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	return &origin{t: t, dir: dir, repo: repo}
}

var sig = &object.Signature{Name: "t", Email: "t@example.com", When: time.Unix(1700000000, 0)}

// commit writes file=content and commits it on the current branch.
func (o *origin) commit(file, content string) plumbing.Hash {
	o.t.Helper()
	if err := os.WriteFile(filepath.Join(o.dir, file), []byte(content), 0o644); err != nil {
		o.t.Fatal(err)
	}
	wt, _ := o.repo.Worktree()
	if _, err := wt.Add(file); err != nil {
		o.t.Fatal(err)
	}
	h, err := wt.Commit("commit "+content, &git.CommitOptions{Author: sig})
	if err != nil {
		o.t.Fatal(err)
	}
	return h
}

func (o *origin) branch(name string, h plumbing.Hash) {
	o.t.Helper()
	if err := o.repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName(name), h)); err != nil {
		o.t.Fatal(err)
	}
}

func read(t *testing.T, dir, file string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return "<missing>"
	}
	return string(b)
}

func TestWorkspaceCheckout(t *testing.T) {
	o := newOrigin(t)
	first := o.commit("app.txt", "one")
	if _, err := o.repo.CreateTag("v1", first, nil); err != nil {
		t.Fatal(err)
	}
	second := o.commit("app.txt", "two")
	if _, err := o.repo.CreateTag("v2", second, &git.CreateTagOptions{Tagger: sig, Message: "release"}); err != nil {
		t.Fatal(err)
	}
	o.branch("main", second)
	o.branch("old", first)

	ws := &Workspace{Root: t.TempDir()}
	ctx := context.Background()
	cases := map[string]struct {
		hash    plumbing.Hash
		content string
	}{
		"main":             {second, "two"},
		"old":              {first, "one"},
		"v1":               {first, "one"},
		"v2":               {second, "two"},
		first.String():     {first, "one"},
		first.String()[:7]: {first, "one"},
	}
	for ref, want := range cases {
		co, err := ws.checkout(ctx, "shop", o.dir, nil, ref)
		if err != nil {
			t.Fatalf("%s: %v", ref, err)
		}
		if co.CommitSHA != want.hash.String() || read(t, co.Dir, "app.txt") != want.content {
			t.Fatalf("%s: got %s %q", ref, co.CommitSHA, read(t, co.Dir, "app.txt"))
		}
	}

	if _, err := ws.checkout(ctx, "shop", o.dir, nil, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown ref: %v", err)
	}
	if _, err := ws.checkout(ctx, "../escape", o.dir, nil, "main"); err == nil {
		t.Fatal("invalid app name accepted")
	}
}

func TestWorkspaceIsCleanAndFetchesNewCommits(t *testing.T) {
	o := newOrigin(t)
	o.branch("main", o.commit("app.txt", "one"))
	ws := &Workspace{Root: t.TempDir()}
	ctx := context.Background()

	co, err := ws.checkout(ctx, "shop", o.dir, nil, "main")
	if err != nil {
		t.Fatal(err)
	}
	// Leftovers from a previous build must not survive the next checkout.
	_ = os.WriteFile(filepath.Join(co.Dir, "app.txt"), []byte("dirty"), 0o644)
	_ = os.MkdirAll(filepath.Join(co.Dir, "node_modules"), 0o755)
	_ = os.WriteFile(filepath.Join(co.Dir, "node_modules", "x"), []byte("x"), 0o644)

	o.branch("main", o.commit("app.txt", "three"))
	co, err = ws.checkout(ctx, "shop", o.dir, nil, "main")
	if err != nil {
		t.Fatal(err)
	}
	if read(t, co.Dir, "app.txt") != "three" {
		t.Fatalf("new commit not fetched or file not restored: %q", read(t, co.Dir, "app.txt"))
	}
	if _, err := os.Stat(filepath.Join(co.Dir, "node_modules")); !os.IsNotExist(err) {
		t.Fatal("untracked directory survived the checkout")
	}
}
