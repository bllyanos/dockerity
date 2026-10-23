package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func parse(t *testing.T, data []byte) dynamicConfig {
	t.Helper()
	var cfg dynamicConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestConfig(t *testing.T) {
	data, err := Config(Env{
		App: "shop", Env: "prod", Domain: "shop.test",
		Services: []Service{
			{Name: "web", Port: 3000, Containers: []string{"dkt-shop-prod-web-d2", "dkt-shop-prod-web-d1"}},
			{Name: "api", Path: "/api/", Port: 9000, Containers: []string{"dkt-shop-prod-api-d2"}},
			{Name: "idle", Port: 1},
		},
	}, Options{Entrypoints: []string{"web", "websecure"}})
	if err != nil {
		t.Fatal(err)
	}
	cfg := parse(t, data)
	web, api := cfg.HTTP.Routers["dkt-shop-prod-web"], cfg.HTTP.Routers["dkt-shop-prod-api"]
	if web.Rule != "Host(`shop.test`)" || api.Rule != "Host(`shop.test`) && PathPrefix(`/api`)" {
		t.Fatalf("rules %q %q", web.Rule, api.Rule)
	}
	if api.Priority <= web.Priority || web.Service != "dkt-shop-prod-web" || strings.Join(web.EntryPoints, ",") != "web,websecure" {
		t.Fatalf("routers %+v %+v", web, api)
	}
	servers := cfg.HTTP.Services["dkt-shop-prod-web"].LoadBalancer.Servers
	if len(servers) != 2 || servers[0].URL != "http://dkt-shop-prod-web-d1:3000" || servers[1].URL != "http://dkt-shop-prod-web-d2:3000" {
		t.Fatalf("servers %+v", servers)
	}
	if _, ok := cfg.HTTP.Routers["dkt-shop-prod-idle"]; ok {
		t.Fatal("a service without containers must not be routed")
	}
}

func TestConfigWithoutDomain(t *testing.T) {
	data, _ := Config(Env{App: "a", Env: "e", Services: []Service{{Name: "web", Port: 80, Containers: []string{"c"}}}}, DefaultOptions)
	if cfg := parse(t, data); len(cfg.HTTP.Routers) != 0 {
		t.Fatalf("unrouted environment got routers: %s", data)
	}
}

func TestFiles(t *testing.T) {
	f := Files{Dir: filepath.Join(t.TempDir(), "dynamic")}
	if err := f.Write("shop", "prod", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := f.Write("shop", "prod", []byte("two")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(f.Dir, "dkt-shop-prod.yml"))
	if err != nil || string(data) != "two" {
		t.Fatalf("got %q %v", data, err)
	}
	entries, _ := os.ReadDir(f.Dir)
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
	if err := f.Remove("shop", "prod"); err != nil {
		t.Fatal(err)
	}
	if err := f.Remove("shop", "prod"); err != nil {
		t.Fatalf("removing a missing file: %v", err)
	}
}

func TestRoutedContainers(t *testing.T) {
	f := Files{Dir: t.TempDir()}
	if got, err := f.Routed("shop", "prod"); err != nil || len(got) != 0 {
		t.Fatalf("missing file: %v %v", got, err)
	}
	data, _ := Config(Env{App: "shop", Env: "prod", Domain: "shop.test", Services: []Service{
		{Name: "web", Port: 80, Containers: []string{"dkt-shop-prod-web-d1", "dkt-shop-prod-web-d2"}},
	}}, DefaultOptions)
	_ = f.Write("shop", "prod", data)
	got, err := f.Routed("shop", "prod")
	if err != nil || !got["dkt-shop-prod-web-d1"] || !got["dkt-shop-prod-web-d2"] || len(got) != 2 {
		t.Fatalf("routed %v %v", got, err)
	}
}

func TestConfigWithTLS(t *testing.T) {
	routes := func(host string) Env {
		return Env{App: "shop", Env: "pr-1", Domain: host, Services: []Service{{Name: "web", Port: 80, Containers: []string{"c1"}}}}
	}
	resolver := Options{TLS: &TLS{CertResolver: "le", WildcardDomain: "apps.test"}}

	data, err := Config(routes("pr-1-shop.apps.test"), resolver)
	if err != nil {
		t.Fatal(err)
	}
	cfg := parse(t, data)
	secure, plain := cfg.HTTP.Routers["dkt-shop-pr-1-web"], cfg.HTTP.Routers["dkt-shop-pr-1-web-http"]
	if strings.Join(secure.EntryPoints, ",") != "websecure" || secure.TLS == nil || secure.TLS.CertResolver != "le" ||
		len(secure.TLS.Domains) != 1 || secure.TLS.Domains[0].Main != "*.apps.test" {
		t.Fatalf("secure router %+v", secure)
	}
	if strings.Join(plain.EntryPoints, ",") != "web" || len(plain.Middlewares) != 1 {
		t.Fatalf("redirect router %+v", plain)
	}
	if mw := cfg.HTTP.Middlewares[plain.Middlewares[0]]; mw.RedirectScheme == nil || mw.RedirectScheme.Scheme != "https" {
		t.Fatalf("middleware %+v", cfg.HTTP.Middlewares)
	}

	// A permanent domain outside the base domain gets its own certificate from the resolver.
	cfg = parse(t, must(Config(routes("shop.example.com"), resolver)))
	if r := cfg.HTTP.Routers["dkt-shop-pr-1-web"]; r.TLS == nil || len(r.TLS.Domains) != 0 || r.TLS.CertResolver != "le" {
		t.Fatalf("custom domain %+v", r)
	}

	// Certificate mode: TLS on, Traefik's default (wildcard) certificate.
	cfg = parse(t, must(Config(routes("pr-1-shop.apps.test"), Options{TLS: &TLS{WildcardDomain: "apps.test"}})))
	if r := cfg.HTTP.Routers["dkt-shop-pr-1-web"]; r.TLS == nil || r.TLS.CertResolver != "" || len(r.TLS.Domains) != 0 {
		t.Fatalf("certificate mode %+v", r)
	}
}

func must(data []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return data
}

func TestCertificateConfig(t *testing.T) {
	f := Files{Dir: t.TempDir()}
	data, err := CertificateConfig("/certs/wild.crt", "/certs/wild.key")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "certFile: /certs/wild.crt") || !strings.Contains(string(data), "defaultCertificate") {
		t.Fatalf("tls config:\n%s", data)
	}
	if err := f.WriteTLS(data); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.Dir, "dkt-tls.yml")); err != nil {
		t.Fatal(err)
	}
	if err := f.RemoveTLS(); err != nil {
		t.Fatal(err)
	}
	if err := f.RemoveTLS(); err != nil {
		t.Fatal(err)
	}
}
