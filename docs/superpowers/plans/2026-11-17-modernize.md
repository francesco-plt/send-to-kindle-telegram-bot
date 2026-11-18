# Modernize SendToKindle Bot — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use the execution method the
> human already supplied — one feature branch per task, open a PR against the
> fork's `master`, wait for Greptile review, address comments, merge. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the frozen 2019–2021 bot to current Go, TLS SMTP, EPUB output,
and a sane container, without changing user-visible behavior.

**Architecture:** Same single-package bot (`bot` + `main`). A1 fixes
correctness + output format with no dependency change; A2 swaps the two dead
deps and adds TLS; B rewrites Docker/CI (file-disjoint, parallel-safe); C adds
hardening. Each task is its own branch + PR.

**Tech Stack:** Go 1.26, `gopkg.in/telebot.v3`, `github.com/wneessen/go-mail`,
`log/slog`, Calibre `ebook-convert`.

**Spec:** `docs/superpowers/specs/2026-11-17-modernize-design.md`

## Global Constraints

- Go module version `go 1.26` (set in A2; A1 keeps `go 1.17` to stay dependency-clean).
- Env vars unchanged: `UBOT_TELEGRAM_TOKEN`, `UBOT_EMAIL_FROM`, `UBOT_EMAIL_TO`,
  `UBOT_SMTP_HOST`, `UBOT_PASSWORD`. New: `UBOT_SMTP_PORT` (default `587`).
- Output is EPUB only (`.mobi` dropped).
- Never push to upstream `michaelfmnk`; PRs target fork `master`.
- `supportedFormats` final list: `doc docx rtf htm html txt epub pdf`.

## Review Focus

Most likely inputs to bite a real user, and where each is pinned:

1. Filename with no extension (`README`) → must not mangle base or panic. *(A1 `TestSplitFileName`)*
2. Uppercase extension (`FILE.PDF`) → treated as supported (case-insensitive). *(A1 `TestSplitFileName` + `TestNeedToConvert`)*
3. Multi-dot filename (`book.part1.pdf`) → only last extension stripped; base keeps dots. *(A1 `TestSplitFileName`)*
4. `UBOT_SMTP_PORT` empty → `587`. *(A1 `TestVerifyConfigDefaults`)*
5. `ebook-convert` fails or missing → error surfaces, no email sent. *(A1 `TestConvert` — A2/C may widen)*

---

## Task A1 — Correctness + EPUB (branch `modernize/a1-correctness`)

No dependency change; `go.mod` stays `go 1.17`.

**Files:**
- Modify: `bot/bot.go`
- Modify: `main.go`
- Modify: `bot/bot_test.go`

**Interfaces:**
- Produces: `func splitFileName(name string) (base, ext string)` — new helper;
  `ext` is lowercased without leading dot, `base` keeps everything before it.
- Produces: `needToConvert(extension string) bool` — unchanged signature, now
  tested.

- [ ] **Step 1: Write failing tests in `bot/bot_test.go`**

  - `TestSplitFileName`: cases — `"book.pdf"` → `("book","pdf")`;
    `"book.part1.pdf"` → `("book.part1","pdf")`; `"README"` → `("README","")`;
    `"FILE.PDF"` → `("FILE","pdf")`; `".hidden"` → `("","hidden")`.
  - `TestNeedToConvert`: `"pdf"`/`"epub"`/`"mobi"` (mobi now unsupported) →
    `true` conversion for `"mobi"`, `false` for `"pdf"`/`"epub"`; `"xlsx"`/`""` → `true`.
  - `TestVerifyConfigDefaults`: config with all fields set except `SMTPPort`
    empty → `verifyConfig()` returns nil and sets `SMTPPort == "587"`.

- [ ] **Step 2: Run tests to see them fail**

  Run: `go test ./bot/ -run 'TestSplitFileName|TestNeedToConvert|TestVerifyConfigDefaults' -v`
  Expected: compile error (`splitFileName` undefined).

- [ ] **Step 3: Implement `splitFileName` and use it in `documentHandler` in `bot/bot.go`**

  `splitFileName(name string) (base, ext string)` using `filepath.Ext` +
  `strings.TrimSuffix` + `strings.TrimPrefix` + `strings.ToLower`. Replace the
  `strings.Split`/`Join` block in `documentHandler` with `splitFileName`.

- [ ] **Step 4: Fix `documentHandler` early-returns**

  After each of download / convert / send, on error log + respond + `return`
  (stop double-respond and sending a broken file).

- [ ] **Step 5: Fix `convert` double-wait in `bot/bot.go`**

  Replace `cmd.Run()` then `cmd.Wait()` with a single `cmd.Run()` (or
  `CombinedOutput` and include stderr in the error). Keep the `os.Stat(out)`
  existence check.

- [ ] **Step 6: Switch MOBI → EPUB**

  In `documentHandler`: output `fileNameWithoutExtension + ".epub"`. Update
  `supportedFormats` to `{"doc","docx","rtf","htm","html","txt","epub","pdf"}`.

- [ ] **Step 7: Wire `UBOT_SMTP_PORT` in `main.go`**

  Add `SMTPPort: os.Getenv("UBOT_SMTP_PORT")` to the struct literal.

- [ ] **Step 8: Run full tests**

  Run: `go test ./...`
  Expected: PASS (existing `TestUnkindleBot_verifyConfig` + new tests).

- [ ] **Step 9: Commit, push, open PR, wait for Greptile**

  ```bash
  git add -A && git commit -m "fix: correctness bugs + EPUB output + SMTP port"
  git push -u origin modernize/a1-correctness
  gh pr create --base master --head modernize/a1-correctness --title "A1: correctness + EPUB" --body "..."
  ```
  Poll `gh pr view` / comments for Greptile; address; merge.

---

## Task A2 — Library refresh + TLS (branch `modernize/a2-libs-tls`)

**Files:**
- Modify: `bot/bot.go` (imports, `sendFileViaEmail`, `Start`, logging)
- Modify: `bot/bot_test.go`
- Modify: `go.mod`, `go.sum` (`go 1.26`, `go mod tidy`)
- Modify: `main.go` (logging only if touched)

**Interfaces:**
- Consumes: `splitFileName`, `needToConvert` from A1.
- Produces: `sendFileViaEmail(path string) error` now via go-mail; `Start()`
  uses `slog`.

- [ ] **Step 1: Swap `telebot.v2` → `telebot.v3`**

  Import `gopkg.in/telebot.v3` as `tb`. Update `tb.Settings`, `tb.LongPoller`,
  `bot.Download`, `bot.Send` to v3 signatures (mostly source-compatible).

- [ ] **Step 2: Replace `scorredoira/email` with `github.com/wneessen/go-mail`**

  Rewrite `sendFileViaEmail` using go-mail: `NewMsg`, `From`, `To`, `AttachFile`,
  `NewClient` with `PlainAuth` and mandatory TLS (STARTTLS on `587`, implicit
  SSL on `465`). This closes the plaintext-credentials hole.

- [ ] **Step 3: `log` → `log/slog`**

  Replace `log.Println` / `log.Fatal` with `slog.Error`/`slog.Warn` +
  `os.Exit(1)` where fatal. Keep messages terse.

- [ ] **Step 4: Bump `go.mod` to `go 1.26` and `go mod tidy`**

  Run: `go get gopkg.in/telebot.v3@latest github.com/wneessen/go-mail@latest`
  then `go mod tidy`.

- [ ] **Step 5: Update `bot_test.go` for new APIs, run tests**

  Run: `go test ./...` → PASS.

- [ ] **Step 6: Commit, push, open PR, wait for Greptile, merge**

---

## Task B — Container + CI (branch `modernize/b-container-ci`, parallel-safe)

**Files:**
- Modify: `Dockerfile`
- Modify: `.github/workflows/go.yml`
- Modify: `.github/workflows/docker-image.yml`

- [ ] **Step 1: Rewrite `Dockerfile` (multi-stage)**

  Stage 1 `golang:1.26` builds the binary. Stage 2 Calibre runtime (install
  `calibre` via its installer or a calibre base image), copy binary, non-root
  user, `/files` writable. Remove `GOFLAGS="-insecure"`, `golang:1.16` copy,
  `python`, `ffmpeg`, `libsm6`, `libxext6`.

- [ ] **Step 2: Update `go.yml`**

  `actions/checkout@v4`, `actions/setup-go@v5` with `go-version: ['1.26','1.27']`
  matrix, add `golangci/golangci-lint-action@v6`, keep build + test.

- [ ] **Step 3: Update `docker-image.yml`**

  `docker/login-action@v3` + `docker/build-push-action@v6` (build + push).

- [ ] **Step 4: Commit, push, open PR, wait for Greptile, merge**

---

## Task C — Hardening (branch `modernize/c-hardening`, after A2)

- [ ] **Step 1:** Graceful shutdown — `signal.NotifyContext` + `bot.Stop()` on
  SIGINT/SIGTERM in `Start`.
- [ ] **Step 2:** Context/timeouts on `bot.Download` and email send.
- [ ] **Step 3:** (Optional) webhook support behind an env flag.
- [ ] **Step 4:** Broader tests — mock SMTP, injected converter.
- [ ] **Step 5:** Commit, push, open PR, wait for Greptile, merge.
