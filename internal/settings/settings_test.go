package settings

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

type memStore map[string]string

func (m memStore) AllSettings(context.Context) (map[string]string, error) { return m, nil }
func (m memStore) SaveSettings(_ context.Context, v map[string]string, _ time.Time) error {
	for k, x := range v {
		m[k] = x
	}
	return nil
}

func TestSettingsRoundTrip(t *testing.T) {
	s := &Service{Store: memStore{}, Now: time.Now}
	ctx := context.Background()
	got, err := s.Get(ctx)
	if err != nil || got.TLSMode != TLSNone || got.BaseDomain != "" {
		t.Fatalf("defaults %+v %v", got, err)
	}
	saved, err := s.Update(ctx, Settings{BaseDomain: " Apps.Example.COM. ", TLSMode: TLSResolver, CertResolver: "letsencrypt"})
	if err != nil || saved.BaseDomain != "apps.example.com" {
		t.Fatalf("update %+v %v", saved, err)
	}
	if got, _ := s.Get(ctx); got != saved {
		t.Fatalf("get %+v want %+v", got, saved)
	}
}

func TestValidate(t *testing.T) {
	ok := []Settings{
		{TLSMode: TLSNone},
		{BaseDomain: "apps.example.com", TLSMode: TLSCertificate, CertFile: "/certs/wild.crt", KeyFile: "/certs/wild.key"},
		{BaseDomain: "apps.example.com", TLSMode: TLSResolver, CertResolver: "le-dns"},
	}
	bad := []Settings{
		{BaseDomain: "not a domain", TLSMode: TLSNone},
		{TLSMode: "magic"},
		{TLSMode: TLSCertificate, CertFile: "certs/x.crt", KeyFile: "/k"},
		{TLSMode: TLSCertificate, CertFile: "/c"},
		{TLSMode: TLSResolver, CertResolver: "bad name"},
	}
	for _, s := range ok {
		if err := Validate(s); err != nil {
			t.Errorf("%+v rejected: %v", s, err)
		}
	}
	var ve *domain.ValidationError
	for _, s := range bad {
		if err := Validate(s); !errors.As(err, &ve) {
			t.Errorf("%+v accepted: %v", s, err)
		}
	}
}
