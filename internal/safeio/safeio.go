package safeio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveWithinRoot resolves a user-provided relative path into an absolute path within root.
// It rejects absolute paths and any path that escapes the root via .. segments.
func ResolveWithinRoot(root string, userPath string) (absPath string, relPath string, err error) {
	if root == "" {
		return "", "", fmt.Errorf("project root is empty")
	}
	if userPath == "" {
		return "", "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(userPath) {
		return "", "", fmt.Errorf("absolute paths are not allowed: %s", userPath)
	}

	clean := filepath.Clean(userPath)
	if clean == "." {
		return "", "", fmt.Errorf("invalid path: %s", userPath)
	}

	sep := string(os.PathSeparator)
	if clean == ".." || strings.HasPrefix(clean, ".."+sep) {
		return "", "", fmt.Errorf("path escapes project root: %s", userPath)
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve project root: %w", err)
	}
	joined := filepath.Join(rootAbs, clean)
	joinedAbs, err := filepath.Abs(joined)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}

	rootWithSep := rootAbs
	if !strings.HasSuffix(rootWithSep, sep) {
		rootWithSep += sep
	}
	if joinedAbs != rootAbs && !strings.HasPrefix(joinedAbs, rootWithSep) {
		return "", "", fmt.Errorf("resolved path is outside project root")
	}

	return joinedAbs, clean, nil
}

// WriteFileWithBackup writes content to absPath. If the file exists, it first writes a backup
// to absPath+".backup".
func WriteFileWithBackup(absPath string, content []byte) (backupPath string, err error) {
	if absPath == "" {
		return "", fmt.Errorf("absPath is empty")
	}

	if info, statErr := os.Stat(absPath); statErr == nil && !info.IsDir() {
		backupPath = absPath + ".backup"
		existing, err := os.ReadFile(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to read existing file for backup: %w", err)
		}
		if err := os.WriteFile(backupPath, existing, 0644); err != nil {
			return "", fmt.Errorf("failed to write backup: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return backupPath, fmt.Errorf("failed to create directories: %w", err)
	}

	if err := os.WriteFile(absPath, content, 0644); err != nil {
		return backupPath, fmt.Errorf("failed to write file: %w", err)
	}

	return backupPath, nil
}
