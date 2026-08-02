package ai

import (
	"context"
	"fmt"
	"strings"

	"eko/internal/api"
)

// ChangeSet holds structured information about file changes between two snapshot states.
type ChangeSet struct {
	Diffs         []api.DiffFile `json:"diffs"`
	AddedFiles    []string       `json:"addedFiles"`
	ModifiedFiles []string       `json:"modifiedFiles"`
	DeletedFiles  []string       `json:"deletedFiles"`
	TotalInsertions int          `json:"totalInsertions"`
	TotalDeletions  int          `json:"totalDeletions"`
}

// SummaryResult holds the final generated AI summary and metadata.
type SummaryResult struct {
	Summary        string   `json:"summary"`
	FilesChanged   int      `json:"filesChanged"`
	AddedFiles     []string `json:"addedFiles"`
	ModifiedFiles  []string `json:"modifiedFiles"`
	DeletedFiles   []string `json:"deletedFiles"`
	TotalInsertions int     `json:"totalInsertions"`
	TotalDeletions  int     `json:"totalDeletions"`
	ProviderUsed   string   `json:"providerUsed"`
}

// AnalyzeDiff converts a slice of DiffFile into a structured ChangeSet.
func AnalyzeDiff(diffs []api.DiffFile) ChangeSet {
	var cs ChangeSet
	cs.Diffs = diffs

	for _, d := range diffs {
		if d.Original == "" && d.Modified != "" {
			cs.AddedFiles = append(cs.AddedFiles, d.Name)
			cs.TotalInsertions += countLines(d.Modified)
		} else if d.Original != "" && d.Modified == "" {
			cs.DeletedFiles = append(cs.DeletedFiles, d.Name)
			cs.TotalDeletions += countLines(d.Original)
		} else {
			cs.ModifiedFiles = append(cs.ModifiedFiles, d.Name)
			ins, del := diffLineCounts(d.Original, d.Modified)
			cs.TotalInsertions += ins
			cs.TotalDeletions += del
		}
	}
	return cs
}

// GenerateSnapshotSummary builds diffs between fromDir and toDir and calls the specified AI provider.
func GenerateSnapshotSummary(ctx context.Context, fromDir, toDir, providerName string) (*SummaryResult, error) {
	diffs, err := api.BuildDiff(fromDir, toDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build diff: %w", err)
	}

	cs := AnalyzeDiff(diffs)
	provider := GetProvider(providerName)

	summaryText, err := provider.GenerateSummary(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("ai provider error: %w", err)
	}

	return &SummaryResult{
		Summary:         summaryText,
		FilesChanged:    len(diffs),
		AddedFiles:      cs.AddedFiles,
		ModifiedFiles:   cs.ModifiedFiles,
		DeletedFiles:    cs.DeletedFiles,
		TotalInsertions: cs.TotalInsertions,
		TotalDeletions:  cs.TotalDeletions,
		ProviderUsed:    provider.Name(),
	}, nil
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func diffLineCounts(orig, mod string) (insertions, deletions int) {
	origLines := strings.Split(orig, "\n")
	modLines := strings.Split(mod, "\n")

	origMap := make(map[string]int)
	for _, l := range origLines {
		origMap[l]++
	}

	modMap := make(map[string]int)
	for _, l := range modLines {
		modMap[l]++
	}

	for _, l := range modLines {
		if origMap[l] > 0 {
			origMap[l]--
		} else {
			insertions++
		}
	}

	for _, l := range origLines {
		if modMap[l] > 0 {
			modMap[l]--
		} else {
			deletions++
		}
	}
	return insertions, deletions
}

// FormatPatchSnippet produces a concise representation of code diffs suitable for LLM prompts.
func FormatPatchSnippet(cs ChangeSet, maxBytes int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Summary of changes: %d files changed (+%d lines, -%d lines)\n\n",
		len(cs.Diffs), cs.TotalInsertions, cs.TotalDeletions))

	for _, d := range cs.Diffs {
		if sb.Len() > maxBytes {
			sb.WriteString("\n... (remaining diffs truncated for brevity)")
			break
		}

		if d.Original == "" {
			sb.WriteString(fmt.Sprintf("--- File Added: %s ---\n", d.Name))
			lines := strings.Split(d.Modified, "\n")
			limit := 20
			if len(lines) < limit {
				limit = len(lines)
			}
			for i := 0; i < limit; i++ {
				sb.WriteString("+ " + lines[i] + "\n")
			}
			if len(lines) > limit {
				sb.WriteString(fmt.Sprintf("+ ... (%d more lines)\n", len(lines)-limit))
			}
		} else if d.Modified == "" {
			sb.WriteString(fmt.Sprintf("--- File Deleted: %s ---\n", d.Name))
		} else {
			sb.WriteString(fmt.Sprintf("--- File Modified: %s ---\n", d.Name))
			ins, del := diffLineCounts(d.Original, d.Modified)
			sb.WriteString(fmt.Sprintf("  (+%d lines, -%d lines)\n", ins, del))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
