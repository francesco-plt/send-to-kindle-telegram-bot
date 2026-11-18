# Repo Share-Friendly Polish — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Make the fork presentable and community-ready: polished README + badges, contribution/community docs, automated dependency updates, package builds (goreleaser binaries + GitHub Releases; GHCR image already exists), and CI coverage.

**Architecture:** Three file-disjoint work units (docs, release+coverage, dependabot) + one repo-metadata action. No product-code changes.

**Tech Stack:** Markdown, GitHub issue forms, GitHub Actions, goreleaser, Dependabot, Codecov, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-11-18-repo-share-friendly-design.md`

## Global Constraints

- Repo/module: `github.com/francesco-plt/send-to-kindle-telegram-bot`. Default branch is **`master`** (not `main`).
- GHCR image: `ghcr.io/francesco-plt/sendtokindle` (already built by the existing `docker-image.yml`; do not touch it).
- Go floor 1.26; CI matrix `['1.26','1.27']` (already in `go.yml`; preserve it).
- Canonical env vars (exact names): `UBOT_TELEGRAM_TOKEN`, `UBOT_EMAIL_FROM`, `UBOT_EMAIL_TO`, `UBOT_SMTP_HOST`, `UBOT_SMTP_PORT`, `UBOT_PASSWORD`, and optional `PORT` (healthz).
- **Do NOT modify:** `main.go`, `bot/*.go`, `go.mod`, `go.sum`, `Dockerfile`, `LICENSE`, `.github/workflows/docker-image.yml`, anything under `docs/superpowers/`.
- LICENSE header (`Copyright 2023 Mykhailo Fomenko`) stays unchanged.
- README must NOT reference `michaelfmnk` except the Dev.to link (line 31); replace the `michael@fomenko.dev` contact with a GitHub Issues link.
- All YAML must parse (`python3 -c 'import yaml,sys; yaml.safe_load(open(sys.argv[1]))' <file>`).

## Review Focus

The five input/failure classes most likely to bite a self-hoster, each pinned by a check in the owning task:

1. **README env table incomplete/inaccurate** — must list all 7 canonical vars. Pin: `grep -c` each var name in README == 1 (task 1).
2. **docker-compose missing a var or the volume** — must pass all vars, mount `./files:/files`, expose `PORT`. Pin: `docker compose config` or YAML parse + grep for each var and `/files` (task 1).
3. **Goreleaser config builds for a wrong arch/go** — Pin: `goreleaser check` (or YAML parse if goreleaser absent) and grep for `amd64`/`arm64` (task 2).
4. **Release/coverage workflow breaks the existing `go.yml` matrix/lint** — Pin: preserve `checkout@v4`, `setup-go@v5`, matrix, `golangci-lint-action` keys; YAML parse (task 2).
5. **Badges/links point at upstream `michaelfmnk` instead of the fork** — Pin: `grep -i michaelfmnk README.md` returns only the Dev.to line (task 1).

---

### Task 1: Community docs + README rewrite

**Files:**
- Modify: `README.md`
- Create: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `docker-compose.yml`
- Create: `.github/ISSUE_TEMPLATE/bug_report.yml`, `.github/ISSUE_TEMPLATE/feature_request.yml`, `.github/PULL_REQUEST_TEMPLATE.md`

**Interfaces:**
- Produces: final README badge row + env table + quickstart that the release/coverage badges (task 2) will light up; the exact badge URLs below must be used verbatim.

- [ ] **Step 1: Rewrite `README.md`** with: hero title + one-line tagline; badge row using these EXACT URLs (verbatim):
  - CI: `https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/go.yml/badge.svg?branch=master`
  - Coverage: `https://codecov.io/gh/francesco-plt/send-to-kindle-telegram-bot/branch/master/graph/badge.svg`
  - Release: `https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/release.yml/badge.svg`
  - Docker: `https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/docker-image.yml/badge.svg?branch=master`
  - Go Report: `https://goreportcard.com/badge/github.com/francesco-plt/send-to-kindle-telegram-bot`
  - Go version: `https://img.shields.io/github/go-mod/go-version/francesco-plt/send-to-kindle-telegram-bot`
  - License: `https://img.shields.io/github/license/francesco-plt/send-to-kindle-telegram-bot`
  Then: Features; Quickstart (`docker compose up --build`); env table with ALL 7 canonical vars (Token/From/To/Host/Port/Password + `PORT` optional); "How it works" (Telegram doc → Calibre `ebook-convert` → EPUB → email to Kindle); Troubleshooting. Replace the `michael@fomenko.dev` line with an Issues link; KEEP the Dev.to link.

- [ ] **Step 2: Create `CONTRIBUTING.md`** — prerequisites (Go 1.26+, Docker optional), `go build ./...` / `go test ./...` / `golangci-lint run`, commit-message convention, PR flow, and a note that Greptile auto-reviews PRs.

- [ ] **Step 3: Create `CODE_OF_CONDUCT.md`** — Contributor Covenant 2.1 (contact: a GitHub Issue), and `SECURITY.md` (how to report vulnerabilities → GitHub Security Advisory or an Issue).

- [ ] **Step 4: Create issue forms + PR template** — `bug_report.yml` (fields: description, repro, expected, env, logs), `feature_request.yml` (problem, solution, alternatives), `PULL_REQUEST_TEMPLATE.md` (what/why/test checklist).

- [ ] **Step 5: Create `docker-compose.yml`** — service `bot` with `build: .`, env for all 7 vars (with `${VAR}` placeholders), `ports: ["${PORT:-8080}:8080"]`, `volumes: ["./files:/files"]`, `restart: unless-stopped`.

- [ ] **Step 6: Verify** — `grep -ci michaelfmnk README.md` == 1 (dev.to only); `grep -c` each of the 7 env-var names in `README.md`; YAML-parse both issue forms and `docker-compose.yml`; `docker compose config` if the daemon is up (it is NOT on this machine — parse only).

- [ ] **Step 7: Commit + push + open PR** against `master` using explicit `--repo francesco-plt/send-to-kindle-telegram-bot`. Do NOT merge.

### Task 2: Release + coverage automation

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/release.yml`
- Modify: `.github/workflows/go.yml` (add coverage + Codecov upload)

**Interfaces:**
- Consumes: repo name, image name, go floor (Global Constraints).
- Produces: `release.yml` workflow + `coverage.out`/Codecov badge that task 1's README already links.

- [ ] **Step 1: Create `.goreleaser.yaml`** — `project_name: send-to-kindle-telegram-bot`; `builds`: one build, binary `bot`, `goos: [linux, darwin, windows]`, `goarch: [amd64, arm64]`; `archives`: `format: tar.gz`; `release`: `github` with `draft: false`. NO `dockers` section (image stays with `docker-image.yml`).

- [ ] **Step 2: Create `.github/workflows/release.yml`** — trigger `push: tags: ['v*']`; `checkout@v4`; `setup-go@v5` (go-version from `go.mod`); `goreleaser/goreleaser-action@v6` with `GITHUB_TOKEN`; permissions `contents: write`.

- [ ] **Step 3: Modify `.github/workflows/go.yml`** — add a step after the existing `go test` step: `go test ./... -coverprofile=coverage.out` then `codecov/codecov-action@v5` with `files: coverage.out`. Preserve the existing `checkout@v4`, `setup-go@v5`, matrix, and `golangci-lint-action` steps verbatim.

- [ ] **Step 4: Verify** — `goreleaser check` (or YAML parse if goreleaser absent); YAML-parse `release.yml` and `go.yml`; confirm `go.yml` still has `checkout@v4`, `setup-go@v5`, matrix `['1.26','1.27']`, `golangci-lint-action`.

- [ ] **Step 5: Commit + push + open PR** against `master` with explicit `--repo`. Do NOT merge.

### Task 3: Dependabot config

**Files:**
- Create: `.github/dependabot.yml`

- [ ] **Step 1: Create `.github/dependabot.yml`** — `version: 2`; three `updates`: `gomod` (`directory: /`, weekly, `groups` for all deps), `docker` (`directory: /`, weekly), `github-actions` (`directory: /.github/workflows`, weekly). Target branch is the default (`master`) — do not set a `target-branch` key.

- [ ] **Step 2: Verify** — YAML-parse; confirm three `package-ecosystem` values (`gomod`, `docker`, `github-actions`).

- [ ] **Step 3: Commit + push + open PR** against `master` with explicit `--repo`. Do NOT merge.

---

### Repo metadata (orchestrator, no branch)

- [ ] `gh repo edit francesco-plt/send-to-kindle-telegram-bot --description "Telegram bot that converts documents and emails them to your Kindle (Calibre + TLS SMTP)." --add-topic telegram,kindle,calibre,golang,ebook,telegram-bot,docker`
