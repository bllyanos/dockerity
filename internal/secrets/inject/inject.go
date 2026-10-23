// Package inject turns a service's secret references and a decrypted secret
// set into what a container needs: environment variables and tmpfs files.
// It has no Docker dependency; the deploy engine applies the plan.
package inject

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// Ref is one secret a service asks for. With File empty the secret becomes an
// environment variable named Name; otherwise it is written to File.
type Ref struct {
	Name string
	File string
}

// File is a secret file to place on a tmpfs mount inside the container.
type File struct {
	Path    string
	Content []byte
	Mode    uint32
}

// FileMode is the permission of injected secret files.
const FileMode = 0o400

// Plan is what to inject into one container.
type Plan struct {
	// Env holds "NAME=value" entries, sorted by name.
	Env   []string
	Files []File
}

// String reports counts only, so a Plan never leaks values into logs.
func (p Plan) String() string {
	return fmt.Sprintf("inject.Plan{env: %d, files: %d}", len(p.Env), len(p.Files))
}

// GoString hides values from %#v as well.
func (p Plan) GoString() string { return p.String() }

// MissingError lists referenced secrets that are not in the secret set.
type MissingError struct {
	Names []string
}

func (e *MissingError) Error() string {
	return "missing secrets: " + strings.Join(e.Names, ", ")
}

// Build resolves refs against values. Every missing name is reported at once.
// Errors name secrets and paths but never include values.
func Build(refs []Ref, values map[string]string) (Plan, error) {
	var (
		plan    Plan
		missing []string
		envSeen = map[string]bool{}
		files   = map[string]string{}
	)
	for _, r := range refs {
		v, ok := values[r.Name]
		if !ok {
			if !slices.Contains(missing, r.Name) {
				missing = append(missing, r.Name)
			}
			continue
		}
		if r.File == "" {
			if !envSeen[r.Name] {
				envSeen[r.Name] = true
				plan.Env = append(plan.Env, r.Name+"="+v)
			}
			continue
		}
		if err := validateFilePath(r.File); err != nil {
			return Plan{}, err
		}
		if prev, dup := files[r.File]; dup {
			if prev == r.Name {
				continue
			}
			return Plan{}, fmt.Errorf("secret file %s is used by both %s and %s", r.File, prev, r.Name)
		}
		files[r.File] = r.Name
		plan.Files = append(plan.Files, File{Path: r.File, Content: []byte(v), Mode: FileMode})
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return Plan{}, &MissingError{Names: missing}
	}
	slices.Sort(plan.Env)
	slices.SortFunc(plan.Files, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	return plan, nil
}

func validateFilePath(p string) error {
	if !path.IsAbs(p) || path.Clean(p) != p || p == "/" {
		return fmt.Errorf("secret file path %q must be an absolute, clean path", p)
	}
	return nil
}
