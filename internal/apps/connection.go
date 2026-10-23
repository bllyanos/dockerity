package apps

import (
	"context"
	"slices"

	"github.com/bllyanos/dockerity/internal/domain"
)

// GitRemote lists branches of a repo using a deploy key.
type GitRemote interface {
	ListBranches(ctx context.Context, repoURL string, privatePEM []byte) ([]string, error)
}

// PrivateKeys reads an app's private deploy key.
type PrivateKeys interface {
	PrivateKey(ctx context.Context, appID int64) ([]byte, error)
}

// ConnectionResult is the outcome of a connection test. Git failures are
// reported in Message rather than as errors, since they are the answer.
type ConnectionResult struct {
	OK                  bool
	DefaultBranchExists bool
	BranchCount         int
	Message             string
}

// ConnectionTester checks that Dockerity can read an app's repo.
type ConnectionTester struct {
	Apps interface {
		Get(ctx context.Context, id int64) (domain.App, error)
	}
	Keys PrivateKeys
	Git  GitRemote
}

// Test lists the remote's branches with the app's deploy key.
func (t *ConnectionTester) Test(ctx context.Context, appID int64) (ConnectionResult, error) {
	app, err := t.Apps.Get(ctx, appID)
	if err != nil {
		return ConnectionResult{}, err
	}
	key, err := t.Keys.PrivateKey(ctx, app.ID)
	if err != nil {
		return ConnectionResult{}, err
	}
	branches, err := t.Git.ListBranches(ctx, app.RepoURL, key)
	if err != nil {
		return ConnectionResult{Message: err.Error()}, nil
	}
	res := ConnectionResult{
		OK:                  true,
		BranchCount:         len(branches),
		DefaultBranchExists: slices.Contains(branches, app.DefaultBranch),
		Message:             "connected",
	}
	if !res.DefaultBranchExists {
		res.OK = false
		res.Message = "connected, but branch " + app.DefaultBranch + " does not exist"
	}
	return res, nil
}
