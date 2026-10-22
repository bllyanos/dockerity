# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

Dockerity is in the design phase. The only source of truth so far is `docs/design.md`.
Read it before implementing anything, and update it when a design decision changes.
No code, build, lint or test commands exist yet. Add them here once the skeleton
(milestone 1 in the design doc) is in place.

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
- Routing is done by generating Traefik labels. No host port bindings for routed services.

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

## Conventions

- Code, comments, commits and docs are in English.
- Conventional Commits, small and focused.
