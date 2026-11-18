# Send to Kindle Telegram Bot

Send documents to your Kindle straight from Telegram.

[![Go](https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/francesco-plt/send-to-kindle-telegram-bot/branch/master/graph/badge.svg)](https://codecov.io/gh/francesco-plt/send-to-kindle-telegram-bot)
[![Release](https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/release.yml/badge.svg)](https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/release.yml)
[![Docker Image](https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/docker-image.yml/badge.svg?branch=master)](https://github.com/francesco-plt/send-to-kindle-telegram-bot/actions/workflows/docker-image.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/francesco-plt/send-to-kindle-telegram-bot)](https://goreportcard.com/report/github.com/francesco-plt/send-to-kindle-telegram-bot)
[![Go Version](https://img.shields.io/github/go-mod/go-version/francesco-plt/send-to-kindle-telegram-bot)](https://github.com/francesco-plt/send-to-kindle-telegram-bot)
[![License](https://img.shields.io/github/license/francesco-plt/send-to-kindle-telegram-bot)](LICENSE)

Send to Kindle is a Telegram bot that delivers documents to your Kindle over TLS SMTP, converting anything Amazon can't read natively to `.epub` with Calibre.

## Features

- **Document conversion** — formats Amazon can't read natively (e.g. `.mobi`) are converted to `.epub` with Calibre's `ebook-convert`; `.pdf`, `.doc`, `.docx`, `.rtf`, `.html`, and `.txt` are delivered as-is.
- **Kindle delivery** — emails the file (converted to `.epub` when needed) to your Kindle email address over TLS SMTP.
- **Health endpoint** — optional `/healthz` HTTP endpoint for uptime monitoring.
- **Containerized** — runs in Docker alongside Calibre, so `ebook-convert` is always available.

## Quickstart

```shell
docker compose up --build
```

Set the environment variables listed below first (for example in a `.env` file next to `docker-compose.yml`), then send a document to your bot on Telegram.

## Environment variables

| Variable               | Required | Description                                                    |
|------------------------|----------|----------------------------------------------------------------|
| `UBOT_TELEGRAM_TOKEN`  | yes      | Telegram bot token from [@BotFather](https://t.me/BotFather).  |
| `UBOT_EMAIL_FROM`      | yes      | Email address the bot sends from.                              |
| `UBOT_EMAIL_TO`        | yes      | Your Kindle email address to send books to.                    |
| `UBOT_SMTP_HOST`       | yes      | SMTP server host.                                              |
| `UBOT_SMTP_PORT`       | no       | SMTP server port (defaults to `587`).                          |
| `UBOT_PASSWORD`        | yes      | Password for the email account.                                |
| `PORT`                 | no       | Port for the optional `/healthz` endpoint (defaults to `8080`). |

## How it works

1. You send a document to the Telegram bot.
2. The bot downloads the file.
3. If needed, it converts the file to `.epub` using Calibre's `ebook-convert`.
4. The bot sends the `.epub` to your Kindle email address over TLS SMTP.

## Troubleshooting

- **The book never reaches your Kindle** — add the sender email (`UBOT_EMAIL_FROM`) to your Kindle's **Approved Personal Document E-mail List** (Amazon → Manage Your Content and Devices → Preferences → Personal Document Settings). Only approved senders can deliver to your Kindle address.
- **Email rejected or flagged as spam (SMTP2GO)** — if you use SMTP2GO, configure SPF and DKIM records for your sending domain. Without them, messages may be rejected or land in spam.
- **"could not convert file"** — make sure the file type is supported and that Calibre's `ebook-convert` is present in the image (the runtime image is `linuxserver/calibre`).
- **Bot won't start** — verify that `UBOT_TELEGRAM_TOKEN`, `UBOT_EMAIL_FROM`, `UBOT_EMAIL_TO`, and `UBOT_PASSWORD` are all set.

## Links

- [Dev.to post](https://dev.to/michaelfmnk/developing-send-to-kindle-telegram-bot-120c)
- Questions? [Open a GitHub Issue](https://github.com/francesco-plt/send-to-kindle-telegram-bot/issues)
