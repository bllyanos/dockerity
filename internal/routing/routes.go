// Package routing generates Traefik dynamic configuration for an
// environment's HTTP services (docs/design.md section 5). Dockerity writes one
// file per environment for Traefik's file provider and owns its server lists,
// so a container can leave routing while it still drains requests.
// The generator is pure; Files writes the result.
package routing

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Options configure how routes are exposed.
type Options struct {
	// Entrypoints are Traefik entrypoint names, e.g. "web".
	Entrypoints []string
}

// DefaultOptions route over the plain HTTP entrypoint of the bootstrap compose file.
var DefaultOptions = Options{Entrypoints: []string{"web"}}

// RouterName names the Traefik router and service of an environment service.
func RouterName(app, env, service string) string {
	return fmt.Sprintf("dkt-%s-%s-%s", app, env, service)
}

// Service is one routed service of an environment.
type Service struct {
	Name string
	Path string // "/" or empty routes the whole domain
	Port int
	// Containers are the container names that receive traffic, reachable by
	// Traefik over the environment network.
	Containers []string
}

// Env is the routing of one environment.
type Env struct {
	App      string
	Env      string
	Domain   string
	Services []Service
}

// Rule is the Traefik router rule for a service on domain.
func Rule(domain, path string) string {
	rule := fmt.Sprintf("Host(`%s`)", domain)
	if p := strings.TrimSuffix(path, "/"); p != "" {
		rule += fmt.Sprintf(" && PathPrefix(`%s`)", p)
	}
	return rule
}

type dynamicConfig struct {
	HTTP httpConfig `yaml:"http"`
}

type httpConfig struct {
	Routers  map[string]router  `yaml:"routers,omitempty"`
	Services map[string]service `yaml:"services,omitempty"`
}

type router struct {
	Rule        string   `yaml:"rule"`
	EntryPoints []string `yaml:"entryPoints"`
	Service     string   `yaml:"service"`
	Priority    int      `yaml:"priority"`
}

type service struct {
	LoadBalancer loadBalancer `yaml:"loadBalancer"`
}

type loadBalancer struct {
	Servers []server `yaml:"servers"`
}

type server struct {
	URL string `yaml:"url"`
}

// Config renders the Traefik dynamic configuration of an environment. An
// environment without a domain, and services without containers, are not
// routed.
func Config(e Env, opts Options) ([]byte, error) {
	cfg := dynamicConfig{HTTP: httpConfig{Routers: map[string]router{}, Services: map[string]service{}}}
	if e.Domain != "" {
		for _, s := range e.Services {
			if len(s.Containers) == 0 {
				continue
			}
			name := RouterName(e.App, e.Env, s.Name)
			rule := Rule(e.Domain, s.Path)
			cfg.HTTP.Routers[name] = router{
				Rule: rule, EntryPoints: opts.Entrypoints, Service: name,
				// A longer rule (a deeper path) wins over a bare host rule.
				Priority: len(rule),
			}
			containers := slices.Sorted(slices.Values(s.Containers))
			servers := make([]server, 0, len(containers))
			for _, c := range containers {
				servers = append(servers, server{URL: "http://" + c + ":" + strconv.Itoa(s.Port)})
			}
			cfg.HTTP.Services[name] = service{LoadBalancer: loadBalancer{Servers: servers}}
		}
	}
	return yaml.Marshal(cfg)
}
