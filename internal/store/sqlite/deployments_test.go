package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// deployFixture creates an app, an environment and a build to deploy.
func deployFixture(t *testing.T) (*sql.DB, domain.Environment, domain.Build) {
	t.Helper()
	db := migratedDB(t)
	ctx := context.Background()
	app := newAppForTest(t, Apps{DB: db}, "shop")
	env, err := Environments{DB: db}.CreateEnvironment(ctx, domain.Environment{AppID: app.ID, Name: "staging", Type: domain.EnvPermanent, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Builds{DB: db}.CreateBuild(ctx, domain.Build{AppID: app.ID, Ref: "main", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	return db, env, b
}

func newDeployment(env domain.Environment, b domain.Build) domain.Deployment {
	return domain.Deployment{EnvID: env.ID, BuildID: b.ID, SpecVersion: 1, SecretVersion: 2, CreatedAt: time.Now()}
}

func TestEnvironmentLock(t *testing.T) {
	db, env, b := deployFixture(t)
	r := Deployments{DB: db}
	ctx := context.Background()

	d, err := r.CreateDeployment(ctx, newDeployment(env, b))
	if err != nil || d.Status != domain.DeploymentPending {
		t.Fatalf("create: %+v %v", d, err)
	}
	if _, err := r.CreateDeployment(ctx, newDeployment(env, b)); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second pending deployment: %v", err)
	}
	if err := r.StartDeployment(ctx, d.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateDeployment(ctx, newDeployment(env, b)); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("deployment while one runs: %v", err)
	}
	if err := r.FinishDeployment(ctx, d.ID, domain.DeploymentFailed, "boom", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateDeployment(ctx, newDeployment(env, b)); err != nil {
		t.Fatalf("lock must be released after finishing: %v", err)
	}
}

func TestEnvironmentLockUnderConcurrency(t *testing.T) {
	db, env, b := deployFixture(t)
	r := Deployments{DB: db}
	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Go(func() { _, errs[i] = r.CreateDeployment(context.Background(), newDeployment(env, b)) })
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, domain.ErrConflict):
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d concurrent deployments were accepted, want 1", ok)
	}
}

func TestCurrentDeploymentAndSteps(t *testing.T) {
	db, env, b := deployFixture(t)
	r := Deployments{DB: db}
	ctx := context.Background()
	if _, err := r.CurrentDeployment(ctx, env.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("no current yet: %v", err)
	}

	succeed := func() domain.Deployment {
		d, err := r.CreateDeployment(ctx, newDeployment(env, b))
		if err != nil {
			t.Fatal(err)
		}
		_ = r.StartDeployment(ctx, d.ID, time.Now())
		if err := r.FinishDeployment(ctx, d.ID, domain.DeploymentSucceeded, "", time.Now()); err != nil {
			t.Fatal(err)
		}
		return d
	}
	first := succeed()
	second := succeed()
	cur, err := r.CurrentDeployment(ctx, env.ID)
	if err != nil || cur.ID != second.ID {
		t.Fatalf("current %+v %v", cur, err)
	}
	if old, _ := r.GetDeployment(ctx, first.ID); old.Current {
		t.Fatal("previous deployment still current")
	}
	if err := r.FinishDeployment(ctx, second.ID, domain.DeploymentFailed, "", time.Now()); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("finished deployment changed: %v", err)
	}

	steps, err := r.InsertSteps(ctx, second.ID, []domain.DeploymentStep{{Kind: "ensure_network"}, {Kind: "job", Service: "migrate"}, {Kind: "finish"}})
	if err != nil || len(steps) != 3 || steps[1].Position != 2 || steps[1].ID == 0 {
		t.Fatalf("insert steps: %+v %v", steps, err)
	}
	code := 1
	_ = r.StartStep(ctx, steps[1].ID, time.Now())
	if err := r.FinishStep(ctx, steps[1].ID, domain.StepFailed, &code, "exit 1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := r.SkipPendingSteps(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := r.ListSteps(ctx, second.ID)
	if got[1].Status != domain.StepFailed || *got[1].ExitCode != 1 || got[1].StartedAt == nil || got[0].Status != domain.StepSkipped || got[2].Status != domain.StepSkipped {
		t.Fatalf("steps %+v", got)
	}

	list, _ := r.ListDeployments(ctx, env.ID, 10)
	if len(list) != 2 || list[0].ID != second.ID || list[0].SecretVersion != 2 {
		t.Fatalf("history %+v", list)
	}
}
