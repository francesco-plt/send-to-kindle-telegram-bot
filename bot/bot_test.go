package bot

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestUnkindleBot_verifyConfig(t *testing.T) {
	type fields struct {
		Token     string
		EmailFrom string
		EmailTo   string
		SmtpHost  string
		SmtpPort  string
		Password  string
	}
	tests := []struct {
		name         string
		fields       fields
		wantErr      bool
		errorMessage string
	}{
		{
			name: "should pass validation",
			fields: fields{
				Token:     "hello",
				EmailFrom: "from",
				EmailTo:   "to",
				SmtpHost:  "host",
				SmtpPort:  "port",
				Password:  "pass",
			},
			wantErr: false,
		},
		{
			name: "should not pass validation if Token empty",
			fields: fields{
				Token:     "",
				EmailFrom: "from",
				EmailTo:   "to",
				SmtpHost:  "host",
				SmtpPort:  "port",
				Password:  "pass",
			},
			wantErr:      true,
			errorMessage: "token for telegram bot not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &SendToKindleBot{
				Token:     tt.fields.Token,
				EmailFrom: tt.fields.EmailFrom,
				EmailTo:   tt.fields.EmailTo,
				SMTPHost:  tt.fields.SmtpHost,
				SMTPPort:  tt.fields.SmtpPort,
				Password:  tt.fields.Password,
			}
			err := b.verifyConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && err.Error() != tt.errorMessage {
				t.Errorf("verifyConfig() error '%v' not matches '%s'", err, tt.errorMessage)
			}
		})
	}
}

func TestSplitFileName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		base string
		ext  string
	}{
		{name: "simple", in: "book.pdf", base: "book", ext: "pdf"},
		{name: "multi dot", in: "book.part1.pdf", base: "book.part1", ext: "pdf"},
		{name: "no extension", in: "README", base: "README", ext: ""},
		{name: "uppercase extension", in: "FILE.PDF", base: "FILE", ext: "pdf"},
		{name: "hidden file", in: ".hidden", base: "", ext: "hidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, ext := splitFileName(tt.in)
			if base != tt.base {
				t.Errorf("splitFileName(%q) base = %q, want %q", tt.in, base, tt.base)
			}
			if ext != tt.ext {
				t.Errorf("splitFileName(%q) ext = %q, want %q", tt.in, ext, tt.ext)
			}
		})
	}
}

func TestNeedToConvert(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{ext: "pdf", want: false},
		{ext: "epub", want: false},
		{ext: "mobi", want: true},
		{ext: "xlsx", want: true},
		{ext: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			if got := needToConvert(tt.ext); got != tt.want {
				t.Errorf("needToConvert(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestVerifyConfigDefaults(t *testing.T) {
	b := &SendToKindleBot{
		Token:     "token",
		EmailFrom: "from",
		EmailTo:   "to",
		SMTPHost:  "host",
		Password:  "pass",
	}

	if err := b.verifyConfig(); err != nil {
		t.Fatalf("verifyConfig() error = %v, want nil", err)
	}

	if b.SMTPPort != "587" {
		t.Errorf("verifyConfig() SMTPPort = %q, want %q", b.SMTPPort, "587")
	}
}

func TestSMTPUsesImplicitTLS(t *testing.T) {
	tests := []struct {
		port string
		want bool
	}{
		{port: "465", want: true},  // implicit TLS
		{port: "587", want: false}, // STARTTLS
		{port: "25", want: false},  // STARTTLS — no plaintext fallback exists
		{port: "", want: false},    // empty defaults to 587 in verifyConfig, so STARTTLS
	}

	for _, tt := range tests {
		t.Run(tt.port, func(t *testing.T) {
			if got := smtpUsesImplicitTLS(tt.port); got != tt.want {
				t.Errorf("smtpUsesImplicitTLS(%q) = %v, want %v", tt.port, got, tt.want)
			}
		})
	}
}

// TestSendFileViaEmailRefusesPlaintext proves the "no plaintext fallback"
// security property: when the SMTP server does not advertise STARTTLS, a
// mandatory-TLS client must refuse to send rather than fall back to plaintext.
func TestSendFileViaEmailRefusesPlaintext(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer func() { _ = conn.Close() }()
		if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			serverDone <- err
			return
		}

		// SMTP greeting.
		if _, err := io.WriteString(conn, "220 localhost ESMTP\r\n"); err != nil {
			serverDone <- err
			return
		}

		br := bufio.NewReader(conn)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				// go-mail closes the connection once it refuses to proceed
				// without STARTTLS; EOF is the expected clean shutdown.
				if err == io.EOF {
					serverDone <- nil
					return
				}
				serverDone <- err
				return
			}

			upper := strings.ToUpper(strings.TrimSpace(line))
			if strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO") {
				// Advertise capabilities that do NOT include STARTTLS.
				if _, err := io.WriteString(conn, "250-localhost\r\n250 SIZE 35882577\r\n"); err != nil {
					serverDone <- err
					return
				}
			}
			// Otherwise keep draining until the client closes the connection.
		}
	}()

	bot := &SendToKindleBot{
		SMTPHost:  "127.0.0.1",
		SMTPPort:  strconv.Itoa(ln.Addr().(*net.TCPAddr).Port),
		EmailFrom: "from@example.com",
		EmailTo:   "to@example.com",
		Password:  "secret",
	}

	tmpFile := filepath.Join(t.TempDir(), "book.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	err = bot.sendFileViaEmail(tmpFile)
	if err == nil {
		t.Fatal("sendFileViaEmail() error = nil, want error: server advertised no STARTTLS")
	}
	// The error must be the mandatory-STARTTLS refusal specifically, not some
	// incidental failure (e.g. a missing AUTH capability). This is what makes
	// the test sensitive to the "no plaintext fallback" property.
	if !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("sendFileViaEmail() error = %v, want a STARTTLS refusal", err)
	}

	select {
	case serr := <-serverDone:
		if serr != nil {
			t.Fatalf("SMTP server: %v", serr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("SMTP server did not shut down cleanly")
	}
}

func TestRunConvertInjected(t *testing.T) {
	t.Run("injected error", func(t *testing.T) {
		var gotOut string
		b := &SendToKindleBot{
			convert: func(in, out string) error {
				gotOut = out
				return errors.New("boom")
			},
		}

		if err := b.runConvert("/in.pdf", "/out.epub"); err == nil || err.Error() != "boom" {
			t.Fatalf("runConvert() error = %v, want boom", err)
		}
		if gotOut != "/out.epub" {
			t.Errorf("runConvert() out = %q, want %q", gotOut, "/out.epub")
		}
	})

	t.Run("injected nil", func(t *testing.T) {
		b := &SendToKindleBot{
			convert: func(in, out string) error { return nil },
		}

		if err := b.runConvert("/in.pdf", "/out.epub"); err != nil {
			t.Fatalf("runConvert() error = %v, want nil", err)
		}
	})
}

func TestDownloadURLTimeout(t *testing.T) {
	canceled := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(canceled)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := downloadURL(ctx, srv.URL, filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("downloadURL() error = nil, want context deadline exceeded")
	}

	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not observe client cancellation")
	}
}
