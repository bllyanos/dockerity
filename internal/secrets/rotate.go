package secrets

import (
	"context"
	"fmt"

	"github.com/bllyanos/dockerity/internal/secrets/crypto"
)

// RotationStore reads and atomically replaces all wrapped data keys.
type RotationStore interface {
	ListDataKeys(ctx context.Context) ([]WrappedKey, error)
	// ReplaceDataKeys updates every given key in one transaction: all or nothing.
	ReplaceDataKeys(ctx context.Context, keys []WrappedKey) error
}

// Rotate rewraps every data key from oldMaster to newMaster. Secret set
// ciphertexts are untouched because the data keys themselves do not change.
// It refuses to run if any key is not wrapped by oldMaster. It returns the
// number of rewrapped keys.
func Rotate(ctx context.Context, store RotationStore, oldMaster, newMaster crypto.Key) (int, error) {
	if oldMaster == newMaster {
		return 0, fmt.Errorf("new master key is the same as the current one")
	}
	keys, err := store.ListDataKeys(ctx)
	if err != nil {
		return 0, err
	}
	rewrapped := make([]WrappedKey, 0, len(keys))
	for _, w := range keys {
		if w.MasterKeyID != oldMaster.ID() {
			return 0, fmt.Errorf("%w: scope %s uses %s, current key is %s", ErrWrongMasterKey, w.Scope, w.MasterKeyID, oldMaster.ID())
		}
		raw, err := crypto.Open(oldMaster, w.Sealed, wrapAAD(w.Scope))
		if err != nil {
			return 0, fmt.Errorf("unwrap data key for %s: %w", w.Scope, err)
		}
		sealed, err := crypto.Seal(newMaster, raw, wrapAAD(w.Scope))
		if err != nil {
			return 0, err
		}
		w.Sealed, w.MasterKeyID = sealed, newMaster.ID()
		rewrapped = append(rewrapped, w)
	}
	if err := store.ReplaceDataKeys(ctx, rewrapped); err != nil {
		return 0, err
	}
	return len(rewrapped), nil
}
