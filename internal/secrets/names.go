package secrets

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Limits keep secret sets small enough to load per deploy.
const (
	MaxKeyLength   = 128
	MaxValueLength = 64 * 1024
	MaxKeys        = 500
)

var keyPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// ValidateKey checks that name can be used as an environment variable.
func ValidateKey(name string) error {
	if len(name) > MaxKeyLength || !keyPattern.MatchString(name) {
		return domain.Invalid("key", "%q must be uppercase letters, digits and '_', not starting with a digit, at most %d characters", name, MaxKeyLength)
	}
	return nil
}

// Changes is an edit of a secret set: keys to set and keys to remove.
type Changes struct {
	Set   map[string]string
	Unset []string
}

// Apply returns a new set with c applied to current. current is not modified.
// Error messages name keys but never include values.
func (c Changes) Apply(current map[string]string) (map[string]string, error) {
	if len(c.Set) == 0 && len(c.Unset) == 0 {
		return nil, domain.Invalid("changes", "nothing to change")
	}
	next := make(map[string]string, len(current)+len(c.Set))
	for k, v := range current {
		next[k] = v
	}
	for _, k := range c.Unset {
		if err := ValidateKey(k); err != nil {
			return nil, err
		}
		if _, set := c.Set[k]; set {
			return nil, domain.Invalid("key", "%s is both set and unset", k)
		}
		if _, ok := next[k]; !ok {
			return nil, domain.Invalid("key", "%s is not set", k)
		}
		delete(next, k)
	}
	for k, v := range c.Set {
		if err := ValidateKey(k); err != nil {
			return nil, err
		}
		if len(v) > MaxValueLength {
			return nil, domain.Invalid("value", "value of %s is longer than %d bytes", k, MaxValueLength)
		}
		next[k] = v
	}
	if len(next) > MaxKeys {
		return nil, domain.Invalid("changes", "a secret set can hold at most %d keys", MaxKeys)
	}
	return next, nil
}

// SortedKeys returns the key names of m in order.
func SortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// AuditMetadata summarizes the changed key names for the audit log, without values.
func (c Changes) AuditMetadata() map[string]string {
	set := make([]string, 0, len(c.Set))
	for k := range c.Set {
		set = append(set, k)
	}
	slices.Sort(set)
	unset := slices.Sorted(slices.Values(c.Unset))
	return map[string]string{"set_keys": fmt.Sprint(set), "unset_keys": fmt.Sprint(unset)}
}
