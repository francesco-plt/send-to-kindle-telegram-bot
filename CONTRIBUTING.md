# Contributing

Thanks for your interest in contributing to Send to Kindle Telegram Bot!

## Prerequisites

- **Go 1.26+** — see `go.mod` for the exact toolchain version.
- **Docker** (optional) — only needed to build and run the container image.

## Getting started

Clone the repository and install dependencies:

```shell
git clone https://github.com/francesco-plt/send-to-kindle-telegram-bot.git
cd send-to-kindle-telegram-bot
go mod download
```

## Build, test, lint

Run these before opening a pull request:

```shell
go build ./...
go test ./...
golangci-lint run ./...
```

## Commit messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) convention:

```
type(scope): description
```

Examples:

```
feat: add /healthz endpoint
fix(smtp): use implicit TLS on port 465
docs: rewrite README
```

## Pull requests

1. Fork the repository and create a branch off `master`.
2. Make your changes, adding tests where relevant.
3. Run `go build ./...`, `go test ./...`, and `golangci-lint run ./...`.
4. Open a pull request against `master` and fill out the PR template.

A Greptile bot will automatically review your pull request and post inline comments. Address its feedback before merging.
