package domain

import "testing"

func TestIsDNSLabel(t *testing.T) {
	for _, ok := range []string{"a", "shop", "my-app2", "a234567890123456789012345678901b"} {
		if !IsDNSLabel(ok) {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"", "1app", "-app", "app-", "My", "my_app", "a.b", "a2345678901234567890123456789012c"} {
		if IsDNSLabel(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}
