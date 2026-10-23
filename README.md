# Dockerity

Internal Docker wrapper for running complex deployments on a single VM.

Dockerity builds apps from Git, deploys them into environments, tracks which build runs
where, and keeps app config and secrets in one place. Spawning a new environment is a
single action.

> **Status:** design phase. See [`docs/design.md`](docs/design.md).

## Features (planned)

- **Build** images locally from a repo commit.
- **Deploy** with zero downtime (blue-green via Traefik) for stateless services.
- **Jobs** such as migrations, run before or after a deploy.
- **Stateful services** such as MySQL, with persistent volumes and automatic backups.
- **Environments**: permanent (staging, production) and ephemeral (auto subdomain, TTL).
- **History and rollback** of every deployment.
- **Secrets** encrypted at rest, versioned with each deployment.
- **Web UI and HTTP API.**

## Bootstrap

On the VM, from `deploy/`:

```sh
openssl rand -base64 32 > master_key && chmod 600 master_key
DOCKERITY_HOST=dockerity.example.com docker compose up -d --build
```

This starts Traefik (ports 80/443) and Dockerity behind it. `DOCKERITY_HTTP_PORT`,
`DOCKERITY_HTTPS_PORT` and `DOCKERITY_IMAGE` override the defaults.

Back up `master_key` outside the VM (for example in your password manager). Losing it
means losing every secret stored in Dockerity.

### Rotating the master key

With Dockerity stopped:

```sh
openssl rand -base64 32 > master_key.new && chmod 600 master_key.new
docker compose run --rm -v "$PWD/master_key.new:/run/new_key:ro" dockerity \
  rotate-master-key --new-key-file /run/new_key
mv master_key.new master_key && docker compose up -d dockerity
```

## Stack

Go, React + Tailwind, SQLite. Dockerity runs as a container next to Traefik on the VM it manages.
