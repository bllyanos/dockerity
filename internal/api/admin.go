package api

import (
	"net/http"

	"github.com/bllyanos/dockerity/internal/auth"
)

// adminOnly rejects authenticated callers that are not admins.
func adminOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFrom(r.Context())
		if !ok || !p.User.IsAdmin {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		h(w, r)
	}
}
