package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yourusername/llamasidekick/internal/config"
)

// Message represents a single conversation message
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Session represents a working session
type Session struct {
	ID          string    `json:"id"`
	ProjectRoot string    `json:"project_root"`
	ActiveFiles []string  `json:"active_files"`
	Mode        string    `json:"mode"`
	History     []Message `json:"history"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// New creates a new session
func New(projectRoot string) *Session {
	return &Session{
		ID:          generateID(),
		ProjectRoot: projectRoot,
		ActiveFiles: []string{},
		Mode:        "",
		History:     []Message{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// AddMessage adds a message to the session history
func (s *Session) AddMessage(role, content string) {
	s.History = append(s.History, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
}

// AddFile adds a file to the active files list
func (s *Session) AddFile(filepath string) {
	// Check if file is already in the list
	for _, f := range s.ActiveFiles {
		if f == filepath {
			return
		}
	}
	s.ActiveFiles = append(s.ActiveFiles, filepath)
	s.UpdatedAt = time.Now()
}

// RemoveFile removes a file from the active files list
func (s *Session) RemoveFile(filepath string) {
	for i, f := range s.ActiveFiles {
		if f == filepath {
			s.ActiveFiles = append(s.ActiveFiles[:i], s.ActiveFiles[i+1:]...)
			s.UpdatedAt = time.Now()
			return
		}
	}
}

// SetMode sets the current mode
func (s *Session) SetMode(mode string) {
	s.Mode = mode
	s.UpdatedAt = time.Now()
}

// Save saves the session to disk
func (s *Session) Save() error {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config dir: %w", err)
	}
	
	sessionFile := filepath.Join(configDir, "session.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	if err := os.WriteFile(sessionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	
	return nil
}

// SaveDebug saves a debug snapshot of the session with mode-specific filename
func (s *Session) SaveDebug(mode string) error {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config dir: %w", err)
	}
	
	timestamp := time.Now().Format("20060102_150405")
	sessionFile := filepath.Join(configDir, fmt.Sprintf("session_%s_%s.json", mode, timestamp))
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	if err := os.WriteFile(sessionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	
	return nil
}

// Load loads a session from disk
func Load(projectRoot string) (*Session, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}
	
	sessionFile := filepath.Join(configDir, "session.json")
	
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No session exists, create a new one
			return New(projectRoot), nil
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}
	
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	
	return &session, nil
}

// generateID generates a simple session ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
