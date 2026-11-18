# Modernize SendToKindle Telegram Bot — Design & Staged Plan

Date: 2026-11-17
Branch: `modernize/2026`
Status: awaiting approval

## Intent / why

The bot works, but its stack is frozen in 2019–2021 and it has real bugs and
security holes. Modernize it in small, reviewable increments so it: runs on
current Go, sends mail over TLS, produces files Kindle actually accepts (EPUB,
not the dead MOBI format), and builds in a sane container. Behavior for the end
user stays identical: send a document to the bot, it lands on your Kindle.

## Goals

- Current Go + maintained libraries.
- Fix every correctness bug (filename panic, double-`Wait`, missing early-returns).
- Fix security: TLS SMTP, remove `GOFLAGS="-insecure"`.
- Switch output MOBI → EPUB.
- Modern Docker + CI.
- Keep the existing env-var interface.

## Non-goals

- No new features (no web UI, multi-user auth, queue, rate limiting).
- No architecture rewrite — keep the single-package bot shape.

## Anti-goals (would make this a failure)

- Shipping without TLS SMTP (still leaks the email password).
- Shipping MOBI output (Amazon's Send-to-Kindle dropped MOBI in 2022).
- Silently breaking the existing env-var names.

## Constraints

- Runs as one Docker container with Calibre (`ebook-convert`) installed.
- Credentials stay environment variables.
- Fork workflow: changes land on a branch, PR-reviewed, not pushed to upstream.

## Approaches — key tech decisions

| Concern | Current | Decision | Why / alternative |
|---|---|---|---|
| Telegram lib | `telebot.v2` (archived) | `gopkg.in/telebot.v3` | v2 is unmaintained; v3 is the live fork. |
| Mail | `scorredoira/email` (2019) | `github.com/wneessen/go-mail` | Actively maintained, first-class TLS/STARTTLS + attachments. Alt: stdlib `net/smtp`+`mime/multipart` (zero-dep, more code, more risk). |
| Output format | `.mobi` | `.epub` | MOBI unsupported by Send-to-Kindle since 2022. |
| SMTP transport | `smtp.PlainAuth` plaintext | STARTTLS (587) / implicit TLS (465) | Plaintext creds are rejected by modern providers. |
| Go | 1.17 | 1.26 (test 1.26 + 1.27) | Current stable is 1.27; pin 1.26 for headroom. |
| Logging | `log` | `log/slog` | Stdlib structured logging, zero dep. |

## Execution order — sequential vs concurrent

Two code changes that touch the same file must be sequential; work on disjoint
files can run in parallel. The stages resolve into three tracks:

- **Track A (code) — sequential.** Two PRs, because the TLS fix overlaps the
  mail-library swap: doing TLS against the old lib would be thrown away.
- **Track B (infra) — parallel with A.** Dockerfile + CI touch no files that A
  touches; run it concurrently.
- **Track C (hardening) — sequential, after A2.** Needs telebot v3 + `slog` +
  `go-mail` in place first.

## Staged plan

### Stage 0 — Baseline (done)

- Forked to `francesco-plt/send-to-kindle-telegram-bot` (`origin`); upstream is
  `michaelfmnk/send-to-kindle-telegram-bot`.
- Verified `go build ./...` and `go test ./...` pass on the untouched tree.

### Track A — code (sequential)

**PR A1 — Correctness + EPUB (no dependency change)**

1. `documentHandler`: `return` after each failed step so it stops responding and
   sending after a download/convert/send error (fixes double-respond + "sends
   broken/absent file" bug).
2. Filename parsing: replace `strings.Split(...)` + `nameParts[:len-1]` (mangles
   multi-dot names via empty-separator join; case-sensitive; mishandles no-dot)
   with a `filepath.Ext`-based `splitFileName` helper.
3. `convert`: remove the redundant `cmd.Wait()` after `cmd.Run()` — the
   double-wait makes conversion return an error even on success.
4. Wire `SMTPPort` from env (`UBOT_SMTP_PORT`, default `587`).
5. Output `.mobi` → `.epub`; update `supportedFormats` (drop `mobi`, add `epub`).
6. Add unit tests: `needToConvert`, filename edge cases (no dot, `.tar.gz`),
   config defaults.

**PR A2 — Library refresh + TLS (after A1)**

1. `scorredoira/email` → `wneessen/go-mail`, with TLS (STARTTLS on 587, implicit
   TLS on 465) — this is where the plaintext-credentials hole actually closes.
2. `telebot.v2` → `telebot.v3` (import path + small API touchups).
3. `log` → `log/slog`.
4. `go.mod`: `go 1.26`, run `go mod tidy`.
5. Update `bot_test.go` for the new APIs.

### Track B — container + CI (parallel with A)

1. Rewrite `Dockerfile` as multi-stage: `golang:1.26` builder → Calibre-based
   runtime. Drop unused `python`/`ffmpeg`/`libsm6`/`libxext6`. Remove
   `GOFLAGS="-insecure"` and the `golang:1.16` copy. Run as non-root.
2. CI (`go.yml`): `actions/checkout@v4`, `actions/setup-go@v5`, test on Go
   1.26 + 1.27, add `golangci-lint`, build + test.
3. Docker workflow: `docker/login-action@v3` + `docker/build-push-action`, or
   keep the existing push pattern.

### Track C — hardening (sequential, after A2)

1. Graceful shutdown (`signal.NotifyContext` + `bot.Stop()`).
2. Context/timeouts on download and email send.
3. Webhook support (optional; long-polling is fine for one user).
4. Broader test coverage (mock SMTP + injected converter).

## What's left to the builder

Exact `go-mail` call signatures, `telebot.v3` API touchups, Dockerfile and CI
YAML specifics, and the precise test cases. Decisions above are final; the
builder fills in the mechanical details.
