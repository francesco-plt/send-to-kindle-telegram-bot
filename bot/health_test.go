package bot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsHealthy(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		lastPing int64 // time.Time.UnixNano() of last ping; 0 means never pinged
		want     bool
	}{
		{name: "never pinged", lastPing: 0, want: false},
		{name: "pinged now", lastPing: now.UnixNano(), want: true},
		{name: "pinged 30s ago", lastPing: now.Add(-30 * time.Second).UnixNano(), want: true},
		{name: "pinged 60s ago", lastPing: now.Add(-60 * time.Second).UnixNano(), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &SendToKindleBot{}
			b.lastPing.Store(tt.lastPing)
			if got := b.isHealthy(now); got != tt.want {
				t.Errorf("isHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		lastPing   int64
		wantStatus int
	}{
		{name: "fresh ping", lastPing: time.Now().UnixNano(), wantStatus: http.StatusOK},
		{name: "never pinged", lastPing: 0, wantStatus: http.StatusServiceUnavailable},
		{name: "stale ping", lastPing: time.Now().Add(-60 * time.Second).UnixNano(), wantStatus: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &SendToKindleBot{}
			b.lastPing.Store(tt.lastPing)

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()

			b.healthHandler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("healthHandler() status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRunHeartbeatPingsImmediately(t *testing.T) {
	firstCall := make(chan struct{}, 1)

	b := &SendToKindleBot{}
	ping := func() error {
		select {
		case firstCall <- struct{}{}:
		default:
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		b.runHeartbeat(ctx, ping)
		close(done)
	}()

	// Block on the first ping (not a sleep): assert it fired without waiting
	// for the 30s ticker and that lastPing was updated.
	select {
	case <-firstCall:
	case <-time.After(5 * time.Second):
		t.Fatal("runHeartbeat() did not ping immediately")
	}

	if b.lastPing.Load() == 0 {
		t.Fatal("runHeartbeat() did not record lastPing after the immediate ping")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runHeartbeat() did not return after context cancellation")
	}
}
