package domain

import (
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Audit actions for apps.
const (
	AuditAppCreated              = "app.created"
	AuditAppUpdated              = "app.updated"
	AuditAppDeleted              = "app.deleted"
	AuditAppDeployKeyRegenerated = "app.deploy_key_regenerated"
	AuditAppSpecUpdated          = "app.spec_updated"
)

// App is one repo deployed by Dockerity.
type App struct {
	ID            int64
	Name          string
	RepoURL       string
	DefaultBranch string
	// DeployPublicKey is the OpenSSH public key to add to the repo as a deploy key.
	DeployPublicKey string
	// MaxEphemeral limits active ephemeral environments; 0 means no limit.
	MaxEphemeral int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ValidateAppName checks an app name.
func ValidateAppName(name string) error {
	if !IsDNSLabel(name) {
		return Invalid("name", "%s", LabelRule)
	}
	return nil
}

// scpLikeRepo matches "git@github.com:org/repo.git".
var scpLikeRepo = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9.-]+:[A-Za-z0-9._~/-]+$`)

// ValidateRepoURL accepts SSH repo URLs, since apps authenticate with deploy keys:
// "git@host:org/repo.git" or "ssh://[user@]host[:port]/org/repo.git".
func ValidateRepoURL(raw string) error {
	if scpLikeRepo.MatchString(raw) && !strings.Contains(raw, "..") {
		return nil
	}
	u, err := url.Parse(raw)
	if err == nil && u.Scheme == "ssh" && u.Hostname() != "" && len(u.Path) > 1 && u.RawQuery == "" && u.Fragment == "" {
		if _, hasPassword := u.User.Password(); !hasPassword {
			return nil
		}
	}
	return Invalid("repoUrl", "must be an SSH URL such as git@github.com:org/repo.git or ssh://git@host/org/repo.git")
}

var branchForbidden = regexp.MustCompile(`[\s~^:?*\[\\]|\.\.|@\{|//`)

// ValidateBranch applies the parts of git's ref-name rules that matter for a branch name.
func ValidateBranch(b string) error {
	if b == "" || len(b) > 200 || strings.HasPrefix(b, "-") || strings.HasPrefix(b, "/") ||
		strings.HasSuffix(b, "/") || strings.HasSuffix(b, ".lock") || strings.HasSuffix(b, ".") ||
		branchForbidden.MatchString(b) {
		return Invalid("defaultBranch", "%q is not a valid branch name", b)
	}
	return nil
}

// AppSpecVersion is one immutable saved version of an app's spec.
type AppSpecVersion struct {
	AppID     int64
	Version   int
	YAML      string
	CreatedAt time.Time
	CreatedBy *int64
}
