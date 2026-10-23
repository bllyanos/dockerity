package gitx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// Checkout is a working tree at a resolved commit.
type Checkout struct {
	Dir       string
	CommitSHA string
}

// Workspace keeps one clone per app under Root and checks out commits in it.
// Operations on the same app are serialized.
type Workspace struct {
	Root   string
	Client *Client

	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// Checkout fetches the repo and checks out ref (branch, tag, or full or short
// commit SHA) as a clean, detached working tree.
func (w *Workspace) Checkout(ctx context.Context, app, repoURL string, privatePEM []byte, ref string) (Checkout, error) {
	auth, err := w.Client.sshAuth(ctx, repoURL, privatePEM)
	if err != nil {
		return Checkout{}, err
	}
	return w.checkout(ctx, app, repoURL, auth, ref)
}

func (w *Workspace) checkout(ctx context.Context, app, repoURL string, auth transport.AuthMethod, ref string) (Checkout, error) {
	if !domain.IsDNSLabel(app) {
		return Checkout{}, fmt.Errorf("invalid app name %q", app)
	}
	dir := filepath.Join(w.Root, app)
	unlock := w.lock(dir)
	defer unlock()

	repo, err := w.sync(ctx, dir, repoURL, auth)
	if err != nil {
		return Checkout{}, err
	}
	hash, err := resolve(repo, ref)
	if err != nil {
		return Checkout{}, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return Checkout{}, err
	}
	if err := wt.Checkout(&git.CheckoutOptions{Hash: hash, Force: true}); err != nil {
		return Checkout{}, fmt.Errorf("checkout %s: %w", hash, err)
	}
	if err := wt.Clean(&git.CleanOptions{Dir: true}); err != nil {
		return Checkout{}, fmt.Errorf("clean worktree: %w", err)
	}
	return Checkout{Dir: dir, CommitSHA: hash.String()}, nil
}

// sync clones the repo into dir, or fetches if it is already there.
func (w *Workspace) sync(ctx context.Context, dir, repoURL string, auth transport.AuthMethod) (*git.Repository, error) {
	repo, err := git.PlainOpen(dir)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
			return nil, err
		}
		repo, err = git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
			URL: repoURL, Auth: auth, NoCheckout: true, Tags: git.AllTags,
		})
		if err != nil {
			_ = os.RemoveAll(dir)
			return nil, explain(err)
		}
		return repo, nil
	}
	if err != nil {
		return nil, err
	}
	if err := setOriginURL(repo, repoURL); err != nil {
		return nil, err
	}
	err = repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*", "+refs/tags/*:refs/tags/*"},
		Auth:       auth,
		Force:      true,
		Prune:      true,
		Tags:       git.AllTags,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, explain(err)
	}
	return repo, nil
}

// setOriginURL keeps the clone pointed at the app's current repo URL.
func setOriginURL(repo *git.Repository, url string) error {
	cfg, err := repo.Config()
	if err != nil {
		return err
	}
	origin, ok := cfg.Remotes["origin"]
	if !ok || len(origin.URLs) == 1 && origin.URLs[0] == url {
		return nil
	}
	origin.URLs = []string{url}
	return repo.SetConfig(cfg)
}

var hexPattern = regexp.MustCompile(`^[0-9a-f]{4,40}$`)

// resolve maps a branch, tag or commit SHA to a commit, in that order.
func resolve(repo *git.Repository, ref string) (plumbing.Hash, error) {
	if ref == "" {
		return plumbing.ZeroHash, domain.Invalid("ref", "is required")
	}
	if r, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", ref), true); err == nil {
		return r.Hash(), nil
	}
	if r, err := repo.Reference(plumbing.NewTagReferenceName(ref), true); err == nil {
		if tag, err := repo.TagObject(r.Hash()); err == nil {
			c, err := tag.Commit()
			if err != nil {
				return plumbing.ZeroHash, fmt.Errorf("tag %s does not point at a commit: %w", ref, err)
			}
			return c.Hash, nil
		}
		return r.Hash(), nil
	}
	if hexPattern.MatchString(ref) {
		if h, err := repo.ResolveRevision(plumbing.Revision(ref)); err == nil {
			return *h, nil
		}
	}
	return plumbing.ZeroHash, domain.NotFoundf("ref %q is not a branch, tag or commit in the repo", ref)
}

func (w *Workspace) lock(dir string) func() {
	w.mu.Lock()
	if w.locks == nil {
		w.locks = map[string]*sync.Mutex{}
	}
	l, ok := w.locks[dir]
	if !ok {
		l = &sync.Mutex{}
		w.locks[dir] = l
	}
	w.mu.Unlock()
	l.Lock()
	return l.Unlock
}
