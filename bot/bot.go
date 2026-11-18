package bot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

	convert func(in, out string) error
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go bot.Start()

	<-ctx.Done()
	slog.Info("shutting down")
	bot.Stop()

	return nil
}

func (b *SendToKindleBot) documentHandler(bot *tb.Bot) tb.HandlerFunc {
	return func(c tb.Context) error {
		msg := c.Message()
		doc := msg.Document
		fileNameWithoutExtension, extension := splitFileName(doc.FileName)

		originalFilePath := tmpFilesPath + doc.FileName
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := b.downloadFile(ctx, bot, &doc.File, originalFilePath); err != nil {
			slog.Error("could not download file", "error", err)
			respond(bot, msg, "Sorry. I could not download file")
			return nil
		}
		defer removeSilently(originalFilePath)

		fileToSend := originalFilePath
		if needToConvert(extension) {
			outputFilePath := tmpFilesPath + fileNameWithoutExtension + ".epub"
			defer removeSilently(outputFilePath)
			if err := b.runConvert(originalFilePath, outputFilePath); err != nil {
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

// downloadFile downloads a Telegram file with a cancellable request. It
// mirrors telebot's Bot.File/Bot.Download, which issue an http.NewRequest
// with no context and therefore cannot be cancelled by a wrapper goroutine.
func (b *SendToKindleBot) downloadFile(ctx context.Context, bot *tb.Bot, file *tb.File, localPath string) error {
	f, err := bot.FileByID(file.FileID)
	if err != nil {
		return err
	}
	url := bot.URL + "/file/bot" + bot.Token + "/" + f.FilePath
	file.FilePath = f.FilePath
	return downloadURL(ctx, url, localPath)
}

// downloadURL fetches url into localPath using a context-aware request.
func downloadURL(ctx context.Context, url, localPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telebot: expected status 200 but got %s", resp.Status)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (b *SendToKindleBot) runConvert(in, out string) error {
	if b.convert != nil {
		return b.convert(in, out)
	}
	return convert(in, out)
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

// smtpUsesImplicitTLS reports whether the given SMTP port requires
// implicit TLS (SSL). Port 465 is implicit TLS; all other ports use
// mandatory STARTTLS. There is intentionally no plaintext path.
func smtpUsesImplicitTLS(port string) bool {
	return port == "465"
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
	if smtpUsesImplicitTLS(b.SMTPPort) {
		opts = append(opts, mail.WithSSL())
	} else {
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	}

	client, err := mail.NewClient(b.SMTPHost, opts...)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return client.DialAndSendWithContext(ctx, msg)
}
