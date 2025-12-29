package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionSaveLoad_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Setenv("LLAMASIDEKICK_CONFIG_DIR", tmp); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("LLAMASIDEKICK_CONFIG_DIR") })

	projectRoot := filepath.Join(tmp, "project")
	s := New(projectRoot)
	s.SetMode("plan")
	s.SetLastEditedFile("main.go")
	s.AddMessage("user", "hello")
	s.AddMessage("assistant", "hi")

	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.ProjectRoot != projectRoot {
		t.Fatalf("expected project root %s, got %s", projectRoot, loaded.ProjectRoot)
	}
	if loaded.Mode != "plan" {
		t.Fatalf("expected mode plan, got %s", loaded.Mode)
	}
	if loaded.LastEditedFile != "main.go" {
		t.Fatalf("expected last edited file main.go, got %s", loaded.LastEditedFile)
	}
	if len(loaded.History) != 2 {
		t.Fatalf("expected 2 history messages, got %d", len(loaded.History))
	}
}
