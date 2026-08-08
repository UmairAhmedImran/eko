package api

import (
	"eko/internal/manifest"
	"eko/internal/objects"
	"eko/internal/util"
	"io/fs"
	"os"
	"path/filepath"
)

// SnapshotRecord is the JSON-serializable form of a stored snapshot.
type SnapshotRecord struct {
	ID           string `json:"id"`
	Message      string `json:"message"`
	Path         string `json:"path"`
	CreatedAt    string `json:"createdAt"`
	FilesChanged int    `json:"filesChanged"`
}

// DiffFile holds the before/after content of a single changed file.
type DiffFile struct {
	Name     string `json:"name"`
	Original string `json:"original"`
	Modified string `json:"modified"`
}

// BuildDiff walks both snapshot targets (manifest or dir), collecting file pairs where content differs.
func BuildDiff(fromTarget, toTarget string) ([]DiffFile, error) {
	fromFiles, err := loadTargetFiles(fromTarget)
	if err != nil && fromTarget != "" {
		return nil, err
	}
	toFiles, err := loadTargetFiles(toTarget)
	if err != nil && toTarget != "" {
		return nil, err
	}

	seen := map[string]bool{}
	for k := range fromFiles {
		seen[k] = true
	}
	for k := range toFiles {
		seen[k] = true
	}

	var results []DiffFile
	for rel := range seen {
		orig := fromFiles[rel]
		mod := toFiles[rel]
		if orig == mod {
			continue
		}
		results = append(results, DiffFile{
			Name:     rel,
			Original: orig,
			Modified: mod,
		})
	}
	return results, nil
}

func loadTargetFiles(target string) (map[string]string, error) {
	res := make(map[string]string)
	if target == "" {
		return res, nil
	}

	// Check if target is a manifest file or ID
	ekoDir := ".eko"
	id := manifest.IDFromPath(target)

	if manifest.Exists(ekoDir, id) {
		m, err := manifest.Read(ekoDir, id)
		if err != nil {
			return nil, err
		}
		store, err := objects.New(ekoDir)
		if err != nil {
			return nil, err
		}

		for rel, entry := range m.Tree {
			b, err := store.Get(entry.Hash)
			if err == nil {
				res[rel] = string(b)
			}
		}
		return res, nil
	}

	// Fallback to reading directory
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		err := filepath.Walk(target, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if util.ShouldIgnore(info.Name(), info.IsDir()) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(target, path)
			relSlash := filepath.ToSlash(rel)
			if b, err := os.ReadFile(path); err == nil {
				res[relSlash] = string(b)
			}
			return nil
		})
		return res, err
	}

	return res, nil
}
