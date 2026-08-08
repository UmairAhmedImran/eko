// Package manifest implements the snapshot manifest system for Eko.
//
// A manifest is a lightweight JSON document that describes the complete file
// tree of a snapshot as a map of relative-path → FileEntry (hash + mode + size).
// Manifests replace the old approach of copying the entire project directory
// into .eko/snapshots/<id>/ — the actual file bytes live in the CAS object
// store; the manifest just records where every file came from.
//
// Layout:
//
//	.eko/manifests/<id>.json
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eko/internal/objects"
)

const manifestsSubdir = "manifests"

// Manifest describes every file in a snapshot.
type Manifest struct {
	ID        string                        `json:"id"`
	CreatedAt time.Time                     `json:"created_at"`
	Message   string                        `json:"message"`
	// Tree maps slash-separated relative paths to their stored blob entries.
	Tree      map[string]objects.FileEntry  `json:"tree"`
	// EnvHash is the CAS hash of the captured .eko_env_vars.json blob.
	EnvHash   string                        `json:"env_hash,omitempty"`
}

// Write atomically serialises m to ekoDir/manifests/<id>.json.
func Write(ekoDir string, m *Manifest) error {
	dir := filepath.Join(ekoDir, manifestsSubdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("manifest: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("manifest: marshal: %w", err)
	}

	path := filepath.Join(dir, m.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("manifest: write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("manifest: rename: %w", err)
	}
	return nil
}

// Read deserialises the manifest for id from ekoDir/manifests/<id>.json.
func Read(ekoDir, id string) (*Manifest, error) {
	path := filepath.Join(ekoDir, manifestsSubdir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("manifest: read %s: %w", id, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: unmarshal %s: %w", id, err)
	}
	return &m, nil
}

// Exists reports whether a manifest file exists for id.
func Exists(ekoDir, id string) bool {
	_, err := os.Stat(filepath.Join(ekoDir, manifestsSubdir, id+".json"))
	return err == nil
}

// Delete removes the manifest file for id.
func Delete(ekoDir, id string) error {
	return os.Remove(filepath.Join(ekoDir, manifestsSubdir, id+".json"))
}

// AllHashes returns every blob hash referenced by any manifest in ekoDir.
// Used by the GC to determine which objects are still live.
func AllHashes(ekoDir string) (map[string]bool, error) {
	dir := filepath.Join(ekoDir, manifestsSubdir)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}

	live := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		m, err := Read(ekoDir, id)
		if err != nil {
			continue // skip corrupt manifests
		}
		for _, entry := range m.Tree {
			live[entry.Hash] = true
		}
		if m.EnvHash != "" {
			live[m.EnvHash] = true
		}
	}
	return live, nil
}

// IDFromPath extracts the snapshot ID from a stored path.
// Supports both the old ".eko/snapshots/<id>" and new ".eko/manifests/<id>.json" forms.
func IDFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".json")
}
