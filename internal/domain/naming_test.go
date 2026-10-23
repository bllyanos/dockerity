package domain

import "testing"

func TestNaming(t *testing.T) {
	if got := ImageTag("shop", "api", "abc123"); got != "dkt/shop/api:abc123" {
		t.Errorf("ImageTag %q", got)
	}
	if got := NetworkName("shop", "staging"); got != "dkt-shop-staging" {
		t.Errorf("NetworkName %q", got)
	}
	if got := ContainerName("shop", "staging", "api", "a1b2c3"); got != "dkt-shop-staging-api-a1b2c3" {
		t.Errorf("ContainerName %q", got)
	}
	if got := VolumeName("shop", "staging", "mysql", "data"); got != "dkt-shop-staging-mysql-data" {
		t.Errorf("VolumeName %q", got)
	}
	l := ImageLabels("shop", "api", "abc")
	if l[LabelApp] != "shop" || l[LabelService] != "api" || l[LabelCommit] != "abc" || len(l) != 3 {
		t.Errorf("ImageLabels %v", l)
	}
}
