package routing

import (
	"maps"
	"strconv"
	"testing"
)

func TestLabelsRootRoute(t *testing.T) {
	got := Labels(Route{App: "shop", Env: "staging", Service: "web", Domain: "staging.shop.test", Path: "/", Port: 8080}, DefaultOptions)
	want := map[string]string{
		"traefik.enable":                                                      "true",
		"traefik.docker.network":                                              "dkt-shop-staging",
		"traefik.http.routers.dkt-shop-staging-web.rule":                      "Host(`staging.shop.test`)",
		"traefik.http.routers.dkt-shop-staging-web.entrypoints":               "web",
		"traefik.http.routers.dkt-shop-staging-web.service":                   "dkt-shop-staging-web",
		"traefik.http.routers.dkt-shop-staging-web.priority":                  strconv.Itoa(len("Host(`staging.shop.test`)")),
		"traefik.http.services.dkt-shop-staging-web.loadbalancer.server.port": "8080",
	}
	if !maps.Equal(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestLabelsSubPathAndPriority(t *testing.T) {
	api := Labels(Route{App: "shop", Env: "prod", Service: "api", Domain: "shop.test", Path: "/api/", Port: 9000}, Options{Entrypoints: []string{"web", "websecure"}})
	web := Labels(Route{App: "shop", Env: "prod", Service: "web", Domain: "shop.test", Port: 3000}, DefaultOptions)

	if api["traefik.http.routers.dkt-shop-prod-api.rule"] != "Host(`shop.test`) && PathPrefix(`/api`)" {
		t.Fatalf("rule %q", api["traefik.http.routers.dkt-shop-prod-api.rule"])
	}
	if api["traefik.http.routers.dkt-shop-prod-api.entrypoints"] != "web,websecure" {
		t.Fatal("entrypoints")
	}
	apiPrio, _ := strconv.Atoi(api["traefik.http.routers.dkt-shop-prod-api.priority"])
	webPrio, _ := strconv.Atoi(web["traefik.http.routers.dkt-shop-prod-web.priority"])
	if apiPrio <= webPrio {
		t.Fatalf("sub-path route must win: %d vs %d", apiPrio, webPrio)
	}
}

func TestLabelsWithoutDomain(t *testing.T) {
	if got := Labels(Route{App: "shop", Env: "dev", Service: "web", Port: 80}, DefaultOptions); len(got) != 0 {
		t.Fatalf("unrouted environment got labels: %v", got)
	}
}
