package sqlite

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

func TestBackups(t *testing.T) {
	db, env, _ := deployFixture(t)
	r := Backups{DB: db}
	ctx := context.Background()

	if n, err := r.BackupRetention(ctx, env.ID); err != nil || n != domain.DefaultBackupRetention {
		t.Fatalf("default retention %d %v", n, err)
	}
	if err := r.SetBackupRetention(ctx, env.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := r.LatestBackup(ctx, "dkt-shop-staging-db-data"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("no backups yet: %v", err)
	}

	for i := range 4 {
		_, err := r.CreateBackup(ctx, domain.Backup{
			EnvID: &env.ID, AppName: "shop", EnvName: "staging", Service: "db", Volume: "dkt-shop-staging-db-data",
			File: fmt.Sprintf("b%d.tar.gz", i), SizeBytes: int64(100 + i), Reason: domain.BackupManual, CreatedAt: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	latest, err := r.LatestBackup(ctx, "dkt-shop-staging-db-data")
	if err != nil || latest.File != "b3.tar.gz" {
		t.Fatalf("latest %+v %v", latest, err)
	}
	old, err := r.BackupsBeyond(ctx, "dkt-shop-staging-db-data", 2)
	if err != nil || len(old) != 2 || old[0].File != "b1.tar.gz" || old[1].File != "b0.tar.gz" {
		t.Fatalf("beyond %+v %v", old, err)
	}
	if err := r.DeleteBackup(ctx, old[0].ID); err != nil {
		t.Fatal(err)
	}
	list, _ := r.ListBackups(ctx, env.ID)
	if len(list) != 3 || list[0].File != "b3.tar.gz" {
		t.Fatalf("list %+v", list)
	}

	// Backups outlive their environment.
	if err := (Environments{DB: db}).DeleteEnvironment(ctx, env.ID); err != nil {
		t.Fatal(err)
	}
	kept, err := r.GetBackup(ctx, latest.ID)
	if err != nil || kept.EnvID != nil || kept.EnvName != "staging" {
		t.Fatalf("after env delete %+v %v", kept, err)
	}
}
