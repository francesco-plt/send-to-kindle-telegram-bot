package bot

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	tb "gopkg.in/telebot.v3"
)

// healthyMaxAge is the maximum age of the last successful Telegram ping for
// the bot to still be considered healthy.
const healthyMaxAge = 45 * time.Second

// isHealthy reports whether the bot has reached Telegram recently enough to be
// considered operational.
func (b *SendToKindleBot) isHealthy(now time.Time) bool {
	lp := b.lastPing.Load()
	return lp > 0 && now.Sub(time.Unix(0, lp)) < healthyMaxAge
}

// healthHandler returns an http.Handler that reports operational status.
// It returns 200 only when the bot has reached Telegram recently, and 503
// otherwise.
func (b *SendToKindleBot) healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if b.isHealthy(time.Now()) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok\n"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unhealthy\n"))
	})
}

// heartbeat pings Telegram immediately, then every 30s, recording the time of
// the last success so the health endpoint can reflect real reachability. It
// runs until ctx is cancelled.
func (b *SendToKindleBot) heartbeat(ctx context.Context, bot *tb.Bot) {
	b.runHeartbeat(ctx, func() error {
		_, err := bot.Raw("getMe", nil)
		return err
	})
}

// runHeartbeat pings immediately, then every 30s until ctx is done. On success
// it stores time.Now().UnixNano() into b.lastPing; on error it logs and leaves
// lastPing stale.
func (b *SendToKindleBot) runHeartbeat(ctx context.Context, ping func() error) {
	pingOnce := func() {
		if err := ping(); err != nil {
			slog.Error("telegram ping failed", "error", err)
			return
		}
		b.lastPing.Store(time.Now().UnixNano())
	}

	// Ping once immediately so /healthz reflects reachability without waiting
	// for the first tick.
	pingOnce()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingOnce()
		}
	}
}

// serveHealth runs an HTTP server exposing the /healthz endpoint on the port
// given by the PORT environment variable. If PORT is empty, it returns
// immediately and the health server is disabled. The server is gracefully
// shut down when ctx is cancelled.
func (b *SendToKindleBot) serveHealth(ctx context.Context) {
	port := os.Getenv("PORT")
	if port == "" {
		return
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           b.healthHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server failed", "error", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("health server shutdown failed", "error", err)
	}
}
