package auth

import (
	"context"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Principal is the authenticated caller of a request.
type Principal struct {
	User domain.User
	// APITokenID is set when the request used an API token instead of a session.
	APITokenID int64
}

type principalKey struct{}

// WithPrincipal returns ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom returns the caller stored in ctx, if any.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
