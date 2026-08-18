// Package notify provides notification delivery mechanisms for Eko snapshots.
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookPayload represents the JSON payload sent to webhook endpoints.
type WebhookPayload struct {
	Text       string `json:"text"`
	SnapshotID string `json:"snapshot_id"`
	Summary    string `json:"summary"`
	Message    string `json:"message,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// SendWebhook sends a JSON POST request to the specified webhook URL.
// It is designed to work with both Slack and Discord webhook endpoints.
// Returns an error if the request fails or receives a non-2xx response.
func SendWebhook(webhookURL string, snapshotID, summary, message string, timestamp time.Time) error {
	payload := WebhookPayload{
		Text:       formatMessage(snapshotID, summary, timestamp),
		SnapshotID: snapshotID,
		Summary:    summary,
		Message:    message,
		Timestamp:  timestamp.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// formatMessage creates a human-readable message for Slack/Discord.
func formatMessage(snapshotID, summary string, timestamp time.Time) string {
	return fmt.Sprintf("Eko Snapshot: %s\n\nSummary: %s\n\nCreated: %s",
		snapshotID,
		summary,
		timestamp.Format(time.RFC3339),
	)
}
