package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bllyanos/dockerity/internal/config"
	"github.com/bllyanos/dockerity/internal/secrets"
	"github.com/bllyanos/dockerity/internal/secrets/crypto"
	"github.com/bllyanos/dockerity/internal/store/sqlite"
)

const rotateSteps = `Rotated %d data key(s) to master key %s.

Next steps:
  1. Replace the master_key Docker secret file with the new key.
  2. Restart Dockerity: docker compose up -d dockerity
  3. Keep the old key until Dockerity starts cleanly, then destroy it.
`

// rotateMasterKey rewraps every data key with a new master key. Run it while
// the Dockerity server is stopped, so no data key is created with the old key.
func rotateMasterKey(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("rotate-master-key", flag.ContinueOnError)
	newKeyFile := fs.String("new-key-file", "", "file holding the new base64 master key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *newKeyFile == "" {
		return errors.New("rotate-master-key: --new-key-file is required")
	}

	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	oldMaster, err := crypto.ReadKeyFile(cfg.MasterKeyFile)
	if err != nil {
		return fmt.Errorf("current master key: %w", err)
	}
	newMaster, err := crypto.ReadKeyFile(*newKeyFile)
	if err != nil {
		return fmt.Errorf("new master key: %w", err)
	}

	db, err := sqlite.Open(ctx, cfg.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	store := sqlite.DataKeys{DB: db}
	if err := secrets.NewKeyring(oldMaster, store, time.Now).Verify(ctx); err != nil {
		return fmt.Errorf("current master key: %w", err)
	}
	n, err := secrets.Rotate(ctx, store, oldMaster, newMaster)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, rotateSteps, n, newMaster.ID())
	return err
}
