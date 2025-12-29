package modes

import (
	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/ollama"
	"github.com/yourusername/llamasidekick/internal/session"
)

// Mode represents a working mode interface
type Mode interface {
	// Name returns the mode name
	Name() string
	
	// Description returns a brief description of the mode
	Description() string
	
	// Run executes the mode
	Run(client *ollama.Client, session *session.Session, cfg *config.Config) error
	
	// GetSystemPrompt returns the system prompt for this mode
	GetSystemPrompt() string
}

// Available modes
const (
	ModePlan  = "plan"
	ModeEdit  = "edit"
	ModeAgent = "agent"
	ModeCmd   = "cmd"
	ModeAsk   = "ask"
)
