package gitx

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"golang.org/x/crypto/ssh"
)

type memHostKeys map[string]KnownHost

func (m memHostKeys) GetHostKey(_ context.Context, host string) (KnownHost, error) {
	k, ok := m[host]
	if !ok {
		return KnownHost{}, domain.ErrNotFound
	}
	return k, nil
}

func (m memHostKeys) AddHostKey(_ context.Context, k KnownHost) error {
	if _, ok := m[k.Host]; !ok {
		m[k.Host] = k
	}
	return nil
}

func hostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestTrustOnFirstUse(t *testing.T) {
	store := memHostKeys{}
	cb := TrustOnFirstUse(context.Background(), store, time.Now)
	addr := &net.TCPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 22}
	first, other := hostKey(t), hostKey(t)

	if err := cb("github.com:22", addr, first); err != nil {
		t.Fatalf("first use: %v", err)
	}
	if store["github.com"].Fingerprint != ssh.FingerprintSHA256(first) {
		t.Fatalf("not pinned under normalized host: %+v", store)
	}
	if err := cb("github.com:22", addr, first); err != nil {
		t.Fatalf("same key: %v", err)
	}
	var mismatch *HostKeyMismatchError
	if err := cb("github.com:22", addr, other); !errors.As(err, &mismatch) {
		t.Fatalf("changed key: %v", err)
	}
	if err := cb("gitlab.example.com:2222", addr, other); err != nil {
		t.Fatalf("other host: %v", err)
	}
	if _, ok := store["[gitlab.example.com]:2222"]; !ok {
		t.Fatalf("non-default port not normalized: %+v", store)
	}
}

func TestListBranchesLocal(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	wt, _ := repo.Worktree()
	commit, err := wt.Commit("init", &git.CommitOptions{AllowEmptyCommits: true, Author: &object.Signature{Name: "t", Email: "t@example.com", When: time.Now()}})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range []string{"main", "release/1.0"} {
		if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName(b), commit)); err != nil {
			t.Fatal(err)
		}
	}
	_ = repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewTagReferenceName("v1"), commit))

	got, err := listBranches(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(got, "main") || !slices.Contains(got, "release/1.0") || slices.Contains(got, "v1") {
		t.Fatalf("branches %v", got)
	}
}

func TestListBranchesMissingRepo(t *testing.T) {
	if _, err := listBranches(context.Background(), t.TempDir()+"/missing", nil); err == nil {
		t.Fatal("expected an error")
	}
}
