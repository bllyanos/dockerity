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

// Entrypoint names of the bootstrap compose file.
const (
	EntrypointHTTP  = "web"
	EntrypointHTTPS = "websecure"
)

// TLS configures HTTPS routing.
type TLS struct {
	// CertResolver, when set, lets that Traefik resolver obtain certificates;
	// otherwise Traefik serves its configured default certificate.
	CertResolver string
	// WildcardDomain is the base domain whose wildcard certificate
	// (*.WildcardDomain) covers hosts directly below it.
	WildcardDomain string
}

// Options configure how routes are exposed.
type Options struct {
	// Entrypoints are Traefik entrypoint names used without TLS.
	Entrypoints []string
	// TLS, when set, serves routes on websecure and redirects HTTP to HTTPS.
	TLS *TLS
}

// DefaultOptions route over the plain HTTP entrypoint of the bootstrap compose file.
var DefaultOptions = Options{Entrypoints: []string{EntrypointHTTP}}

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
	Routers     map[string]router     `yaml:"routers,omitempty"`
	Services    map[string]service    `yaml:"services,omitempty"`
	Middlewares map[string]middleware `yaml:"middlewares,omitempty"`
}

type router struct {
	Rule        string     `yaml:"rule"`
	EntryPoints []string   `yaml:"entryPoints"`
	Service     string     `yaml:"service"`
	Priority    int        `yaml:"priority"`
	Middlewares []string   `yaml:"middlewares,omitempty"`
	TLS         *routerTLS `yaml:"tls,omitempty"`
}

type routerTLS struct {
	CertResolver string      `yaml:"certResolver,omitempty"`
	Domains      []tlsDomain `yaml:"domains,omitempty"`
}

type tlsDomain struct {
	Main string `yaml:"main"`
}

type middleware struct {
	RedirectScheme *redirectScheme `yaml:"redirectScheme,omitempty"`
}

type redirectScheme struct {
	Scheme    string `yaml:"scheme"`
	Permanent bool   `yaml:"permanent"`
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
	cfg := dynamicConfig{HTTP: httpConfig{Routers: map[string]router{}, Services: map[string]service{}, Middlewares: map[string]middleware{}}}
	if e.Domain != "" {
		redirect := fmt.Sprintf("dkt-%s-%s-https", e.App, e.Env)
		for _, s := range e.Services {
			if len(s.Containers) == 0 {
				continue
			}
			name := RouterName(e.App, e.Env, s.Name)
			rule := Rule(e.Domain, s.Path)
			// A longer rule (a deeper path) wins over a bare host rule.
			r := router{Rule: rule, EntryPoints: opts.Entrypoints, Service: name, Priority: len(rule)}
			if opts.TLS != nil {
				r.EntryPoints = []string{EntrypointHTTPS}
				r.TLS = newRouterTLS(opts.TLS, e.Domain)
				cfg.HTTP.Routers[name+"-http"] = router{
					Rule: rule, EntryPoints: []string{EntrypointHTTP}, Service: name, Priority: len(rule), Middlewares: []string{redirect},
				}
				cfg.HTTP.Middlewares[redirect] = middleware{RedirectScheme: &redirectScheme{Scheme: "https", Permanent: true}}
			}
			cfg.HTTP.Routers[name] = r
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

// routerTLS asks for the wildcard certificate when host sits directly below
// the wildcard domain, and lets the resolver (if any) obtain it.
func newRouterTLS(t *TLS, host string) *routerTLS {
	out := &routerTLS{CertResolver: t.CertResolver}
	if t.CertResolver != "" && t.WildcardDomain != "" && strings.HasSuffix(host, "."+t.WildcardDomain) &&
		!strings.Contains(strings.TrimSuffix(host, "."+t.WildcardDomain), ".") {
		out.Domains = []tlsDomain{{Main: "*." + t.WildcardDomain}}
	}
	return out
}

// RoutedContainers parses a dynamic config written by Config and returns the
// container names that receive traffic.
func RoutedContainers(data []byte) (map[string]bool, error) {
	var cfg dynamicConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, svc := range cfg.HTTP.Services {
		for _, s := range svc.LoadBalancer.Servers {
			host := strings.TrimPrefix(s.URL, "http://")
			if i := strings.LastIndexByte(host, ':'); i >= 0 {
				host = host[:i]
			}
			out[host] = true
		}
	}
	return out, nil
}
