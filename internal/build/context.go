package build

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
)

// ContextDir resolves a service's build context inside repoDir. It rejects
// contexts or symlinks that lead outside the repo.
func ContextDir(repoDir, context string) (string, error) {
	if context == "" || path.IsAbs(context) {
		return "", fmt.Errorf("build context %q must be a relative path inside the repo", context)
	}
	root, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		return "", err
	}
	dir, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(context)))
	if err != nil {
		return "", fmt.Errorf("build context %q: %w", context, err)
	}
	if rel, err := filepath.Rel(root, dir); err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("build context %q is outside the repo", context)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("build context %q is not a directory", context)
	}
	return dir, nil
}

// CheckDockerfile validates that dockerfile is a relative path inside the context.
func CheckDockerfile(contextDir, dockerfile string) error {
	clean := path.Clean(dockerfile)
	if path.IsAbs(dockerfile) || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("dockerfile %q must be inside the build context", dockerfile)
	}
	info, err := os.Stat(filepath.Join(contextDir, filepath.FromSlash(clean)))
	if err != nil {
		return fmt.Errorf("dockerfile %q: %w", dockerfile, err)
	}
	if info.IsDir() {
		return fmt.Errorf("dockerfile %q is a directory", dockerfile)
	}
	return nil
}

// Tar streams contextDir as a tar archive, honoring .dockerignore. The
// Dockerfile and .dockerignore are always included, like the Docker CLI does.
// Symlinks are archived as links, never followed.
func Tar(contextDir, dockerfile string) (io.ReadCloser, error) {
	pm, err := ignoreMatcher(contextDir)
	if err != nil {
		return nil, err
	}
	keep := map[string]bool{path.Clean(dockerfile): true, ".dockerignore": true}

	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		err := filepath.WalkDir(contextDir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(contextDir, p)
			if err != nil || rel == "." {
				return err
			}
			rel = filepath.ToSlash(rel)
			if !keep[rel] && pm != nil {
				excluded, err := pm.MatchesOrParentMatches(rel)
				if err != nil {
					return err
				}
				if excluded {
					if d.IsDir() && !pm.Exclusions() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			return addEntry(tw, p, rel, d)
		})
		if err == nil {
			err = tw.Close()
		}
		pw.CloseWithError(err)
	}()
	return pr, nil
}

func ignoreMatcher(contextDir string) (*patternmatcher.PatternMatcher, error) {
	f, err := os.Open(filepath.Join(contextDir, ".dockerignore"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	patterns, err := ignorefile.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf(".dockerignore: %w", err)
	}
	return patternmatcher.New(patterns)
}

func addEntry(tw *tar.Writer, p, rel string, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	link := ""
	if info.Mode()&fs.ModeSymlink != 0 {
		if link, err = os.Readlink(p); err != nil {
			return err
		}
	}
	hdr, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return err
	}
	hdr.Name = rel
	if info.IsDir() {
		hdr.Name += "/"
	}
	// The host owner is meaningless inside the image; files belong to root like with the Docker CLI.
	hdr.Uid, hdr.Gid, hdr.Uname, hdr.Gname = 0, 0, "", ""
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}
