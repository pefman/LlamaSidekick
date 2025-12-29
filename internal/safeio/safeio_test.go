package safeio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWithinRoot_AllowsRelative(t *testing.T) {
	root := t.TempDir()
	abs, rel, err := ResolveWithinRoot(root, "dir/file.txt")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rel != filepath.Clean("dir/file.txt") {
		t.Fatalf("unexpected rel: %s", rel)
	}
	if abs == "" {
		t.Fatalf("expected abs path")
	}
	if !filepath.IsAbs(abs) {
		t.Fatalf("expected absolute path, got %s", abs)
	}
}

func TestResolveWithinRoot_RejectsAbsolute(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if !filepath.IsAbs(other) {
		t.Fatalf("expected temp dir to be absolute, got %s", other)
	}
	_, _, err := ResolveWithinRoot(root, other)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolveWithinRoot_RejectsEscape(t *testing.T) {
	root := t.TempDir()
	sep := string(os.PathSeparator)
	_, _, err := ResolveWithinRoot(root, ".."+sep+"secrets.txt")
	if err == nil {
		t.Fatalf("expected error")
	}
}
