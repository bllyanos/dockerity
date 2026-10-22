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

## Stack

Go, React + Tailwind, SQLite. Dockerity runs as a container next to Traefik on the VM it manages.
