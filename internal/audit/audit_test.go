package audit

import "testing"

func TestSanitize(t *testing.T) {
	in := map[string]string{
		"username":     "alice",
		"password":     "hunter2",
		"New_Password": "hunter3",
		"secret":       "s",
		"secret_value": "v",
		"api_key":      "k",
		"token":        "dkta_x",
		"token_id":     "7",
		"secret_name":  "DB_PASSWORD",
		"key_count":    "3",
	}
	got := Sanitize(in)
	redacted := []string{"password", "New_Password", "secret", "secret_value", "api_key", "token"}
	kept := []string{"username", "token_id", "secret_name", "key_count"}
	for _, k := range redacted {
		if got[k] != Redacted {
			t.Errorf("%s not redacted: %q", k, got[k])
		}
	}
	for _, k := range kept {
		if got[k] != in[k] {
			t.Errorf("%s changed: %q", k, got[k])
		}
	}
	if in["password"] != "hunter2" {
		t.Error("input map must not be modified")
	}
	if Sanitize(nil) == nil {
		t.Error("nil metadata should become an empty map")
	}
}
