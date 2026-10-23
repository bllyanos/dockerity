package domain

import "regexp"

// MaxSecretKeyLength bounds secret names.
const MaxSecretKeyLength = 128

var secretKeyPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// ValidateSecretKey checks that name can be used as an environment variable.
func ValidateSecretKey(name string) error {
	if len(name) > MaxSecretKeyLength || !secretKeyPattern.MatchString(name) {
		return Invalid("key", "%q must be uppercase letters, digits and '_', not starting with a digit, at most %d characters", name, MaxSecretKeyLength)
	}
	return nil
}
