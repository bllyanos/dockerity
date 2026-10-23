package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/audit"
	"github.com/bllyanos/dockerity/internal/domain"
)

func TestAuditRecordAndList(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	u, _ := Users{DB: db}.CreateFirstAdmin(ctx, newUser("alice"))
	log := AuditLog{DB: db}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 5 {
		e := domain.AuditEvent{At: t0.Add(time.Duration(i) * time.Second), Action: domain.AuditLogin, RemoteAddr: "1.2.3.4"}
		if i%2 == 0 {
			e.ActorUserID = &u.ID
		}
		if err := log.Record(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	page, err := log.ListAudit(ctx, 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 3 || page[0].ID <= page[1].ID {
		t.Fatalf("want newest first, got %+v", page)
	}
	if page[0].ActorUsername != "alice" || page[0].ActorUserID == nil || *page[0].ActorUserID != u.ID {
		t.Fatalf("actor not resolved: %+v", page[0])
	}
	if page[1].ActorUserID != nil || page[1].ActorUsername != "" {
		t.Fatalf("anonymous event has actor: %+v", page[1])
	}

	rest, err := log.ListAudit(ctx, page[2].ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rest) != 2 || rest[0].ID >= page[2].ID {
		t.Fatalf("second page wrong: %+v", rest)
	}
}

func TestAuditNeverStoresSecretValues(t *testing.T) {
	db := migratedDB(t)
	ctx := context.Background()
	log := AuditLog{DB: db}
	err := log.Record(ctx, domain.AuditEvent{
		At:       time.Now(),
		Action:   "secret.updated",
		Metadata: map[string]string{"secret_name": "DB_PASSWORD", "value": "hunter2", "password": "hunter2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := db.QueryRow(`SELECT metadata FROM audit_events`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "hunter2") || !strings.Contains(raw, audit.Redacted) || !strings.Contains(raw, "DB_PASSWORD") {
		t.Fatalf("stored metadata %s", raw)
	}
}
