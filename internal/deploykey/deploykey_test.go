package deploykey

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
	"golang.org/x/crypto/ssh"
)

func TestGenerate(t *testing.T) {
	a, err := Generate("dockerity-shop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(a.Public, "ssh-ed25519 ") || !strings.HasSuffix(a.Public, " dockerity-shop") {
		t.Fatalf("public key %q", a.Public)
	}
	if !bytes.HasPrefix(a.PrivatePEM, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) {
		t.Fatal("private key is not OpenSSH PEM")
	}
	signer, err := Signer(a.PrivatePEM)
	if err != nil {
		t.Fatal(err)
	}
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(a.Public))
	if err != nil || !bytes.Equal(pub.Marshal(), signer.PublicKey().Marshal()) {
		t.Fatalf("public and private keys do not match: %v", err)
	}
	b, _ := Generate("dockerity-shop")
	if a.Public == b.Public {
		t.Fatal("keys must be random")
	}
	if s := fmt.Sprintf("%v %+v %#v", a, a, a); strings.Contains(s, "PRIVATE") {
		t.Fatal("formatting leaks the private key")
	}
}

// memSecrets is a minimal in-memory SecretStore.
type memSecrets struct {
	versions map[secrets.Scope][]map[string]string
}

func (m *memSecrets) Update(_ context.Context, s secrets.Scope, c secrets.Changes, _ *int64) (secrets.Version, error) {
	cur := map[string]string{}
	if vs := m.versions[s]; len(vs) > 0 {
		cur = vs[len(vs)-1]
	}
	next, err := c.Apply(cur)
	if err != nil {
		return secrets.Version{}, err
	}
	m.versions[s] = append(m.versions[s], next)
	return secrets.Version{Scope: s, Version: len(m.versions[s])}, nil
}

func (m *memSecrets) Latest(_ context.Context, s secrets.Scope) (secrets.Version, error) {
	if len(m.versions[s]) == 0 {
		return secrets.Version{}, domain.ErrNotFound
	}
	return secrets.Version{Scope: s, Version: len(m.versions[s])}, nil
}

func (m *memSecrets) Versions(context.Context, secrets.Scope) ([]secrets.Version, error) {
	return nil, nil
}

func (m *memSecrets) Values(_ context.Context, s secrets.Scope, v int) (map[string]string, error) {
	return m.versions[s][v-1], nil
}

func TestStore(t *testing.T) {
	st := Store{Secrets: &memSecrets{versions: map[secrets.Scope][]map[string]string{}}}
	ctx := context.Background()
	if _, err := st.PrivateKey(ctx, 1); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing key: %v", err)
	}
	if err := st.Save(ctx, 1, []byte("first"), nil); err != nil {
		t.Fatal(err)
	}
	if err := st.Save(ctx, 1, []byte("second"), nil); err != nil {
		t.Fatal(err)
	}
	got, err := st.PrivateKey(ctx, 1)
	if err != nil || string(got) != "second" {
		t.Fatalf("got %q %v", got, err)
	}
}
