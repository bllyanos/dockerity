package domain

import (
	"regexp"
	"strings"
	"time"
)

// Audit actions for environments.
const (
	AuditEnvironmentCreated = "environment.created"
	AuditEnvironmentUpdated = "environment.updated"
	AuditEnvironmentDeleted = "environment.deleted"
)

// EnvType tells permanent environments from ephemeral ones.
type EnvType string

// Environment types.
const (
	EnvPermanent EnvType = "permanent"
	EnvEphemeral EnvType = "ephemeral"
)

// Environment is a running copy of an app.
type Environment struct {
	ID     int64
	AppID  int64
	Name   string
	Type   EnvType
	Domain string // empty when the environment is not routed
	// AppName is filled when reading, for naming Docker objects.
	AppName   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ValidateEnvName checks an environment name.
func ValidateEnvName(name string) error {
	if !IsDNSLabel(name) {
		return Invalid("name", "%s", LabelRule)
	}
	return nil
}

var hostLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidateDomain checks a lowercase hostname such as "shop.example.com".
// An empty domain means the environment is not routed.
func ValidateDomain(domain string) error {
	if domain == "" {
		return nil
	}
	labels := strings.Split(domain, ".")
	if len(domain) > 253 || len(labels) < 2 {
		return Invalid("domain", "%q must be a hostname such as shop.example.com", domain)
	}
	for _, l := range labels {
		if !hostLabel.MatchString(l) {
			return Invalid("domain", "%q must be a lowercase hostname such as shop.example.com", domain)
		}
	}
	return nil
}
