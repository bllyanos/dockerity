// Package backups archives and restores the volumes of stateful services
// (docs/design.md section 9).
package backups

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/bllyanos/dockerity/internal/dockerx"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

// Store persists backup records.
type Store interface {
	CreateBackup(ctx context.Context, b domain.Backup) (domain.Backup, error)
	GetBackup(ctx context.Context, id int64) (domain.Backup, error)
	ListBackups(ctx context.Context, envID int64) ([]domain.Backup, error)
	BackupsBeyond(ctx context.Context, volume string, keep int) ([]domain.Backup, error)
	DeleteBackup(ctx context.Context, id int64) error
	BackupRetention(ctx context.Context, envID int64) (int, error)
}

// Docker is what backups need from the engine.
type Docker interface {
	ListManaged(ctx context.Context, labels map[string]string) ([]dockerx.ContainerInfo, error)
	StopContainer(ctx context.Context, id string, grace time.Duration) error
	StartContainer(ctx context.Context, id string) error
	ArchiveVolume(ctx context.Context, volume, backupsVolume, file string) (int64, error)
	RestoreVolume(ctx context.Context, volume, backupsVolume, file string) error
	RemoveArchive(ctx context.Context, backupsVolume, file string) error
}

// Specs returns the spec running in an environment.
type Specs interface {
	CurrentSpec(ctx context.Context, env domain.Environment) (spec.Spec, error)
}

// Service takes, lists and restores backups.
type Service struct {
	Envs interface {
		Get(ctx context.Context, id int64) (domain.Environment, error)
	}
	Store         Store
	Docker        Docker
	Specs         Specs
	BackupsVolume string
	Now           func() time.Time
}

// List returns an environment's backups, newest first.
func (s *Service) List(ctx context.Context, envID int64) ([]domain.Backup, error) {
	if _, err := s.Envs.Get(ctx, envID); err != nil {
		return nil, err
	}
	return s.Store.ListBackups(ctx, envID)
}

// BackupService backs up every volume of a stateful service: its running
// containers are stopped around the archive so the copy is consistent.
func (s *Service) BackupService(ctx context.Context, envID int64, service string, reason domain.BackupReason) ([]domain.Backup, error) {
	env, err := s.Envs.Get(ctx, envID)
	if err != nil {
		return nil, err
	}
	svc, err := s.statefulService(ctx, env, service)
	if err != nil {
		return nil, err
	}
	var out []domain.Backup
	err = s.withServiceStopped(ctx, env, svc, func() error {
		out, err = s.Archive(ctx, env, svc, reason)
		return err
	})
	return out, err
}

// Archive backs up the service's volumes without touching its containers;
// callers make sure nothing writes to them. Old backups beyond the
// environment's retention are pruned.
func (s *Service) Archive(ctx context.Context, env domain.Environment, svc spec.Service, reason domain.BackupReason) ([]domain.Backup, error) {
	keep, err := s.Store.BackupRetention(ctx, env.ID)
	if err != nil {
		return nil, err
	}
	var out []domain.Backup
	for _, v := range svc.Volumes {
		volume := domain.VolumeName(env.AppName, env.Name, svc.Name, v.Name)
		file, err := archiveFile(volume, s.Now())
		if err != nil {
			return out, err
		}
		size, err := s.Docker.ArchiveVolume(ctx, volume, s.BackupsVolume, file)
		if err != nil {
			return out, err
		}
		envID := env.ID
		b, err := s.Store.CreateBackup(ctx, domain.Backup{
			EnvID: &envID, AppName: env.AppName, EnvName: env.Name, Service: svc.Name, Volume: volume,
			File: file, SizeBytes: size, Reason: reason, CreatedAt: s.Now().UTC(),
		})
		if err != nil {
			return out, err
		}
		out = append(out, b)
		if err := s.prune(ctx, volume, keep); err != nil {
			return out, err
		}
	}
	return out, nil
}

// ArchiveEnvironment backs up the volumes of every stateful service of the
// environment's current deployment, for example before deleting it. Its
// containers must already be stopped. Nothing deployed means nothing to save.
func (s *Service) ArchiveEnvironment(ctx context.Context, env domain.Environment) ([]domain.Backup, error) {
	sp, err := s.Specs.CurrentSpec(ctx, env)
	if errors.Is(err, domain.ErrConflict) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []domain.Backup
	for _, svc := range sp.Services {
		if svc.Kind != spec.KindStateful || len(svc.Volumes) == 0 {
			continue
		}
		list, err := s.Archive(ctx, env, svc, domain.BackupEnvDelete)
		out = append(out, list...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// Restore replaces a volume's content with a backup, stopping the service's
// containers while it runs. The service is down meanwhile.
func (s *Service) Restore(ctx context.Context, backupID int64) (domain.Backup, error) {
	b, err := s.Store.GetBackup(ctx, backupID)
	if errors.Is(err, domain.ErrNotFound) {
		return b, domain.NotFoundf("backup %d not found", backupID)
	}
	if err != nil {
		return b, err
	}
	if b.EnvID == nil {
		return b, domain.Conflictf("the environment of backup %d was deleted", backupID)
	}
	env, err := s.Envs.Get(ctx, *b.EnvID)
	if err != nil {
		return b, err
	}
	svc, err := s.statefulService(ctx, env, b.Service)
	if err != nil {
		return b, err
	}
	return b, s.withServiceStopped(ctx, env, svc, func() error {
		return s.Docker.RestoreVolume(ctx, b.Volume, s.BackupsVolume, b.File)
	})
}

func (s *Service) statefulService(ctx context.Context, env domain.Environment, name string) (spec.Service, error) {
	sp, err := s.Specs.CurrentSpec(ctx, env)
	if err != nil {
		return spec.Service{}, err
	}
	svc, ok := sp.Service(name)
	if !ok || svc.Kind != spec.KindStateful || len(svc.Volumes) == 0 {
		return spec.Service{}, domain.Invalid("service", "%s is not a stateful service with volumes in %s", name, env.Name)
	}
	return svc, nil
}

// withServiceStopped stops the service's running containers, runs fn and
// starts them again, even when fn fails.
func (s *Service) withServiceStopped(ctx context.Context, env domain.Environment, svc spec.Service, fn func() error) error {
	list, err := s.Docker.ListManaged(ctx, map[string]string{
		domain.LabelApp: env.AppName, domain.LabelEnv: env.Name, domain.LabelService: svc.Name,
	})
	if err != nil {
		return err
	}
	var stopped []string
	for _, c := range list {
		if !c.Running {
			continue
		}
		if err := s.Docker.StopContainer(ctx, c.ID, svc.EffectiveStopGracePeriod()); err != nil {
			return err
		}
		stopped = append(stopped, c.ID)
	}
	ferr := fn()
	var errs []error
	for _, id := range stopped {
		if err := s.Docker.StartContainer(context.Background(), id); err != nil {
			errs = append(errs, fmt.Errorf("restart %.12s: %w", id, err))
		}
	}
	return errors.Join(append([]error{ferr}, errs...)...)
}

func (s *Service) prune(ctx context.Context, volume string, keep int) error {
	old, err := s.Store.BackupsBeyond(ctx, volume, keep)
	if err != nil {
		return err
	}
	for _, b := range old {
		if err := s.Docker.RemoveArchive(ctx, s.BackupsVolume, b.File); err != nil {
			return err
		}
		if err := s.Store.DeleteBackup(ctx, b.ID); err != nil {
			return err
		}
	}
	return nil
}

// archiveFile names an archive after its volume and time, plus a random suffix.
func archiveFile(volume string, now time.Time) (string, error) {
	suffix := make([]byte, 3)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s.tar.gz", volume, now.UTC().Format("20060102T150405Z"), hex.EncodeToString(suffix)), nil
}
