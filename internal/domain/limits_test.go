package domain

import "testing"

func TestResourceOverridesFor(t *testing.T) {
	o := ResourceOverrides{AllServices: {Memory: "256m", CPUs: "1"}, "db": {Memory: "2g"}}
	if got := o.For("db"); got != (ResourceOverride{Memory: "2g", CPUs: "1"}) {
		t.Fatalf("db %+v", got)
	}
	if got := o.For("web"); got != (ResourceOverride{Memory: "256m", CPUs: "1"}) {
		t.Fatalf("web %+v", got)
	}
	if got := (ResourceOverrides{}).For("web"); got != (ResourceOverride{}) {
		t.Fatalf("none %+v", got)
	}
}
