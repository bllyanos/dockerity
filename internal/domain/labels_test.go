package domain

import "testing"

func TestIsSystem(t *testing.T) {
	cases := []struct {
		labels map[string]string
		want   bool
	}{
		{nil, false},
		{map[string]string{}, false},
		{map[string]string{LabelSystem: "true"}, true},
		{map[string]string{LabelSystem: "false"}, false},
		{map[string]string{LabelSystem: "TRUE"}, false},
		{map[string]string{LabelApp: "shop"}, false},
	}
	for _, c := range cases {
		if got := IsSystem(c.labels); got != c.want {
			t.Errorf("IsSystem(%v) = %v, want %v", c.labels, got, c.want)
		}
	}
}
