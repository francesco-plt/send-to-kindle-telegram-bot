package bot

import (
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
