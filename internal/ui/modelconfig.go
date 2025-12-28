package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/ollama"
)

type modelConfigModel struct {
	client        *ollama.Client
	cfg           *config.Config
	availableModels []ollama.Model
	currentMode   string
	modes         []string
	cursor        int
	modelCursor   int
	state         string // "select_mode" or "select_model"
	err           error
}

func newModelConfigModel(client *ollama.Client, cfg *config.Config) modelConfigModel {
	return modelConfigModel{
		client: client,
		cfg:    cfg,
		modes:  []string{"plan", "edit", "agent", "cmd", "ask"},
		cursor: 0,
		state:  "select_mode",
	}
}

func (m modelConfigModel) Init() tea.Cmd {
	return func() tea.Msg {
		models, err := m.client.ListModels()
		if err != nil {
			return errMsg{err}
		}
		return modelsLoadedMsg{models}
	}
}

type modelsLoadedMsg struct {
	models []ollama.Model
}

type errMsg struct {
	err error
}

func (m modelConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case modelsLoadedMsg:
		m.availableModels = msg.models
		return m, nil
		
	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc", "left", "h":
			if m.state == "select_model" {
				m.state = "select_mode"
				m.modelCursor = 0
			} else {
				return m, tea.Quit
			}

		case "up", "k":
			if m.state == "select_mode" {
				if m.cursor > 0 {
					m.cursor--
				}
			} else {
				if m.modelCursor > 0 {
					m.modelCursor--
				}
			}

		case "down", "j":
			if m.state == "select_mode" {
				if m.cursor < len(m.modes)-1 {
					m.cursor++
				}
			} else {
				if m.modelCursor < len(m.availableModels)-1 {
					m.modelCursor++
				}
			}

		case "enter":
			if m.state == "select_mode" {
				m.currentMode = m.modes[m.cursor]
				m.state = "select_model"
				m.modelCursor = 0
			} else {
				// Save selected model for current mode
				selectedModel := m.availableModels[m.modelCursor].Name
				switch m.currentMode {
				case "plan":
					m.cfg.Models.Plan = selectedModel
				case "edit":
					m.cfg.Models.Edit = selectedModel
				case "agent":
					m.cfg.Models.Agent = selectedModel
				case "cmd":
					m.cfg.Models.CMD = selectedModel
				}
				
				// Save config
				if err := m.cfg.Save(); err != nil {
					m.err = err
				}
				
				m.state = "select_mode"
			}
		}
	}

	return m, nil
}

func (m modelConfigModel) View() string {
	var s strings.Builder

	// Title - bold + magenta
	s.WriteString("\n\033[1;38;5;205m⚙️  Configure Models\033[0m\n\n")

	if m.err != nil {
		s.WriteString("\033[38;5;9mError: " + m.err.Error() + "\033[0m\n")
		return s.String()
	}

	if len(m.availableModels) == 0 {
		s.WriteString("\033[38;5;240mLoading models...\033[0m")
		return s.String()
	}

	if m.state == "select_mode" {
		s.WriteString("\033[38;5;240mSelect a mode to configure:\033[0m\n\n")

		for i, mode := range m.modes {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}

			currentModel := m.cfg.GetModelForMode(mode)
			if m.cursor == i {
				s.WriteString(cursor + "\033[1;38;5;170m" + strings.ToUpper(mode) + "\033[0m\n")
			} else {
				s.WriteString(cursor + strings.ToUpper(mode) + "\n")
			}
			s.WriteString("  \033[38;5;240mCurrent: " + currentModel + "\033[0m\n")
		}

		s.WriteString("\n")
		s.WriteString("\033[38;5;240mPress Enter to change, left/h to go back, q to quit\033[0m\n")
	} else {
		s.WriteString(fmt.Sprintf("\033[38;5;240mSelect model for \033[1;38;5;205m%s\033[0;38;5;240m mode:\033[0m\n\n", strings.ToUpper(m.currentMode)))

		for i, model := range m.availableModels {
			cursor := "  "
			if m.modelCursor == i {
				cursor = "> "
			}

			// Show size in human-readable format
			size := float64(model.Size) / (1024 * 1024 * 1024)
			sizeStr := fmt.Sprintf("%.1f GB", size)

			if m.modelCursor == i {
				s.WriteString(cursor + "\033[1;38;5;170m" + model.Name + "\033[0m\n")
			} else {
				s.WriteString(cursor + model.Name + "\n")
			}
			s.WriteString("  " + sizeStr + "\n")
		}

		s.WriteString("\n")
		s.WriteString("\033[38;5;240mPress Enter to select, left/h/Esc to go back\033[0m\n")
	}

	return s.String()
}

// RunModelConfig starts the model configuration UI
func RunModelConfig(client *ollama.Client, cfg *config.Config) error {
	p := tea.NewProgram(newModelConfigModel(client, cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
