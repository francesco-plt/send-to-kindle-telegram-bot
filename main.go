package main

import (
	"log/slog"
	"os"

	"github.com/francesco-plt/send-to-kindle-telegram-bot/bot"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	unkindleBot := bot.SendToKindleBot{
		Token:     os.Getenv("UBOT_TELEGRAM_TOKEN"),
		EmailFrom: os.Getenv("UBOT_EMAIL_FROM"),
		EmailTo:   os.Getenv("UBOT_EMAIL_TO"),
		SMTPHost:  os.Getenv("UBOT_SMTP_HOST"),
		SMTPPort:  os.Getenv("UBOT_SMTP_PORT"),
		Password:  os.Getenv("UBOT_PASSWORD"),
	}
	if err := unkindleBot.Start(); err != nil {
		slog.Error("could not start telegram bot", "error", err)
		os.Exit(1)
	}
}
