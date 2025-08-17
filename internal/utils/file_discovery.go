// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var DefaultIgnorePatterns = []string{
	"node_modules", ".git", "build", "dist", "target",
	"bin", ".cache", ".next", ".vscode", ".idea",
	"vendor", "__pycache__", ".nuxt", ".output",
	".svelte-kit", ".turbo", "coverage", ".nyc_output",
}

type FileInfo struct {
	Path    string
	ModTime time.Time
	Size    int64
}

type FileDiscoveryOptions struct {
	MaxDepth       int
	IgnorePatterns []string
	FileExtensions []string
	SortByModTime  bool
	MaxFiles       int // Limit number of results for performance
}

// FindJSONFiles finds JSON files in the given directory with default options
func FindJSONFiles(rootPath string) ([]string, error) {
	opts := FileDiscoveryOptions{
		MaxDepth:       3,
		IgnorePatterns: DefaultIgnorePatterns,
		FileExtensions: []string{".json"},
		SortByModTime:  true,
		MaxFiles:       100,
	}
	return FindFiles(rootPath, opts)
}

// FindConfigFiles finds all config files (JSON, YAML, and Markdown) in a directory structure
func FindConfigFiles(rootPath string) ([]string, error) {
	opts := FileDiscoveryOptions{
		MaxDepth:       3,
		IgnorePatterns: DefaultIgnorePatterns,
		FileExtensions: []string{".json", ".yaml", ".yml", ".md", ".markdown"},
		SortByModTime:  true,
		MaxFiles:       100,
	}
	return FindFiles(rootPath, opts)
}

// FindFiles recursively finds files matching the given options
func FindFiles(rootPath string, opts FileDiscoveryOptions) ([]string, error) {
	if opts.MaxDepth == 0 {
		opts.MaxDepth = 3
	}
	if opts.MaxFiles == 0 {
		opts.MaxFiles = 100
	}

	var files []FileInfo
	ignoreMap := make(map[string]bool)
	for _, pattern := range opts.IgnorePatterns {
		ignoreMap[pattern] = true
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Calculate depth
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check ignore patterns
		if info.IsDir() && ignoreMap[info.Name()] {
			return filepath.SkipDir
		}

		// Check if it's a file we want
		if !info.IsDir() {
			if len(opts.FileExtensions) > 0 {
				ext := filepath.Ext(path)
				found := false
				for _, allowedExt := range opts.FileExtensions {
					if ext == allowedExt {
						found = true
						break
					}
				}
				if !found {
					return nil
				}
			}

			// Convert to relative path for display
			files = append(files, FileInfo{
				Path:    relPath,
				ModTime: info.ModTime(),
				Size:    info.Size(),
			})

			// Stop if we hit the max files limit
			if len(files) >= opts.MaxFiles {
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files
	if opts.SortByModTime {
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime.After(files[j].ModTime)
		})
	} else {
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})
	}

	// Convert to string slice
	result := make([]string, 0, len(files))
	for _, file := range files {
		result = append(result, file.Path)
	}

	return result, nil
}
