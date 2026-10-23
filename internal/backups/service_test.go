package backups

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

type fakeDocker struct {
	ops         []string
	containers  []dockerx.ContainerInfo
	failArchive bool
}

func (f *fakeDocker) ListManaged(context.Context, map[string]string) ([]dockerx.ContainerInfo, error) {
	return f.containers, nil
}
func (f *fakeDocker) StopContainer(_ context.Context, id string, _ time.Duration) error {
	f.ops = append(f.ops, "stop "+id)
	return nil
}
func (f *fakeDocker) StartContainer(_ context.Context, id string) error {
	f.ops = append(f.ops, "start "+id)
	return nil
}
func (f *fakeDocker) ArchiveVolume(_ context.Context, volume, _, file string) (int64, error) {
	if f.failArchive {
		return 0, errors.New("disk full")
	}
	f.ops = append(f.ops, "archive "+volume)
	return 1234, nil
}
func (f *fakeDocker) RestoreVolume(_ context.Context, volume, _, file string) error {
	f.ops = append(f.ops, "restore "+volume+" "+file)
	return nil
}
func (f *fakeDocker) RemoveArchive(_ context.Context, _, file string) error {
	f.ops = append(f.ops, "rm "+file)
	return nil
}

type memStore struct {
	backups []domain.Backup
	keep    int
}

func (m *memStore) CreateBackup(_ context.Context, b domain.Backup) (domain.Backup, error) {
	b.ID = int64(len(m.backups) + 1)
	m.backups = append(m.backups, b)
	return b, nil
}
func (m *memStore) GetBackup(_ context.Context, id int64) (domain.Backup, error) {
	for _, b := range m.backups {
		if b.ID == id {
			return b, nil
		}
	}
	return domain.Backup{}, domain.ErrNotFound
}
func (m *memStore) ListBackups(context.Context, int64) ([]domain.Backup, error) {
	return m.backups, nil
}
func (m *memStore) BackupsBeyond(_ context.Context, volume string, keep int) ([]domain.Backup, error) {
	var mine []domain.Backup
	for i := len(m.backups) - 1; i >= 0; i-- {
		if m.backups[i].Volume == volume {
			mine = append(mine, m.backups[i])
		}
	}
	if len(mine) <= keep {
		return nil, nil
	}
	return mine[keep:], nil
}
func (m *memStore) DeleteBackup(_ context.Context, id int64) error {
	for i, b := range m.backups {
		if b.ID == id {
			m.backups = append(m.backups[:i], m.backups[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}
func (m *memStore) BackupRetention(context.Context, int64) (int, error) { return m.keep, nil }

type env struct{}

func (env) Get(_ context.Context, id int64) (domain.Environment, error) {
	return domain.Environment{ID: id, AppID: 1, AppName: "shop", Name: "prod"}, nil
}

type specs struct{ s spec.Spec }

func (s specs) CurrentSpec(context.Context, domain.Environment) (spec.Spec, error) { return s.s, nil }

func newService(t *testing.T) (*Service, *fakeDocker, *memStore) {
	t.Helper()
	sp, err := spec.Load([]byte("services:\n  db:\n    kind: stateful\n    image: mysql:8.4\n    volumes:\n      data: /var/lib/mysql\n      logs: /var/log/mysql\n  web:\n    kind: deployment\n    strategy: recreate\n    image: nginx\n"))
	if err != nil {
		t.Fatal(err)
	}
	d := &fakeDocker{containers: []dockerx.ContainerInfo{{ID: "db1", Running: true}, {ID: "old", Running: false}}}
	st := &memStore{keep: 2}
	return &Service{Envs: env{}, Store: st, Docker: d, Specs: specs{sp}, BackupsVolume: "dockerity-backups", Now: time.Now}, d, st
}

func TestBackupServiceStopsAroundArchive(t *testing.T) {
	s, d, st := newService(t)
	got, err := s.BackupService(context.Background(), 1, "db", domain.BackupManual)
	if err != nil || len(got) != 2 {
		t.Fatalf("backups %+v %v", got, err)
	}
	want := "stop db1,archive dkt-shop-prod-db-data,archive dkt-shop-prod-db-logs,start db1"
	if strings.Join(d.ops, ",") != want {
		t.Fatalf("ops %v", d.ops)
	}
	name := regexp.MustCompile(`^dkt-shop-prod-db-data-\d{8}T\d{6}Z-[0-9a-f]{6}\.tar\.gz$`)
	if !name.MatchString(got[0].File) || got[0].SizeBytes != 1234 || got[0].Reason != domain.BackupManual || *got[0].EnvID != 1 {
		t.Fatalf("backup %+v", got[0])
	}
	if len(st.backups) != 2 {
		t.Fatalf("stored %d", len(st.backups))
	}
}

func TestRetentionPrunesOldArchives(t *testing.T) {
	s, d, st := newService(t)
	for range 3 {
		if _, err := s.BackupService(context.Background(), 1, "db", domain.BackupManual); err != nil {
			t.Fatal(err)
		}
	}
	if len(st.backups) != 4 { // 2 volumes x keep 2
		t.Fatalf("kept %d backups", len(st.backups))
	}
	removed := 0
	for _, op := range d.ops {
		if strings.HasPrefix(op, "rm ") {
			removed++
		}
	}
	if removed != 2 {
		t.Fatalf("removed %d archives: %v", removed, d.ops)
	}
}

func TestBackupRestartsEvenWhenArchiveFails(t *testing.T) {
	s, d, _ := newService(t)
	d.failArchive = true
	if _, err := s.BackupService(context.Background(), 1, "db", domain.BackupManual); err == nil {
		t.Fatal("expected error")
	}
	if strings.Join(d.ops, ",") != "stop db1,start db1" {
		t.Fatalf("ops %v", d.ops)
	}
}

func TestBackupRejectsNonStateful(t *testing.T) {
	s, _, _ := newService(t)
	var ve *domain.ValidationError
	for _, svc := range []string{"web", "missing"} {
		if _, err := s.BackupService(context.Background(), 1, svc, domain.BackupManual); !errors.As(err, &ve) {
			t.Errorf("%s: %v", svc, err)
		}
	}
}

func TestRestore(t *testing.T) {
	s, d, st := newService(t)
	backups, _ := s.BackupService(context.Background(), 1, "db", domain.BackupManual)
	d.ops = nil
	if _, err := s.Restore(context.Background(), backups[0].ID); err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("stop db1,restore dkt-shop-prod-db-data %s,start db1", backups[0].File)
	if strings.Join(d.ops, ",") != want {
		t.Fatalf("ops %v", d.ops)
	}
	st.backups[0].EnvID = nil
	if _, err := s.Restore(context.Background(), backups[0].ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("deleted environment: %v", err)
	}
	if _, err := s.Restore(context.Background(), 99); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestArchiveEnvironment(t *testing.T) {
	s, d, st := newService(t)
	got, err := s.ArchiveEnvironment(context.Background(), domain.Environment{ID: 1, AppName: "shop", Name: "prod"})
	if err != nil || len(got) != 2 || got[0].Reason != domain.BackupEnvDelete {
		t.Fatalf("backups %+v %v", got, err)
	}
	for _, op := range d.ops {
		if strings.HasPrefix(op, "stop") || strings.HasPrefix(op, "start") {
			t.Fatalf("containers are already gone; got %v", d.ops)
		}
	}
	if len(st.backups) != 2 {
		t.Fatalf("stored %d", len(st.backups))
	}
}

type nothingDeployed struct{}

func (nothingDeployed) CurrentSpec(context.Context, domain.Environment) (spec.Spec, error) {
	return spec.Spec{}, domain.Conflictf("nothing is deployed")
}

func TestArchiveEnvironmentWithNothingDeployed(t *testing.T) {
	s, _, _ := newService(t)
	s.Specs = nothingDeployed{}
	if got, err := s.ArchiveEnvironment(context.Background(), domain.Environment{ID: 1}); err != nil || len(got) != 0 {
		t.Fatalf("got %v %v", got, err)
	}
}
