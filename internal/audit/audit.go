// Package audit defines how actions are recorded in the audit log.
package audit

import (
	"context"
	"strings"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Recorder appends events to the audit log.
type Recorder interface {
	Record(ctx context.Context, e domain.AuditEvent) error
}

// Redacted replaces metadata values whose key suggests a secret.
const Redacted = "[redacted]"

// sensitiveSuffixes match keys like "password", "new_password", "secret",
// "secret_value", "api_key" or "token". Keys such as "token_id" or
// "secret_name" are kept.
var sensitiveSuffixes = []string{"password", "secret", "value", "token", "key", "hash"}

// Sanitize returns a copy of meta with values of sensitive-looking keys
// redacted. It is a safety net: callers should not pass secrets at all.
func Sanitize(meta map[string]string) map[string]string {
	if len(meta) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(meta))
	for k, v := range meta {
		if isSensitive(k) {
			v = Redacted
		}
		out[k] = v
	}
	return out
}

func isSensitive(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveSuffixes {
		if strings.HasSuffix(k, s) {
			return true
		}
	}
	return false
}
