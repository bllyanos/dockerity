// Package settings holds instance-wide configuration edited from the UI.
package settings

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/routing"
)

// TLSMode is how Traefik gets certificates for app domains.
type TLSMode string

// TLS modes.
const (
	// TLSNone serves apps over plain HTTP.
	TLSNone TLSMode = "none"
	// TLSCertificate serves a wildcard certificate from files Traefik can read
	// (an internal CA, or a certificate obtained outside Dockerity).
	TLSCertificate TLSMode = "certificate"
	// TLSResolver lets a Traefik certificate resolver (for example Let's
	// Encrypt with a DNS challenge) obtain the wildcard certificate.
	TLSResolver TLSMode = "resolver"
)

// Settings are the instance-wide settings.
type Settings struct {
	// BaseDomain is the wildcard domain for ephemeral environments: they get
	// <env>.<app>.<BaseDomain>. Empty disables ephemeral environments.
	BaseDomain string
	TLSMode    TLSMode
	// CertFile and KeyFile are paths inside the Traefik container (certificate mode).
	CertFile string
	KeyFile  string
	// CertResolver is the Traefik resolver name (resolver mode).
	CertResolver string
}

// Keys in the settings table.
const (
	keyBaseDomain   = "base_domain"
	keyTLSMode      = "tls_mode"
	keyCertFile     = "tls_cert_file"
	keyKeyFile      = "tls_key_file"
	keyCertResolver = "tls_cert_resolver"
)

// Store persists key/value settings.
type Store interface {
	AllSettings(ctx context.Context) (map[string]string, error)
	SaveSettings(ctx context.Context, values map[string]string, at time.Time) error
}

// Service reads and updates settings.
type Service struct {
	Store Store
	Now   func() time.Time
	// OnChange, when set, runs after settings are saved, e.g. to rewrite
	// Traefik's TLS and route files.
	OnChange func(ctx context.Context, s Settings) error
}

// RouteOptions turns the TLS settings into routing options.
func (s Settings) RouteOptions() routing.Options {
	switch s.TLSMode {
	case TLSCertificate:
		return routing.Options{TLS: &routing.TLS{WildcardDomain: s.BaseDomain}}
	case TLSResolver:
		return routing.Options{TLS: &routing.TLS{CertResolver: s.CertResolver, WildcardDomain: s.BaseDomain}}
	default:
		return routing.DefaultOptions
	}
}

// RouteOptions returns the routing options of the current settings.
func (s *Service) RouteOptions(ctx context.Context) (routing.Options, error) {
	cur, err := s.Get(ctx)
	if err != nil {
		return routing.Options{}, err
	}
	return cur.RouteOptions(), nil
}

// Get returns the current settings, with TLS off by default.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	m, err := s.Store.AllSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	out := Settings{
		BaseDomain: m[keyBaseDomain], TLSMode: TLSMode(m[keyTLSMode]),
		CertFile: m[keyCertFile], KeyFile: m[keyKeyFile], CertResolver: m[keyCertResolver],
	}
	if out.TLSMode == "" {
		out.TLSMode = TLSNone
	}
	return out, nil
}

// Update validates and saves all settings.
func (s *Service) Update(ctx context.Context, in Settings) (Settings, error) {
	in.BaseDomain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(in.BaseDomain)), ".")
	if err := Validate(in); err != nil {
		return Settings{}, err
	}
	err := s.Store.SaveSettings(ctx, map[string]string{
		keyBaseDomain: in.BaseDomain, keyTLSMode: string(in.TLSMode),
		keyCertFile: in.CertFile, keyKeyFile: in.KeyFile, keyCertResolver: in.CertResolver,
	}, s.Now().UTC())
	if err != nil {
		return Settings{}, err
	}
	if s.OnChange != nil {
		if err := s.OnChange(ctx, in); err != nil {
			return in, fmt.Errorf("settings saved, but applying them failed: %w", err)
		}
	}
	return in, nil
}

var resolverName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Validate checks settings for consistency.
func Validate(s Settings) error {
	if err := domain.ValidateDomain(s.BaseDomain); err != nil {
		return domain.Invalid("baseDomain", "%q must be a hostname such as apps.example.com", s.BaseDomain)
	}
	switch s.TLSMode {
	case TLSNone:
	case TLSCertificate:
		for field, p := range map[string]string{"certFile": s.CertFile, "keyFile": s.KeyFile} {
			if !path.IsAbs(p) || path.Clean(p) != p {
				return domain.Invalid(field, "must be an absolute path inside the Traefik container")
			}
		}
	case TLSResolver:
		if !resolverName.MatchString(s.CertResolver) {
			return domain.Invalid("certResolver", "must be the name of a certificate resolver configured in Traefik")
		}
	default:
		return domain.Invalid("tlsMode", "must be none, certificate or resolver")
	}
	return nil
}
