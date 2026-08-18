package notify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendWebhook_Success(t *testing.T) {
	var receivedPayload WebhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	timestamp := time.Date(2026, 8, 17, 10, 30, 0, 0, time.UTC)
	err := SendWebhook(server.URL, "8c9d1a2f", "Added AI change summary", "test message", timestamp)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if receivedPayload.SnapshotID != "8c9d1a2f" {
		t.Errorf("expected snapshot_id '8c9d1a2f', got '%s'", receivedPayload.SnapshotID)
	}
	if receivedPayload.Summary != "Added AI change summary" {
		t.Errorf("expected summary 'Added AI change summary', got '%s'", receivedPayload.Summary)
	}
	if receivedPayload.Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", receivedPayload.Message)
	}
	if receivedPayload.Timestamp != "2026-08-17T10:30:00Z" {
		t.Errorf("expected timestamp '2026-08-17T10:30:00Z', got '%s'", receivedPayload.Timestamp)
	}
	if receivedPayload.Text == "" {
		t.Error("expected non-empty text field")
	}
}

func TestSendWebhook_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := SendWebhook(server.URL, "8c9d1a2f", "test summary", "test message", time.Now())

	if err == nil {
		t.Error("expected error for server error response, got nil")
	}
}

func TestSendWebhook_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	err := SendWebhook(server.URL, "8c9d1a2f", "test summary", "test message", time.Now())

	if err == nil {
		t.Error("expected error for bad request response, got nil")
	}
}

func TestSendWebhook_InvalidURL(t *testing.T) {
	err := SendWebhook("not-a-valid-url", "8c9d1a2f", "test summary", "test message", time.Now())

	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestFormatMessage(t *testing.T) {
	timestamp := time.Date(2026, 8, 17, 10, 30, 0, 0, time.UTC)
	msg := formatMessage("abc123", "Test summary", timestamp)

	if msg == "" {
		t.Error("expected non-empty message")
	}

	// Check that all expected components are present
	expectedParts := []string{"abc123", "Test summary", "2026-08-17T10:30:00Z"}
	for _, part := range expectedParts {
		if !contains(msg, part) {
			t.Errorf("expected message to contain '%s', got '%s'", part, msg)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
