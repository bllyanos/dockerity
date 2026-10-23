// Package routing generates the Traefik labels that route an environment's
// HTTP services (docs/design.md section 5). It is pure.
package routing

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bllyanos/dockerity/internal/domain"
)

// Route is one HTTP service exposed through Traefik.
type Route struct {
	App     string
	Env     string
	Service string
	Domain  string
	Path    string // "/" or empty routes the whole domain
	Port    int
}

// Options configure how routes are exposed.
type Options struct {
	// Entrypoints are Traefik entrypoint names, e.g. "web".
	Entrypoints []string
}

// DefaultOptions route over the plain HTTP entrypoint of the bootstrap compose file.
var DefaultOptions = Options{Entrypoints: []string{"web"}}

// RouterName is shared by every container of a service in an environment, so
// blue-green containers join the same Traefik service.
func RouterName(app, env, service string) string {
	return fmt.Sprintf("dkt-%s-%s-%s", app, env, service)
}

// Rule is the Traefik router rule for r.
func (r Route) Rule() string {
	rule := fmt.Sprintf("Host(`%s`)", r.Domain)
	if p := strings.TrimSuffix(r.Path, "/"); p != "" {
		rule += fmt.Sprintf(" && PathPrefix(`%s`)", p)
	}
	return rule
}

// Labels returns the Traefik labels for r. Routes without a domain get no labels.
func Labels(r Route, opts Options) map[string]string {
	if r.Domain == "" {
		return map[string]string{}
	}
	name := RouterName(r.App, r.Env, r.Service)
	router := "traefik.http.routers." + name
	rule := r.Rule()
	return map[string]string{
		"traefik.enable":         "true",
		"traefik.docker.network": domain.NetworkName(r.App, r.Env),
		router + ".rule":         rule,
		router + ".entrypoints":  strings.Join(opts.Entrypoints, ","),
		router + ".service":      name,
		// A longer rule (a deeper path) wins over the bare host rule of another service.
		router + ".priority": strconv.Itoa(len(rule)),
		"traefik.http.services." + name + ".loadbalancer.server.port": strconv.Itoa(r.Port),
	}
}
