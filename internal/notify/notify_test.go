package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Error("Default config should be enabled")
	}
	if !cfg.Desktop.Enabled {
		t.Error("Default desktop should be enabled")
	}
}

func TestNewNotifier(t *testing.T) {
	cfg := DefaultConfig()
	n := New(cfg)
	if n == nil {
		t.Fatal("New returned nil")
	}
	if !n.enabledSet[EventAgentError] {
		t.Error("EventAgentError should be enabled")
	}
}

func TestNotifyDisabled(t *testing.T) {
	cfg := Config{Enabled: false}
	n := New(cfg)
	err := n.Notify(Event{Type: EventAgentError})
	if err != nil {
		t.Errorf("Notify failed when disabled: %v", err)
	}
}

func TestWebhookNotification(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["text"] != "NTM: agent.error - Test error" {
			t.Errorf("Unexpected payload: %v", payload)
		}
	}))
	defer ts.Close()

	cfg := Config{
		Enabled: true,
		Events:  []string{"agent.error"},
		Webhook: WebhookConfig{
			Enabled:  true,
			URL:      ts.URL,
			Template: `{"text": "NTM: {{.Type}} - {{.Message}}"}`,
		},
	}

	n := New(cfg)
	err := n.Notify(Event{
		Type:    EventAgentError,
		Message: "Test error",
	})
	if err != nil {
		t.Errorf("Notify failed: %v", err)
	}
}

func TestLogNotification(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		Enabled: true,
		Events:  []string{"agent.error"},
		Log: LogConfig{
			Enabled: true,
			Path:    logPath,
		},
	}

	n := New(cfg)
	err := n.Notify(Event{
		Type:      EventAgentError,
		Message:   "Test log",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	content, _ := os.ReadFile(logPath)
	if len(content) == 0 {
		t.Error("Log file is empty")
	}
}

func TestHelperFunctions(t *testing.T) {
	evt := NewRateLimitEvent("sess", "p1", "cc", 30)
	if evt.Type != EventRateLimit {
		t.Errorf("NewRateLimitEvent type = %v", evt.Type)
	}
	if evt.Details["wait_seconds"] != "30" {
		t.Errorf("NewRateLimitEvent details = %v", evt.Details)
	}

	evt = NewAgentCrashedEvent("sess", "p1", "cc")
	if evt.Type != EventAgentCrashed {
		t.Errorf("NewAgentCrashedEvent type = %v", evt.Type)
	}

	evt = NewAgentErrorEvent("sess", "p1", "cc", "error")
	if evt.Type != EventAgentError {
		t.Errorf("NewAgentErrorEvent type = %v", evt.Type)
	}

	evt = NewHealthDegradedEvent("sess", 5, 1, 0)
	if evt.Type != EventHealthDegraded {
		t.Errorf("NewHealthDegradedEvent type = %v", evt.Type)
	}

	evt = NewRotationNeededEvent("sess", 1, "cc", "cmd")
	if evt.Type != EventRotationNeeded {
		t.Errorf("NewRotationNeededEvent type = %v", evt.Type)
	}
}
