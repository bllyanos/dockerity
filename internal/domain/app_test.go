package domain

import "testing"

func TestValidateRepoURL(t *testing.T) {
	ok := []string{
		"git@github.com:org/repo.git",
		"git@gitlab.example.com:group/sub/repo",
		"ssh://git@host.example.com/org/repo.git",
		"ssh://git@host.example.com:2222/org/repo.git",
		"ssh://host/repo",
	}
	bad := []string{
		"", "https://github.com/org/repo.git", "github.com/org/repo", "git@github.com",
		"ssh://git:secret@host/org/repo", "ssh:///repo", "ssh://host/", "file:///tmp/repo",
		"git@github.com:org/../etc", "git@github.com:org/repo.git?x=1",
	}
	for _, s := range ok {
		if err := ValidateRepoURL(s); err != nil {
			t.Errorf("%q rejected: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := ValidateRepoURL(s); err == nil {
			t.Errorf("%q accepted", s)
		}
	}
}

func TestValidateBranch(t *testing.T) {
	for _, ok := range []string{"main", "release/1.2", "feat-x", "v1.0.0"} {
		if err := ValidateBranch(ok); err != nil {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"", "-x", "/x", "x/", "a..b", "a b", "x.lock", "a~1", "a:b", "x@{y", "a//b", "end."} {
		if err := ValidateBranch(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestValidateAppName(t *testing.T) {
	if ValidateAppName("shop") != nil || ValidateAppName("Shop") == nil {
		t.Fatal("app name rule")
	}
}
