package api

import "net/http"

// routes registers API handlers and remembers which patterns are public.
// Every other pattern, including unknown paths, requires authentication.
type routes struct {
	mux      *http.ServeMux
	public   map[string]bool
	patterns []string
}

func newRoutes() *routes {
	return &routes{mux: http.NewServeMux(), public: map[string]bool{}}
}

// handle registers a route that requires an authenticated principal.
func (r *routes) handle(pattern string, h http.HandlerFunc) {
	r.mux.HandleFunc(pattern, h)
	r.patterns = append(r.patterns, pattern)
}

// handlePublic registers a route reachable without authentication.
func (r *routes) handlePublic(pattern string, h http.HandlerFunc) {
	r.handle(pattern, h)
	r.public[pattern] = true
}

// isPublic reports whether req resolves to a public route.
func (r *routes) isPublic(req *http.Request) bool {
	_, pattern := r.mux.Handler(req)
	return r.public[pattern]
}
