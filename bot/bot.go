package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mail "github.com/wneessen/go-mail"
	tb "gopkg.in/telebot.v3"
)

const (
	defaultSMTPPort = "587"
	tmpFilesPath    = "/files/"
)

var (
	// ErrNoToken - represents a validation error when Token not set
	ErrNoToken = errors.New("token for telegram bot not set")
	// ErrNoPassword - represents a validation error when Password not set
	ErrNoPassword = errors.New("password for email not set")
	// ErrNoEmailFrom - represents a validation error when EmailFrom not set
	ErrNoEmailFrom = errors.New("emailfrom not set")
	// ErrNoEmailTo - represents a validation error when EmailTo not set
	ErrNoEmailTo = errors.New("emailto not set")
	// ErrStartup - represents an error during bot initialization process
	ErrStartup = errors.New("could not create telebot instance")

	errConversion    = errors.New("could not convert file")
	supportedFormats = []string{"doc", "docx", "rtf", "htm", "html", "txt", "epub", "pdf"}
)

// SendToKindleBot stores bot configuration
type SendToKindleBot struct {
	Token     string
	EmailFrom string
	EmailTo   string
	SMTPHost  string
	SMTPPort  string
	Password  string
}

// Start starts bot. It is blocking.
// If there is an error during startup, returns it. Otherwise blocks
func (b *SendToKindleBot) Start() error {
	if err := b.verifyConfig(); err != nil {
		return err
	}

	bot, err := tb.NewBot(tb.Settings{
		Token:  b.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return ErrStartup
	}

	bot.Handle(tb.OnDocument, b.documentHandler(bot))
	bot.Start()

	return nil
}

func (b *SendToKindleBot) documentHandler(bot *tb.Bot) tb.HandlerFunc {
	return func(c tb.Context) error {
		msg := c.Message()
		doc := msg.Document
		fileNameWithoutExtension, extension := splitFileName(doc.FileName)

		originalFilePath := tmpFilesPath + doc.FileName
		if err := bot.Download(&doc.File, originalFilePath); err != nil {
			slog.Error("could not download file", "error", err)
			respond(bot, msg, "Sorry. I could not download file")
			return nil
		}
		defer removeSilently(originalFilePath)

		fileToSend := originalFilePath
		if needToConvert(extension) {
			outputFilePath := tmpFilesPath + fileNameWithoutExtension + ".epub"
			defer removeSilently(outputFilePath)
			if err := convert(originalFilePath, outputFilePath); err != nil {
				slog.Error("could not convert file", "error", err)
				respond(bot, msg, "Sorry. I could not convert file")
				return nil
			}
			fileToSend = outputFilePath
		}

		if err := b.sendFileViaEmail(fileToSend); err != nil {
			slog.Error("could not send file", "error", err)
			respond(bot, msg, "Sorry. I could not send file")
			return nil
		}

		return nil
	}
}

func needToConvert(extension string) bool {
	for _, format := range supportedFormats {
		if format == extension {
			return false
		}
	}
	return true
}

func splitFileName(name string) (base, ext string) {
	ext = filepath.Ext(name)
	return strings.TrimSuffix(name, ext), strings.TrimPrefix(strings.ToLower(ext), ".")
}

func respond(bot *tb.Bot, m *tb.Message, text string) {
	if _, err := bot.Send(m.Sender, text); err != nil {
		slog.Error("could not send a message", "chat_id", m.Sender.ID, "error", err)
	}
}

func convert(in, out string) error {
	output, err := exec.Command("ebook-convert", in, out).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ebook-convert failed: %w: %s", err, output)
	}
	if _, err := os.Stat(out); errors.Is(err, os.ErrNotExist) {
		return errConversion
	}
	return nil
}

func removeSilently(path string) {
	if err := os.Remove(path); err != nil {
		slog.Error("could not delete file", "path", path, "error", err)
	}
}

func (b *SendToKindleBot) verifyConfig() error {
	if b.Token == "" {
		return ErrNoToken
	}
	if b.Password == "" {
		return ErrNoPassword
	}
	if b.EmailFrom == "" {
		return ErrNoEmailFrom
	}
	if b.EmailTo == "" {
		return ErrNoEmailTo
	}
	if b.SMTPPort == "" {
		b.SMTPPort = defaultSMTPPort
	}
	return nil
}

func (b *SendToKindleBot) sendFileViaEmail(path string) error {
	msg := mail.NewMsg()
	if err := msg.From(b.EmailFrom); err != nil {
		return err
	}
	if err := msg.To(b.EmailTo); err != nil {
		return err
	}
	msg.Subject("Send to Kindle")
	msg.AttachFile(path)

	port, err := strconv.Atoi(b.SMTPPort)
	if err != nil {
		return fmt.Errorf("invalid SMTP port %q: %w", b.SMTPPort, err)
	}

	opts := []mail.Option{
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(b.EmailFrom),
		mail.WithPassword(b.Password),
	}

	// TLS is mandatory: port 465 uses implicit TLS, everything else (587) uses
	// STARTTLS. WithTLSPortPolicy(TLSMandatory) is STARTTLS-only, so implicit TLS
	// on 465 needs WithSSL.
	if port == mail.DefaultPortSSL {
		opts = append(opts, mail.WithSSL())
	} else {
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	}

	client, err := mail.NewClient(b.SMTPHost, opts...)
	if err != nil {
		return err
	}

	return client.DialAndSend(msg)
}
