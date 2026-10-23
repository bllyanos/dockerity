# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

The design lives in `docs/design.md`. Read it before implementing anything, and update it
when a design decision changes.

## Commands

Go is pinned in `mise.toml`, Node in `.nvmrc`. Lint needs `golangci-lint` v2 on `PATH`
(or `make lint GOLANGCI_LINT=/path/to/golangci-lint`).

```sh
make web-deps          # npm ci in web/ (once)
make build             # build UI into web/dist, then bin/dockerity
make build-go          # Go binary only (embeds whatever is in web/dist)
make test              # go test ./...
make lint              # golangci-lint + oxlint
make run               # build and run with data in ./.data (needs Docker socket)
go test ./internal/store/sqlite -run TestMigrateIsIdempotent   # single test
cd web && npm run dev  # Vite dev server, proxies /api and /healthz to :8080
```

Bootstrap stack: `deploy/docker-compose.yml` (see README). CI runs Go lint and race tests,
web lint and build, and a Docker image build (`.github/workflows/ci.yml`).

`web/dist/.gitkeep` is tracked so `go:embed` compiles before the UI is built; the web
build re-creates it. Server config is env vars (`DOCKERITY_*`, see `internal/config`).

## What Dockerity is

An internal, containerized Docker wrapper for a **single VM**. It builds apps from Git,
deploys them into environments, tracks which build runs where, and owns all app config
and secrets. Dockerity itself runs as a container on the same VM it manages.

Stack: Go backend, React + Tailwind (Vite) frontend embedded into the Go binary via
`go:embed`, SQLite (`modernc.org/sqlite`, pure Go, WAL mode) on a Docker volume.

## Core concepts

- **App**: one repo. Has many services, each either built from the repo or using a public image.
- **App spec**: compose-like YAML held and versioned by Dockerity, not stored in the repo.
- **Service kinds**: `job` (runs once, e.g. migration, phase `pre-deploy` or `post-deploy`),
  `deployment` (long-running, stateless), `stateful` (long-running with persistent named volumes).
- **Environment**: `permanent` (fixed domain) or `ephemeral` (auto subdomain under a wildcard
  domain, TTL, removed by the reaper).
- **Deployment**: immutable record of Build + AppSpecVersion + SecretSetVersion on an Environment.
  Rollback is a new Deployment pointing at older versions. History is never rewritten.

## Architectural rules that span the codebase

- Talk to Docker only through the Go Docker SDK over the mounted socket. Never shell out to
  `docker compose`, because it cannot do blue-green.
- Update strategies: `deployment` defaults to blue-green (start new, wait for healthcheck,
  Traefik balances both, drain and stop old). `recreate` is opt-in for singletons.
  `stateful` is always `recreate` and must never run two containers on one volume.
- Deploy order follows `depends_on`: stateful services, then pre-deploy jobs (failure aborts and
  leaves running services untouched), then deployments, then post-deploy jobs.
  One deployment at a time per environment.
- Naming is owned by Dockerity: `dkt-<app>-<env>-<service>-<deploymentShortId>` for containers,
  `dkt-<app>-<env>` for networks, `dkt-<app>-<env>-<service>-<volume>` for volumes,
  `dkt/<app>/<service>:<commitSha>` for images.
- Every managed object carries labels `dockerity.app`, `dockerity.env`, `dockerity.service`,
  `dockerity.deployment`. Labels link Docker state to the database and drive the reconciler.
- Containers labeled `dockerity.system=true` (Dockerity itself and Traefik) must be excluded
  from every deploy, cleanup and garbage-collection operation.
- Stateful volumes belong to the environment, are created by Dockerity as external volumes,
  and survive redeploys. They are backed up before a stateful image change and before
  environment deletion.
- App routing is done by writing Traefik file-provider configuration (one file per
  environment, `internal/routing`), so Dockerity controls which containers receive traffic.
  No host port bindings for routed services.

## Secrets

- Built-in store behind a `SecretStore` interface, so OpenBao or Infisical can be added later.
- AES-256-GCM from the Go standard library with envelope encryption: per-environment and
  per-app data keys, wrapped by a master key read from `/run/secrets/master_key`.
  The master key is never stored in SQLite.
- Secret sets are versioned and immutable. API and UI are write-only (masked values).
- Secrets are injected at runtime as env vars or tmpfs files, never into images, build args or logs.
- Each app has a Dockerity-generated ed25519 deploy key. The private key lives in the secret store.

## Code organization

Planned layout is in section 12 of `docs/design.md`. Keep stable, pure logic (spec validation,
naming, deploy planning, Traefik label generation) in packages with no Docker or SQLite
dependency so it can be unit-tested in isolation. Docker access goes through `internal/dockerx`.

## Planning

Work is tracked as GitHub epics (label `epic`) that are broken into sub-issues during planning.
Priority labels, milestones and ordering rules are in `docs/prioritization.md`.

## Conventions

- Code, comments, commits and docs are in English.
- Conventional Commits, small and focused.
