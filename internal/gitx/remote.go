package gitx

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/deploykey"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Client talks to git remotes with per-app deploy keys.
type Client struct {
	HostKeys HostKeyStore
	Now      func() time.Time
}

// ListBranches returns the remote's branch names, authenticating with privatePEM.
func (c *Client) ListBranches(ctx context.Context, repoURL string, privatePEM []byte) ([]string, error) {
	auth, err := c.sshAuth(ctx, repoURL, privatePEM)
	if err != nil {
		return nil, err
	}
	return listBranches(ctx, repoURL, auth)
}

func (c *Client) sshAuth(ctx context.Context, repoURL string, privatePEM []byte) (transport.AuthMethod, error) {
	signer, err := deploykey.Signer(privatePEM)
	if err != nil {
		return nil, err
	}
	ep, err := transport.NewEndpoint(repoURL)
	if err != nil {
		return nil, fmt.Errorf("repo URL: %w", err)
	}
	user := ep.User
	if user == "" {
		user = "git"
	}
	return &gitssh.PublicKeys{
		User:   user,
		Signer: signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: TrustOnFirstUse(ctx, c.HostKeys, c.Now),
		},
	}, nil
}

// listBranches lists refs/heads on a remote. auth may be nil for local remotes.
func listBranches(ctx context.Context, repoURL string, auth transport.AuthMethod) ([]string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{Name: "origin", URLs: []string{repoURL}})
	refs, err := remote.ListContext(ctx, &git.ListOptions{Auth: auth})
	if err != nil {
		return nil, explain(err)
	}
	var branches []string
	for _, r := range refs {
		if r.Name().IsBranch() {
			branches = append(branches, r.Name().Short())
		}
	}
	sort.Strings(branches)
	return branches, nil
}

// explain turns common transport failures into actionable messages.
func explain(err error) error {
	var mismatch *HostKeyMismatchError
	msg := err.Error()
	switch {
	case errors.As(err, &mismatch):
		return mismatch
	case errors.Is(err, transport.ErrRepositoryNotFound):
		return fmt.Errorf("repository not found, or the deploy key has no access to it: %w", err)
	case errors.Is(err, transport.ErrAuthenticationRequired), errors.Is(err, transport.ErrAuthorizationFailed),
		strings.Contains(msg, "unable to authenticate"):
		return fmt.Errorf("the deploy key was rejected; add the public key to the repo as a read-only deploy key: %w", err)
	case errors.Is(err, transport.ErrEmptyRemoteRepository):
		return fmt.Errorf("the repository is empty: %w", err)
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "connection refused"), strings.Contains(msg, "i/o timeout"):
		return fmt.Errorf("cannot reach the git host: %w", err)
	default:
		return err
	}
}
