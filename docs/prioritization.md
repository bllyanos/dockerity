# Prioritization Guide

How to decide what to work on next. Use this when breaking an epic into sub-issues
and when picking the next issue to work on.

## Priority labels

Every issue gets exactly one priority label.

| Label | Meaning | Examples |
|---|---|---|
| `priority: P0` | Blocks other work, or is on the critical path of the current milestone. Do first. | Migration runner, deploy planner, master key loading |
| `priority: P1` | Needed for the current milestone, but nothing waits on it. | Spec diff view, build cancel |
| `priority: P2` | Valuable, but the milestone ships without it. | Drift view, retention settings UI |
| `priority: P3` | Nice to have. Pick up only when nothing above is open. | CLI, UI polish |

Bugs follow the same scale. A bug that corrupts data, leaks secrets, or touches
system containers is always `P0`.

## Milestones

Work is grouped into milestones that each end in something usable.

| Milestone | Epics | Outcome |
|---|---|---|
| M1: First deploy | #1 to #6 | Build an app from Git and deploy it to a permanent environment from the UI |
| M2: Core MVP | #7, #8, #9 | Zero-downtime deploys, stateful services with backups, one-click ephemeral environments |
| M3: Operations | #10 | Rollback, crash recovery, cleanup |

Finish the current milestone before starting issues from the next one, unless
the issue is a spike that reduces risk for the next milestone.

## How to order work

When two issues have the same priority, apply these rules in order:

1. **Unblock others first.** Prefer the issue that the most open issues depend on.
2. **Thin vertical slice early.** Get one path working end to end (for example, deploy one
   stateless service) before widening any single layer.
3. **Risk first.** Unknowns get a time-boxed spike issue early, before the work that depends
   on them. Known risky areas: blue-green switching with Traefik, BuildKit through the Docker
   API, SQLite under concurrent writes.
4. **Stable before volatile.** Pure logic (spec validation, naming, planning, label generation)
   comes before the Docker, SQLite, API and UI wiring that uses it.
5. **Security is not deferred.** Auth and secret handling must be in place before Dockerity is
   reachable by anyone other than the developer.
6. **Smaller first.** Between otherwise equal issues, pick the one that closes sooner.

## Assigning priority during planning

When breaking an epic into sub-issues:

1. Mark the minimal set of sub-issues that delivers the epic's acceptance criteria. Those are
   `P0` if other issues wait on them, otherwise `P1`.
2. Everything else in the epic is `P2` or `P3`. Consider moving it to a follow-up issue instead
   of keeping it in the epic.
3. Record dependencies between sub-issues ("Blocked by #N") so rule 1 above can be applied.
4. Re-check priorities when a milestone closes. A `P2` from an earlier milestone may become
   `P0` for the next one.
