package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Provider interface defines the contract for generating AI change summaries.
type Provider interface {
	Name() string
	GenerateSummary(ctx context.Context, cs ChangeSet) (string, error)
}

// GetProvider returns the Provider matching providerName, falling back to "auto".
func GetProvider(providerName string) Provider {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "heuristic", "offline", "mock":
		return &HeuristicProvider{}
	case "openai":
		return &OpenAIProvider{}
	case "gemini":
		return &GeminiProvider{}
	case "auto", "":
		if os.Getenv("GEMINI_API_KEY") != "" {
			return &GeminiProvider{}
		}
		if os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("EKO_AI_API_KEY") != "" {
			return &OpenAIProvider{}
		}
		return &HeuristicProvider{}
	default:
		return &HeuristicProvider{}
	}
}

// HeuristicProvider generates structured summaries locally without network/LLM dependencies.
type HeuristicProvider struct{}

func (h *HeuristicProvider) Name() string {
	return "heuristic"
}

func (h *HeuristicProvider) GenerateSummary(ctx context.Context, cs ChangeSet) (string, error) {
	totalFiles := len(cs.Diffs)
	if totalFiles == 0 {
		return "No changes detected in snapshot.", nil
	}

	var parts []string
	if len(cs.AddedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("added %d file(s) [%s]", len(cs.AddedFiles), summarizeFileList(cs.AddedFiles)))
	}
	if len(cs.ModifiedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("modified %d file(s) [%s]", len(cs.ModifiedFiles), summarizeFileList(cs.ModifiedFiles)))
	}
	if len(cs.DeletedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("deleted %d file(s) [%s]", len(cs.DeletedFiles), summarizeFileList(cs.DeletedFiles)))
	}

	summary := fmt.Sprintf("Snapshot Changes (%d files: +%d/-%d lines): %s.",
		totalFiles, cs.TotalInsertions, cs.TotalDeletions, strings.Join(parts, ", "))

	// Add detailed breakdown by component/extension if available
	categories := categorizeFiles(cs)
	if len(categories) > 0 {
		summary += " High-level impact: " + strings.Join(categories, "; ") + "."
	}

	return summary, nil
}

func summarizeFileList(files []string) string {
	if len(files) <= 3 {
		return strings.Join(files, ", ")
	}
	return fmt.Sprintf("%s, %s and %d more", files[0], files[1], len(files)-2)
}

func categorizeFiles(cs ChangeSet) []string {
	var results []string
	extCounts := make(map[string]int)

	allFiles := append(append(cs.AddedFiles, cs.ModifiedFiles...), cs.DeletedFiles...)
	for _, f := range allFiles {
		ext := filepath.Ext(f)
		if ext == "" {
			ext = filepath.Base(f)
		}
		extCounts[ext]++
	}

	for ext, count := range extCounts {
		results = append(results, fmt.Sprintf("%d %s file(s)", count, ext))
	}
	return results
}

// OpenAIProvider calls OpenAI-compatible Chat Completions API.
type OpenAIProvider struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func (o *OpenAIProvider) Name() string {
	return "openai"
}

func (o *OpenAIProvider) GenerateSummary(ctx context.Context, cs ChangeSet) (string, error) {
	apiKey := o.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("EKO_AI_API_KEY")
		}
	}
	if apiKey == "" {
		// Fall back gracefully to heuristic provider if key is missing
		hp := &HeuristicProvider{}
		return hp.GenerateSummary(ctx, cs)
	}

	baseURL := o.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("EKO_AI_ENDPOINT")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
	}

	model := o.Model
	if model == "" {
		model = os.Getenv("EKO_AI_MODEL")
		if model == "" {
			model = "gpt-4o-mini"
		}
	}

	snippet := FormatPatchSnippet(cs, 4000)
	prompt := "You are Eko, an AI snapshot assistant. Summarize the following code changes concisely in 1-3 sentences focusing on what functional changes were introduced.\n\n" + snippet

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a concise codebase summary generator."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 200,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := o.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		// Fallback to heuristic on connection error
		hp := &HeuristicProvider{}
		res, _ := hp.GenerateSummary(ctx, cs)
		return res, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		hp := &HeuristicProvider{}
		res, _ := hp.GenerateSummary(ctx, cs)
		return res, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &parsed); err != nil || len(parsed.Choices) == 0 {
		hp := &HeuristicProvider{}
		return hp.GenerateSummary(ctx, cs)
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// GeminiProvider calls Google Gemini REST API.
type GeminiProvider struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func (g *GeminiProvider) Name() string {
	return "gemini"
}

func (g *GeminiProvider) GenerateSummary(ctx context.Context, cs ChangeSet) (string, error) {
	apiKey := g.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("EKO_AI_API_KEY")
		}
	}
	if apiKey == "" {
		hp := &HeuristicProvider{}
		return hp.GenerateSummary(ctx, cs)
	}

	model := g.Model
	if model == "" {
		model = os.Getenv("EKO_AI_MODEL")
		if model == "" {
			model = "gemini-1.5-flash"
		}
	}

	snippet := FormatPatchSnippet(cs, 4000)
	prompt := "You are Eko, an AI snapshot assistant. Summarize the following code changes concisely in 1-3 sentences focusing on what functional changes were introduced:\n\n" + snippet

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := g.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		hp := &HeuristicProvider{}
		res, _ := hp.GenerateSummary(ctx, cs)
		return res, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		hp := &HeuristicProvider{}
		res, _ := hp.GenerateSummary(ctx, cs)
		return res, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &parsed); err != nil || len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		hp := &HeuristicProvider{}
		return hp.GenerateSummary(ctx, cs)
	}

	return strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text), nil
}
