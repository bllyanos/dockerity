package apps

import (
	"context"
	"errors"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
)

type fakeGit struct {
	branches []string
	err      error
	gotKey   []byte
}

func (f *fakeGit) ListBranches(_ context.Context, _ string, key []byte) ([]string, error) {
	f.gotKey = key
	return f.branches, f.err
}

type fixedKeys struct{}

func (fixedKeys) PrivateKey(context.Context, int64) ([]byte, error) { return []byte("pem"), nil }

func TestConnectionTester(t *testing.T) {
	s := newService()
	ctx := context.Background()
	a, _ := s.Create(ctx, CreateInput{Name: "shop", RepoURL: "git@github.com:org/shop.git"})

	git := &fakeGit{branches: []string{"develop", "main"}}
	tester := &ConnectionTester{Apps: s, Keys: fixedKeys{}, Git: git}
	res, err := tester.Test(ctx, a.ID)
	if err != nil || !res.OK || !res.DefaultBranchExists || res.BranchCount != 2 || string(git.gotKey) != "pem" {
		t.Fatalf("ok case: %+v %v", res, err)
	}

	git.branches = []string{"develop"}
	if res, _ = tester.Test(ctx, a.ID); res.OK || res.DefaultBranchExists {
		t.Fatalf("missing branch: %+v", res)
	}

	git.err = errors.New("the deploy key was rejected")
	if res, err = tester.Test(ctx, a.ID); err != nil || res.OK || res.Message != "the deploy key was rejected" {
		t.Fatalf("git failure: %+v %v", res, err)
	}

	if _, err := tester.Test(ctx, 99); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing app: %v", err)
	}
}
