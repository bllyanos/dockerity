# Dockerity: Design Document

Status: draft, based on the initial design discussion (2026-10-22).

## 1. Goal

Dockerity is an internal, containerized Docker wrapper for a **single VM** that runs
apps with non-trivial deployments. It owns three things:

1. **Build**: turn a repo commit into versioned images.
2. **Deploy and track**: always know which build runs in which environment, with history and rollback.
3. **Configuration**: app specs, env config and secrets live in Dockerity, not on laptops or in ad-hoc files on the server.

Spawning a new environment must be a single action.

### Non-goals (for now)

- Multiple nodes, Swarm or Kubernetes.
- Multi-repo apps (one app = one repo).
- Pulling prebuilt images from CI (planned after MVP).
- CLI (nice to have, after MVP).

## 2. Decisions so far

| Topic | Decision |
|---|---|
| Target | Single VM. Dockerity itself runs on the same VM, as a container. |
| Workload kinds | `job`, `deployment`, `stateful` |
| Update strategy | `deployment`: blue-green (zero downtime) by default, `recreate` opt-in for singletons. `stateful`: always `recreate`. |
| Stateful volumes | Named volumes owned by the environment, reused across deploys. |
| New environment | Volumes start empty. |
| Delete environment | Volumes are backed up by default, then deleted. |
| Stateful image change | Automatic volume backup before the change, plus a downtime warning in the UI. |
| Build | Local build on the VM for MVP. |
| Interface | Web UI + HTTP API. CLI later. |
| Secrets | Built in, encrypted at rest, behind a pluggable interface. |
| Environment types | `permanent` and `ephemeral` |
| Source | One app = one repo. An app has many services, built from the repo or from a public image. |
| Stack | Go backend, React + Tailwind frontend, SQLite on a Docker volume. |

## 3. Domain model

```
App 1──* AppSpecVersion
App 1──* Build 1──* BuildImage (one per built service)
App 1──* Environment
Environment 1──* SecretSetVersion
Environment 1──1 DataKey
Environment 1──* Volume 1──* Backup
Environment 1──* Deployment ──> Build, AppSpecVersion, SecretSetVersion
Deployment 1──* DeploymentStep
Deployment 1──* JobRun
User 1──* AuditEvent
```

| Entity | Purpose | Key fields |
|---|---|---|
| App | One repo, one deployable unit | name, repo_url, default_branch, deploy key (public key + encrypted private key), limits |
| AppSpecVersion | Versioned app spec held by Dockerity | app_id, version, yaml, created_by |
| Build | One build of one commit | app_id, ref, commit_sha, status, log |
| BuildImage | Image built for one service | build_id, service, image_tag, digest |
| Environment | A running copy of the app | app_id, name, type, domain, expires_at, status, overrides |
| Deployment | "Build X with spec Y and secrets Z on env E" | env_id, build_id, spec_version, secrets_version, status, triggered_by |
| DeploymentStep | One step of a deployment plan | deployment_id, order, kind, service, status, log |
| JobRun | One execution of a job | deployment_id, service, exit_code, log |
| SecretSetVersion | Immutable encrypted snapshot of an env's secrets | env_id, version, ciphertext, nonce |
| DataKey | Per-environment data key, encrypted by master key | env_id, wrapped_key, master_key_id |
| Volume | Named Docker volume of a stateful service | env_id, service, docker_name |
| Backup | Archive of a volume | volume_id, path, size, reason, created_at |
| AuditEvent | Who did what | actor, action, target, at |

**Rollback** is just a new Deployment that points at an older Build, AppSpecVersion
and SecretSetVersion. History is never rewritten.

## 4. App spec

The spec is held by Dockerity (versioned in SQLite), compose-like so it feels familiar.
Example with invented values:

```yaml
name: shop
services:
  mysql:
    kind: stateful
    image: mysql:8.4
    volumes:
      data: /var/lib/mysql
    env:
      MYSQL_DATABASE: shop
    secrets: [MYSQL_ROOT_PASSWORD]
    healthcheck:
      cmd: ["mysqladmin", "ping", "-h", "127.0.0.1"]

  migrate:
    kind: job
    phase: pre-deploy          # pre-deploy | post-deploy
    build:
      context: .
      dockerfile: Dockerfile
    command: ["./app", "migrate"]
    depends_on: [mysql]
    secrets: [DATABASE_URL]

  api:
    kind: deployment
    strategy: blue-green       # blue-green | recreate
    build:
      context: .
      dockerfile: Dockerfile
    command: ["./app", "serve"]
    depends_on: [mysql]
    secrets: [DATABASE_URL]
    http:
      port: 8080
      path: /api
    healthcheck:
      http: /healthz
    resources:
      memory: 512m
      cpus: 0.5

  scheduler:
    kind: deployment
    strategy: recreate         # must never run twice
    build:
      context: .
    command: ["./app", "cron"]
```

Environment-level overrides (replicas, resources, env values) are stored on the
Environment, not in the spec.

### Spec format reference

Top level: `name` and `services` (required). Service fields:

| Field | Type | Notes |
|---|---|---|
| `kind` | `job` \| `deployment` \| `stateful` | required |
| `image` | string | public image; exactly one of `image` / `build` |
| `build` | string (context) or mapping | `context`, `dockerfile`, `target`, `args` |
| `command` | list of strings | |
| `env` | mapping | plain, non-secret values |
| `secrets` | list | `NAME` (env var) or `{name, file}` (tmpfs file) |
| `volumes` | mapping name → path | stateful only |
| `depends_on` | list of service names | must form a DAG |
| `phase` | `pre-deploy` \| `post-deploy` | jobs only |
| `strategy` | `blue-green` \| `recreate` | deployments only |
| `http` | `port`, `path` | routed through Traefik |
| `healthcheck` | `cmd` (list) or `http` (path), `interval`, `timeout`, `start_period`, `retries` | |
| `resources` | `memory` (e.g. `512m`), `cpus` (e.g. `0.5`) | |
| `drain` | duration, default `5s` | how long a replaced container keeps running after leaving routing |
| `stop_grace_period` | duration, default `30s` | time between SIGTERM and SIGKILL |

Compose fields that Dockerity owns are rejected with a hint: `container_name`, `ports`,
`networks`, `network_mode`, `restart`, `deploy`. Unknown fields are errors. Every
problem is reported with its line and column.

### Spec validation rules

- `deployment` with `strategy: blue-green` must have a healthcheck and no host port binding.
- No `container_name`: Dockerity owns all names.
- `stateful` cannot use `blue-green`.
- `depends_on` must form a DAG.
- Every referenced secret must exist in the target environment before deploy.

## 5. Architecture

```
┌──────────────────────────── VM ────────────────────────────┐
│                                                            │
│  ┌──────────┐    ┌──────────────── dockerity ───────────┐  │
│  │ Traefik  │    │ HTTP API + embedded React UI         │  │
│  │ (system) │    │ Build runner   Deploy engine         │  │
│  └────┬─────┘    │ Secrets store  Reaper  Reconciler    │  │
│       │          │ SQLite (/data)                       │  │
│       │          └──────────┬───────────────────────────┘  │
│       │                     │ /var/run/docker.sock         │
│       ▼                     ▼                              │
│  ┌──────────────── Docker engine ───────────────────────┐  │
│  │ env shop-staging: mysql, api(blue), api(green), ...  │  │
│  │ env shop-pr-42:   mysql, api, ...                    │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

- Dockerity talks to Docker through the mounted socket, using the official Go Docker SDK.
  It does **not** shell out to `docker compose`, because compose cannot do blue-green.
- App traffic is routed by Traefik's **file provider**: Dockerity writes one dynamic
  configuration file per environment (routers by domain and path, and an explicit server
  list of container names on the environment network) into a volume shared with Traefik.
  Labels cannot change on a running container, so with labels an old blue-green container
  could not leave routing while it drains; with the file, Dockerity adds a container only
  once it is healthy and removes it before stopping it. Dockerity's own UI is still routed
  by labels from the bootstrap compose file.
- Dockerity and Traefik are **system containers**. They carry the label
  `dockerity.system=true` and are excluded from every deploy, cleanup and GC operation.

### Naming and labels

| Object | Name |
|---|---|
| Network | `dkt-<app>-<env>` |
| Container | `dkt-<app>-<env>-<service>-<deploymentShortId>` |
| Volume | `dkt-<app>-<env>-<service>-<volumeName>` |
| Image | `dkt/<app>/<service>:<commitSha>` |

Every managed object carries labels `dockerity.app`, `dockerity.env`,
`dockerity.service`, `dockerity.deployment`. Labels are the link between Docker state
and the database, and they drive the reconciler.

## 6. Flows

### 6.1 Build

1. Clone or fetch the repo into the workspace volume, check out the commit.
2. For every service with `build:`, build via the Docker API with BuildKit.
3. Tag as `dkt/<app>/<service>:<sha>`, store the digest in BuildImage.
4. Stream logs to the UI, persist them on the Build.

Secrets are never passed as build args. If a build needs a secret, use BuildKit secret mounts (later).

**BuildKit through the Engine API (spike #65).** `POST /build?version=2` with a tar
context works without a buildx session (Docker 29, Go SDK `ImageBuild` with
`Version: BuilderBuildKit`). Findings:

- Progress arrives as JSON messages `{"id":"moby.buildkit.trace","aux":"<base64>"}`. The
  payload is a BuildKit `StatusResponse` protobuf: field 1 vertexes (digest, name,
  cached, started, completed, error), field 3 logs (vertex, stream, msg). Dockerity
  decodes just these fields with `google.golang.org/protobuf/encoding/protowire`
  instead of importing the whole BuildKit module.
- The result is `{"id":"moby.image.id","aux":{"ID":"sha256:..."}}`. Failures arrive as
  `error` / `errorDetail` messages.
- Rebuilding the same context reuses the cache: same image ID, about 0.6 s instead of
  5.4 s for the first build.
- The classic builder (`version=1`) still works and prints readable `stream` lines,
  but it is deprecated, so it is not used.
- Session features (secret mounts, SSH forwarding) would need a BuildKit session; out of
  scope for now.
- Without a session, BuildKit may fail to resolve a base image that is not present locally
  ("failed to resolve source metadata … no active sessions"). It worked on Docker 29 with
  the containerd image store but failed on the CI runner. Dockerity therefore pulls every
  `FROM` image through the Engine API before building (`build.BaseImages` parses the
  Dockerfile, substituting global `ARG`s; `dockerx.PullImage` pulls).

### 6.1.1 Git access with deploy keys

Each app has its own deploy key:

1. On app creation, Dockerity generates an ed25519 key pair.
2. The UI shows the public key, and the user adds it as a read-only deploy key on the repo.
3. The private key is encrypted with the secret store (app-level data key) and only
   decrypted in memory during clone or fetch.
4. The key can be regenerated from the UI, which invalidates the old one.

### 6.2 Deploy

A deployment is planned first, then executed step by step. One deployment at a time
per environment (environment lock).

1. **Plan**: resolve Build + AppSpecVersion + SecretSetVersion, compute the step list in
   `depends_on` order, detect stateful changes and downtime.
2. **Prepare**: ensure network and volumes exist.
3. **Stateful services**: for each changed one, back up if the image changed, stop old,
   start new on the same volume, wait for healthy.
4. **Pre-deploy jobs**: run to completion. A non-zero exit aborts the deployment and leaves
   running services untouched.
5. **Deployments**: blue-green or recreate, see below.
6. **Post-deploy jobs**.
7. **Finish**: mark the deployment as current, remove old containers.

#### Blue-green for one service

1. Start the new container next to the old one, same Traefik labels.
2. Wait for the healthcheck. On failure, remove the new container and abort.
3. Traefik now balances across both.
4. Remove the old container from routing (Dockerity rewrites the environment's Traefik
   file), wait for the `drain` period, send SIGTERM, then remove it after the
   `stop_grace_period`.

If the new container fails its health check it is removed before it is ever routed, and
the old one keeps serving; the deployment fails.

Requirements on the app: handle SIGTERM gracefully, and keep migrations backward
compatible (expand then contract), because old and new versions run together briefly.

#### Recreate

Stop old, start new, wait for healthy. Causes downtime. Used for stateful services and
singleton deployments such as schedulers.

### 6.3 Spawn environment

1. Create Environment (type, name, domain, TTL for ephemeral).
2. Copy secrets from a template environment or enter them fresh.
3. Deploy a chosen Build. Volumes start empty.

### 6.4 Delete environment

1. Stop and remove containers and network.
2. Back up each volume (default on, can be skipped explicitly).
3. Remove volumes.

### 6.5 Reconciler

Runs on startup and every 5 minutes. Compares Docker objects (by label) to the database.
Deployments interrupted by a crash are marked failed on startup. For each environment
without a deployment in progress it reports drift (`GET /environments/{id}/drift`):

- `missing` / `stopped`: a long-running service of the current deployment has no running
  container (e.g. removed by hand). Reported only; a redeploy recreates it.
- `extra`: a running container the current deployment does not expect. Reported only.
- `dangling_route`: the route file names a container that does not run.

Safe repairs: routes are rewritten from what runs, and stopped containers nothing expects
are removed. Running containers and system containers are never touched.

## 7. Environments

| | Permanent | Ephemeral |
|---|---|---|
| Examples | staging, production | per branch or feature |
| Domain | Fixed, configured | `<env>-<app>.<baseDomain>`, automatic |
| Lifetime | Until deleted | TTL, extended on deploy, deleted by the reaper |
| Deploy | Deliberate | Can be triggered by API from CI |
| Limits | User-managed | User-managed |

Ephemeral routing uses a **wildcard DNS record** (`*.<baseDomain>`) pointing to the VM,
plus a **wildcard TLS certificate** (`*.<baseDomain>`) served by Traefik. The base domain
is set in Dockerity's settings. Ephemeral hosts are `<env>-<app>.<baseDomain>`, one label
below the base domain, because a TLS wildcard only matches a single label.

Certificates (settings, TLS mode):

- `none`: plain HTTP.
- `certificate`: Traefik serves a wildcard certificate from files (an internal CA, or a
  certificate obtained outside Dockerity). Dockerity writes `dkt-tls.yml` making it the
  default certificate.
- `resolver`: a Traefik certificate resolver configured in the bootstrap compose (for
  example Let's Encrypt with a DNS challenge) obtains `*.<baseDomain>` for ephemeral hosts
  and per-host certificates for permanent domains.

With TLS on, routes listen on `websecure` and HTTP redirects to HTTPS.

### Limits

Limits are **user-managed**, with no built-in defaults. Users configure them in the UI:

- Per app: max active ephemeral environments.
- Per environment or service: memory and CPU limits.

When a limit is not set, it is not enforced.

## 8. Secrets

Built in, behind a `SecretStore` interface so OpenBao or Infisical can be added as
backends later.

- **Primitives**: AES-256-GCM from the Go standard library. No custom crypto.
- **Envelope encryption**: each environment has a random data key, wrapped by the
  master key. Each app also has one, for app-level secrets such as the deploy key.
  Rotating the master key only rewraps data keys.
- **Master key**: read from a file mounted as a Docker secret (`/run/secrets/master_key`),
  never stored in SQLite. It must be backed up separately, for example in the team
  password manager. Losing it means losing all secrets.
- **Versioned**: every change creates a new immutable SecretSetVersion. Deployments
  reference a version, so rollback restores the secrets used at that time.
- **Write-only**: API and UI show masked values. "Reveal", if added, is audited.
- **Injection**: as env vars or as files on tmpfs. Never in images, build logs or stored config.
- **Known limit**: root on the VM can read secrets. Accepted for an internal single-VM tool.

## 9. Backups

- Generic volume backup for MVP: stop the container, archive the volume with a helper
  container into the backups volume, start again.
- Triggered by: stateful image change, environment deletion, manual action.
- Retention policy per environment (keep last N).
- Later: per-service custom backup command (for example `mysqldump --single-transaction`)
  for near-zero downtime.

## 10. Downtime warning in the UI

Shown before any deployment whose plan contains a `recreate` step:

- Which services will stop, including dependents (for example `api` depends on `mysql`).
- Whether a backup will be taken, and when the last backup was.
- Image version change, with stronger emphasis on a major version change.

## 11. Security

- Docker socket access is root-equivalent, so the UI and API always require login.
- **Local users only** for now: password hashed with argon2id, session cookie for the UI,
  API tokens for automation. SSO is out of scope.
- The first admin user is created on first startup (setup screen).
- Sessions: opaque token in an `HttpOnly`, `SameSite=Lax` cookie (`Secure` behind HTTPS),
  only its SHA-256 digest is stored, sliding 7-day idle expiry. CSRF is handled by Go's
  `http.CrossOriginProtection` (Sec-Fetch-Site / Origin checks). Failed logins are rate
  limited per username and per client IP.
- Every `/api/v1` route requires authentication unless explicitly registered as public
  (setup and login only).
- Audit log for deploys, secret changes, environment create and delete.

## 12. Code layout

```
dockerity/
├── cmd/dockerity/            # main: wire config, store, services, HTTP server
├── internal/
│   ├── domain/               # entities and pure rules (spec validation, naming)
│   ├── spec/                 # app spec parsing and validation
│   ├── store/sqlite/         # repositories, migrations
│   ├── dockerx/              # thin adapter over the Docker SDK
│   ├── build/                # git checkout + image build
│   ├── deploy/
│   │   ├── planner/          # spec + current state -> step list
│   │   ├── executor/         # runs steps, records progress
│   │   └── strategy/         # bluegreen, recreate, job
│   ├── routing/              # Traefik label generation
│   ├── secrets/              # SecretStore interface, builtin AES-GCM store
│   ├── backup/
│   ├── reaper/               # ephemeral env TTL
│   ├── reconcile/
│   ├── auth/
│   └── api/                  # HTTP handlers, SSE log streaming
├── web/                      # React + Tailwind (Vite), embedded via go:embed
├── deploy/
│   └── docker-compose.yml    # bootstrap: traefik + dockerity
├── Dockerfile                # multi-stage: web build, go build, small runtime
└── docs/
```

Notes:

- SQLite via `modernc.org/sqlite` (pure Go, no cgo), WAL mode.
- One Docker image contains the API and the built UI.
- Stable, pure logic (spec validation, naming, planning, label generation) lives in
  packages with no Docker or SQLite dependency, so it is easy to test.

## 13. Bootstrap

`deploy/docker-compose.yml` is the only thing started by hand:

- `traefik`: ports 80/443, Docker provider, label `dockerity.system=true`.
- `dockerity`: mounts `/var/run/docker.sock`, volume `dockerity-data:/data`
  (SQLite, workspace), volume `dockerity-backups:/backups`, Docker secret `master_key`,
  label `dockerity.system=true`.

Updating Dockerity itself is done manually with this compose file, outside Dockerity.

## 14. MVP milestones

1. **Skeleton**: Go service, SQLite migrations, auth, embedded UI shell, bootstrap compose.
2. **Apps and specs**: create app, edit and version spec, validation.
3. **Build**: local build with live logs.
4. **Environments and secrets**: create permanent env, builtin secret store.
5. **Deploy**: planner, recreate strategy, jobs, deployment history.
6. **Zero downtime**: blue-green with Traefik.
7. **Stateful**: volumes, backups, downtime warning.
8. **Ephemeral**: auto subdomain, TTL reaper, limits.
9. **Rollback and reconciler**.

## 15. Open questions

- Log retention for running containers (today bounded only by Docker's log driver settings).

### Resolved

| Question | Decision |
|---|---|
| Login method | Local users only for now |
| Ephemeral domain and TLS | Wildcard DNS + wildcard certificate, hosts `<env>-<app>.<baseDomain>` |
| Wildcard certificate source | Either: certificate files (internal CA or external), or a Traefik resolver (e.g. Let's Encrypt DNS challenge), chosen in the settings |
| Git authentication | Deploy key per app, generated by Dockerity |
| Resource limits and max ephemeral envs | User-managed, no built-in defaults |
| Log retention and cleanup | Build and deployment logs (job output included) are deleted after `DOCKERITY_LOG_RETENTION_DAYS` (default 30, 0 keeps them); records stay and show "log expired". Hourly cleanup also removes workspaces of deleted apps and prunes the BuildKit cache to `DOCKERITY_BUILD_CACHE_GB` (default 20, 0 disables) |
| Image garbage collection | Per app, keep images of the last 3 builds, the last 3 succeeded deployments per environment, deployments in progress, and anything a container uses; other `dkt/<app>/*` images are removed after each successful deployment and on the reconciler schedule. A rollback further back needs a rebuild |
