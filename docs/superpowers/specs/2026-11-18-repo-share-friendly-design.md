# Repo Share-Friendly Polish — Design

**Date:** 2026-11-18
**Scope:** repository presentation + community/automation scaffolding (no product-code changes).

## Intent & why

Make the fork presentable and community-ready so a stranger can understand,
run, and contribute to it in a few minutes: an attractive README with badges,
contribution guidelines, automated dependency updates, package builds
(goreleaser + GHCR), and CI with coverage.

## Goals

- README rewrite: hero + badge row, features, quickstart, full env table, "how it works".
- `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` (Contributor Covenant), `SECURITY.md`.
- Issue templates (bug, feature) + PR template.
- `docker-compose.yml` for one-command local run.
- Dependabot on gomod + Docker + GitHub Actions (grouped, auto-rebase).
- Goreleaser: tag-triggered → cross-platform binaries + GHCR image + GitHub Release.
- CI: add a Codecov coverage step to `go.yml`; add a release workflow.
- Repo metadata (`description`, `topics`, `homepage`) via `gh repo edit`.

## Non-goals

- No product/code changes, no new features, no refactors.
- No webhook mode (already decided against).
- No Renovate (chose Dependabot).
- No branch-protection rules.

## Anti-goals

- Don't break existing CI (`go.yml` matrix 1.26+1.27 + golangci-lint; `docker-image.yml` push).
- Don't drop or rewrite the MIT license / original attribution (`© Mykhailo Fomenko`).
- Don't introduce doc/README inaccuracies about env vars or behavior.

## Approaches considered

1. **Docker-only distribution** — simpler, matches how the bot actually runs (needs Calibre). Rejected: user wants package builds.
2. **Goreleaser + GHCR** — binaries + image + Releases from one tag. **Chosen.**
3. **Renovate** — more configurable dependency updates. Rejected: Dependabot is GitHub-native and sufficient.

## Decisions (user-confirmed)

1. Goreleaser + GHCR image.
2. Dependabot.
3. Codecov (app installed on the account).

## My calls (proposals — confirm or override)

- README: replace "message Michael at michael@fomenko.dev" with "open a GitHub Issue". Keep the Dev.to link for attribution.
- Fix README env table: add `UBOT_SMTP_PORT` and `PORT` (healthz).
- Keep the MIT license header as-is (original author attribution preserved).

## What's left to the builder

Exact badge URLs, goreleaser config, dependabot schedule, docker-compose env wiring — all standard conventions.
