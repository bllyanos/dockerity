package deploykey

import (
	"context"
	"errors"

	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/secrets"
)

// secretName is the key under the app's secret scope that holds the private key.
const secretName = "DEPLOY_KEY"

// Store keeps private deploy keys in the secret store, under scope app:<id>.
type Store struct {
	Secrets secrets.SecretStore
}

// Save stores a new private key version for the app.
func (s Store) Save(ctx context.Context, appID int64, privatePEM []byte, actor *int64) error {
	_, err := s.Secrets.Update(ctx, secrets.AppScope(appID),
		secrets.Changes{Set: map[string]string{secretName: string(privatePEM)}}, actor)
	return err
}

// PrivateKey returns the app's current private key for clone or fetch.
// It must only be used in memory and never returned to API clients.
func (s Store) PrivateKey(ctx context.Context, appID int64) ([]byte, error) {
	scope := secrets.AppScope(appID)
	v, err := s.Secrets.Latest(ctx, scope)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.NotFoundf("app %d has no deploy key", appID)
	}
	if err != nil {
		return nil, err
	}
	values, err := s.Secrets.Values(ctx, scope, v.Version)
	if err != nil {
		return nil, err
	}
	key, ok := values[secretName]
	if !ok {
		return nil, domain.NotFoundf("app %d has no deploy key", appID)
	}
	return []byte(key), nil
}
