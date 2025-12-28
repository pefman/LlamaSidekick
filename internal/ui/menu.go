package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/modes"
	"github.com/yourusername/llamasidekick/internal/ollama"
	"github.com/yourusername/llamasidekick/internal/session"
)

type menuItem struct {
	name        string
	description string
	isMode      bool
	mode        modes.Mode
}

type menuModel struct {
	choices  []menuItem
	cursor   int
	selected bool
	cfg      *config.Config
	client   *ollama.Client
	session  *session.Session
}

func initialModel(cfg *config.Config) menuModel {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Load or create session
	sess, err := session.Load(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load session: %v\n", err)
		sess = session.New(cwd)
	}

	// Create Ollama client
	client := ollama.NewClient(cfg.Ollama.Host, cfg.Ollama.Model)
	client.Debug = cfg.Ollama.Debug

	return menuModel{
		choices: []menuItem{
			{name: "Plan", description: "Create development plans and break down tasks", isMode: true, mode: &modes.PlanMode{}},
			{name: "Edit", description: "Get help editing code with suggestions and diffs", isMode: true, mode: &modes.EditMode{}},
			{name: "Agent", description: "Autonomous multi-step task execution and problem solving", isMode: true, mode: &modes.AgentMode{}},
			{name: "CMD", description: "Get help with commands - generates but never executes", isMode: true, mode: &modes.CmdMode{}},
			{name: "Configure Models", description: "Assign different models to different modes", isMode: false},
		},
		cursor:   0,
		selected: false,
		cfg:      cfg,
		client:   client,
		session:  sess,
	}
}

func (m menuModel) Init() tea.Cmd {
	return nil
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter":
			m.selected = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m menuModel) View() string {
	var s strings.Builder

	// Title - bold + magenta
	s.WriteString("\n\033[1;38;5;205m🦙 LlamaSidekick\033[0m\n\n")
	s.WriteString("\033[38;5;240mSelect a mode:\033[0m\n\n")

	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
			// Bold + color for selected
			s.WriteString(cursor + "\033[1;38;5;170m" + choice.name + "\033[0m\n")
		} else {
			s.WriteString(cursor + choice.name + "\n")
		}
		s.WriteString("  \033[38;5;240m" + choice.description + "\033[0m\n")
	}

	s.WriteString("\n")
	s.WriteString("\033[38;5;240mPress q to quit\033[0m\n")

	return s.String()
}

// Run starts the UI
func Run(cfg *config.Config) error {
	// Check Ollama connection first
	client := ollama.NewClient(cfg.Ollama.Host, cfg.Ollama.Model)
	client.Debug = cfg.Ollama.Debug
	if err := client.CheckConnection(); err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s: %w\nMake sure Ollama is running with: ollama serve", cfg.Ollama.Host, err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Load or create session
	sess, err := session.Load(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load session: %v\n", err)
		sess = session.New(cwd)
	}

	// Handle first run - if no model is configured, prompt user to select one
	if cfg.Ollama.Model == "" {
		selectedModel, err := RunFirstRun(client, cfg)
		if err != nil {
			return fmt.Errorf("first run setup failed: %w", err)
		}
		
		// Update config with selected model
		cfg.Ollama.Model = selectedModel
		cfg.Models.Plan = selectedModel
		cfg.Models.Edit = selectedModel
		cfg.Models.Agent = selectedModel
		cfg.Models.CMD = selectedModel
		
		// Save config
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		
		fmt.Printf("\n✓ Configuration saved! Using %s as default model.\n\n", selectedModel)
	}

	// Show welcome message and start prompt
	fmt.Println("\n\033[1;38;5;205m🦙 LlamaSidekick\033[0m")
	fmt.Println("\033[38;5;240mQuick commands: /plan, /edit, /agent, /cmd | Press 'm' for menu | 'q' to quit\033[0m")
	fmt.Println()

	return RunPrompt(cfg, client, sess)
}

// ShowMenu displays the interactive menu (called from prompt)
func ShowMenu(cfg *config.Config, client *ollama.Client, sess *session.Session) error {
	for {
		// Run the menu
		p := tea.NewProgram(initialModelWithSession(cfg, sess), tea.WithAltScreen())
		m, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running menu: %w", err)
		}

		// If a selection was made, process it
		model := m.(menuModel)
		if !model.selected {
			// User quit
			return nil
		}

		selectedItem := model.choices[model.cursor]
		
		if selectedItem.isMode {
			// Run the mode
			if err := selectedItem.mode.Run(model.client, model.session, model.cfg); err != nil {
				return err
			}
			// After mode exits, loop back to main menu
		} else if selectedItem.name == "Configure Models" {
			// Configure Models option
			if err := RunModelConfig(model.client, model.cfg); err != nil {
				return err
			}
			// Reload config after changes
			newCfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("error reloading config: %w", err)
			}
			cfg = newCfg
			// Update client debug flag
			client.Debug = cfg.Ollama.Debug
		} else if selectedItem.name == "Settings" {
			// Settings option
			if err := RunSettings(model.cfg); err != nil {
				return err
			}
			// Reload config after changes
			newCfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("error reloading config: %w", err)
			}
			cfg = newCfg
			// Update client debug flag
			client.Debug = cfg.Ollama.Debug
		} else if selectedItem.name == "Toggle Debug Mode" {
			// Toggle debug mode
			cfg.Ollama.Debug = !cfg.Ollama.Debug
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			// Update client debug flag
			client.Debug = cfg.Ollama.Debug
			// Show confirmation
			status := "OFF"
			if cfg.Ollama.Debug {
				status = "ON"
			}
			fmt.Printf("\n\033[1;32m✓ Debug mode is now %s\033[0m\n\n", status)
			// Reload config to refresh menu
			newCfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("error reloading config: %w", err)
			}
			cfg = newCfg
		}
	}
}

func initialModelWithSession(cfg *config.Config, sess *session.Session) menuModel {
	// Create Ollama client
	client := ollama.NewClient(cfg.Ollama.Host, cfg.Ollama.Model)
	client.Debug = cfg.Ollama.Debug

	return menuModel{
		choices: []menuItem{
			{name: "Plan", description: "Create development plans and break down tasks", isMode: true, mode: &modes.PlanMode{}},
			{name: "Edit", description: "Get help editing code with suggestions and diffs", isMode: true, mode: &modes.EditMode{}},
			{name: "Agent", description: "Autonomous multi-step task execution and problem solving", isMode: true, mode: &modes.AgentMode{}},
			{name: "CMD", description: "Get help with commands - generates but never executes", isMode: true, mode: &modes.CmdMode{}},
			{name: "Configure Models", description: "Assign different models to different modes", isMode: false},
			{name: "Settings", description: "Toggle debug mode and other settings", isMode: false},
		},
		cursor:   0,
		selected: false,
		cfg:      cfg,
		client:   client,
		session:  sess,
	}
}
